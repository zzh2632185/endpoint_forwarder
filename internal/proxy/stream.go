package proxy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"endpoint_forwarder/internal/endpoint"
	"endpoint_forwarder/internal/transport"
)

// handleSSERequest handles Server-Sent Events streaming requests
func (h *Handler) handleSSERequest(w http.ResponseWriter, r *http.Request, bodyBytes []byte) {
	// Set SSE headers immediately
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Enable flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Get healthy endpoints with fast testing if enabled
	ctx := r.Context()
	var endpoints []*endpoint.Endpoint
	if h.endpointManager.GetConfig().Strategy.Type == "fastest" && h.endpointManager.GetConfig().Strategy.FastTestEnabled {
		endpoints = h.endpointManager.GetFastestEndpointsWithRealTimeTest(ctx)
	} else {
		endpoints = h.endpointManager.GetHealthyEndpoints()
	}
	
	if len(endpoints) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		h.writeSSEError(w, "No healthy endpoints available", flusher)
		return
	}

	slog.InfoContext(ctx, fmt.Sprintf("🌊 [SSE 流式传输] 开始建立连接 - 客户端: %s, 路径: %s", 
		r.RemoteAddr, r.URL.Path))
	slog.InfoContext(ctx, fmt.Sprintf("🎯 [SSE 流式传输] 选择端点: %s (共%d个可用)", 
		endpoints[0].Config.Name, len(endpoints)))

	// Try endpoints in order until one succeeds
	for i, ep := range endpoints {
		err := h.streamFromEndpoint(ctx, w, r, ep, bodyBytes, flusher)
		if err == nil {
			// Success
			return
		}

		slog.ErrorContext(ctx, fmt.Sprintf("❌ [SSE 流式传输] 端点连接失败: %s - 错误: %s", ep.Config.Name, err.Error()))

		// If this isn't the last endpoint, try the next one
		if i < len(endpoints)-1 {
			h.writeSSEEvent(w, "retry", fmt.Sprintf("🔄 切换到备用端点: %s", endpoints[i+1].Config.Name), flusher)
			continue
		}

		// All endpoints failed
		h.writeSSEError(w, fmt.Sprintf("💥 所有端点连接失败，最后错误: %v", err), flusher)
		return
	}
}

// streamFromEndpoint streams response from a specific endpoint
func (h *Handler) streamFromEndpoint(ctx context.Context, w http.ResponseWriter, r *http.Request, ep *endpoint.Endpoint, bodyBytes []byte, flusher http.Flusher) error {
	// Create request to target endpoint
	targetURL := ep.Config.URL + r.URL.Path
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create a context without timeout for streaming requests
	streamCtx := context.WithoutCancel(ctx)
	req, err := http.NewRequestWithContext(streamCtx, r.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers
	h.copyHeaders(r, req, ep)

	// Create HTTP client optimized for real-time streaming with proxy support
	httpTransport, err := transport.CreateTransport(h.config)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}
	
	// Optimize transport for streaming
	httpTransport.DisableKeepAlives = false
	httpTransport.MaxIdleConns = 10
	httpTransport.MaxIdleConnsPerHost = 2
	httpTransport.IdleConnTimeout = 0 // No idle timeout
	httpTransport.TLSHandshakeTimeout = 10 * time.Second
	httpTransport.ExpectContinueTimeout = 1 * time.Second
	httpTransport.ResponseHeaderTimeout = 15 * time.Second // Reduced for faster response
	// Critical: Disable compression to prevent buffering delays
	httpTransport.DisableCompression = true
	// Set smaller buffer sizes for lower latency
	httpTransport.WriteBufferSize = 4096 // Smaller write buffer
	httpTransport.ReadBufferSize = 4096  // Smaller read buffer
	
	client := &http.Client{
		Timeout:   0, // No timeout for streaming
		Transport: httpTransport,
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if response is successful
	if resp.StatusCode >= 400 {
		return fmt.Errorf("endpoint returned error: %d", resp.StatusCode)
	}

	// Start streaming the response - use byte-level streaming for maximum responsiveness
	return h.streamResponseByBytes(ctx, w, resp, flusher)
}

// streamResponse streams the HTTP response to the client
func (h *Handler) streamResponse(ctx context.Context, w http.ResponseWriter, resp *http.Response, flusher http.Flusher) error {
	slog.InfoContext(ctx, "📡 Starting real-time stream forwarding",
		"content_type", resp.Header.Get("Content-Type"),
		"status_code", resp.StatusCode)

	// Copy response headers first
	for key, values := range resp.Header {
		// Skip hop-by-hop headers and headers we set manually
		if key == "Connection" || key == "Transfer-Encoding" || key == "Content-Length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Ensure SSE headers are set
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Write status code
	w.WriteHeader(resp.StatusCode)
	flusher.Flush()

	// Use a smaller buffer for more responsive streaming
	scanner := bufio.NewScanner(resp.Body)
	// Use a smaller buffer size for lower latency
	buf := make([]byte, 4096) // Reduced from 64KB to 4KB
	scanner.Buffer(buf, 4096)

	// Create a ticker for heartbeat to keep connection alive
	heartbeatTicker := time.NewTicker(h.config.Streaming.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	lastActivity := time.Now()
	lineCount := 0
	
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "🚫 Stream cancelled by client",
				"lines_sent", lineCount,
				"duration", time.Since(lastActivity))
			return ctx.Err()
		case <-heartbeatTicker.C:
			// Send heartbeat if no activity for configured max idle time
			if time.Since(lastActivity) >= h.config.Streaming.MaxIdleTime {
				// Send SSE heartbeat comment (ignored by clients)
				fmt.Fprintf(w, ": heartbeat %s\n\n", time.Now().Format(time.RFC3339))
				flusher.Flush()
			}
		default:
			// Use a very short read timeout for responsiveness
			if conn, ok := resp.Body.(interface{ SetReadDeadline(time.Time) error }); ok {
				// Use a shorter timeout for better responsiveness
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			}

			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					// Check if it's a timeout error, which is expected
					if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
						// Timeout is expected, just continue the loop
						continue
					}
					slog.ErrorContext(ctx, "❌ Stream read error",
						"error", err.Error(),
						"lines_sent", lineCount)
					return fmt.Errorf("error reading response: %w", err)
				}
				// End of stream
				slog.InfoContext(ctx, "✅ Stream completed normally",
					"lines_sent", lineCount,
					"duration", time.Since(lastActivity))
				return nil
			}

			line := scanner.Text()
			lastActivity = time.Now()
			lineCount++
			
			// Write the line to the client immediately
			_, err := fmt.Fprintf(w, "%s\n", line)
			if err != nil {
				slog.ErrorContext(ctx, "❌ Error writing to client",
					"error", err.Error(),
					"lines_sent", lineCount)
				return fmt.Errorf("error writing to client: %w", err)
			}

			// CRITICAL: Flush immediately after each line for real-time streaming
			flusher.Flush()

			// Log progress every 10 lines (for debugging)
			if lineCount%10 == 0 {
				slog.DebugContext(ctx, "📊 Stream progress",
					"lines_sent", lineCount,
					"last_line_length", len(line))
			}
		}
	}
}

// writeSSEEvent writes a Server-Sent Event to the client
func (h *Handler) writeSSEEvent(w http.ResponseWriter, eventType, data string, flusher http.Flusher) {
	if eventType != "" {
		fmt.Fprintf(w, "event: %s\n", eventType)
	}
	
	// Handle multiline data
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		fmt.Fprintf(w, "data: %s\n", line)
	}
	
	fmt.Fprintf(w, "\n")
	flusher.Flush()
}

// writeSSEError writes an error event to the client
func (h *Handler) writeSSEError(w http.ResponseWriter, message string, flusher http.Flusher) {
	h.writeSSEEvent(w, "error", message, flusher)
}

// streamResponseByBytes streams the HTTP response byte-by-byte for maximum real-time performance
func (h *Handler) streamResponseByBytes(ctx context.Context, w http.ResponseWriter, resp *http.Response, flusher http.Flusher) error {
	slog.InfoContext(ctx, fmt.Sprintf("🚀 [实时流传输] 开始字节级转发 - 状态码: %d, 内容类型: %s", 
		resp.StatusCode, resp.Header.Get("Content-Type")))

	// Copy response headers first
	for key, values := range resp.Header {
		// Skip hop-by-hop headers and headers we set manually
		if key == "Connection" || key == "Transfer-Encoding" || key == "Content-Length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Ensure SSE headers are set
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Write status code
	w.WriteHeader(resp.StatusCode)
	flusher.Flush()

	// Create heartbeat ticker
	heartbeatTicker := time.NewTicker(h.config.Streaming.HeartbeatInterval)
	defer heartbeatTicker.Stop()

	lastActivity := time.Now()
	bytesTransferred := int64(0)
	lineBuffer := make([]byte, 0, 1024)

	// Create a small buffer for reading bytes
	buffer := make([]byte, 1024)

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, fmt.Sprintf("🚫 [实时流传输] 客户端断开连接 - 已传输: %d字节, 耗时: %v", 
				bytesTransferred, time.Since(lastActivity)))
			return ctx.Err()
			
		case <-heartbeatTicker.C:
			// Send heartbeat if no activity for configured max idle time
			if time.Since(lastActivity) >= h.config.Streaming.MaxIdleTime {
				fmt.Fprintf(w, ": heartbeat %s\n\n", time.Now().Format(time.RFC3339))
				flusher.Flush()
			}

		default:
			// Set a very short read deadline for responsiveness
			if conn, ok := resp.Body.(interface{ SetReadDeadline(time.Time) error }); ok {
				conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			}

			n, err := resp.Body.Read(buffer)
			if n > 0 {
				lastActivity = time.Now()
				bytesTransferred += int64(n)

				// Process each byte to detect line endings and flush immediately
				for i := 0; i < n; i++ {
					b := buffer[i]
					lineBuffer = append(lineBuffer, b)

					// If we hit a newline or the buffer is getting large, flush
					if b == '\n' || len(lineBuffer) >= 512 {
						_, writeErr := w.Write(lineBuffer)
						if writeErr != nil {
						slog.ErrorContext(ctx, fmt.Sprintf("❌ [实时流传输] 写入客户端失败 - 错误: %s, 已传输: %d字节", 
							writeErr.Error(), bytesTransferred))
							return fmt.Errorf("error writing to client: %w", writeErr)
						}
						
						// CRITICAL: Flush after every line for real-time streaming
						flusher.Flush()
						
						// Reset the line buffer
						lineBuffer = lineBuffer[:0]
					}
				}

				// Log progress periodically
				if bytesTransferred%10240 == 0 { // Every 10KB
					slog.DebugContext(ctx, fmt.Sprintf("📈 [流传输进度] 已传输: %d字节, 缓冲区: %d字节", 
						bytesTransferred, len(lineBuffer)))
				}
			}

			if err != nil {
				// Handle different types of errors
				if netErr, ok := err.(interface{ Timeout() bool }); ok && netErr.Timeout() {
					// Timeout is expected due to our short deadline, continue
					continue
				}
				
				// Check for EOF (end of stream)
				if err.Error() == "EOF" {
					// Flush any remaining data in the line buffer
					if len(lineBuffer) > 0 {
						w.Write(lineBuffer)
						flusher.Flush()
					}
					
					slog.InfoContext(ctx, fmt.Sprintf("✅ [实时流传输] 传输完成 - 总计: %d字节, 耗时: %v", 
						bytesTransferred, time.Since(lastActivity)))
					return nil
				}
				
				slog.ErrorContext(ctx, fmt.Sprintf("❌ [实时流传输] 读取错误 - 错误: %s, 已传输: %d字节", 
					err.Error(), bytesTransferred))
				return fmt.Errorf("error reading response: %w", err)
			}
		}
	}
}