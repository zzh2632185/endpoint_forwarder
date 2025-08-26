package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"endpoint_forwarder/config"
	"endpoint_forwarder/internal/endpoint"
)

// Context key for endpoint information
type contextKey string

const EndpointContextKey = contextKey("endpoint")

// Handler handles HTTP proxy requests
type Handler struct {
	endpointManager *endpoint.Manager
	config          *config.Config
	retryHandler    *RetryHandler
}

// NewHandler creates a new proxy handler
func NewHandler(endpointManager *endpoint.Manager, cfg *config.Config) *Handler {
	retryHandler := NewRetryHandler(cfg)
	retryHandler.SetEndpointManager(endpointManager)
	
	return &Handler{
		endpointManager: endpointManager,
		config:          cfg,
		retryHandler:    retryHandler,
	}
}

// ServeHTTP implements the http.Handler interface
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Create a context for this request
	ctx := r.Context()
	
	// Clone request body for potential retries
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}
		r.Body.Close()
	}

	// Check if this is an SSE request
	acceptHeader := r.Header.Get("Accept")
	isSSE := strings.Contains(acceptHeader, "text/event-stream") || 
			 strings.Contains(r.Header.Get("Cache-Control"), "no-cache")

	if isSSE {
		h.handleSSERequest(w, r, bodyBytes)
		return
	}

	// Handle regular HTTP request with retry logic
	h.handleRegularRequest(ctx, w, r, bodyBytes)
}

// handleRegularRequest handles non-streaming requests
func (h *Handler) handleRegularRequest(ctx context.Context, w http.ResponseWriter, r *http.Request, bodyBytes []byte) {
	var lastErr error
	var selectedEndpointName string
	
	operation := func(ep *endpoint.Endpoint) error {
		// Store the selected endpoint name for logging
		selectedEndpointName = ep.Config.Name
		
		// Create request to target endpoint
		targetURL := ep.Config.URL + r.URL.Path
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Copy headers from original request
		h.copyHeaders(r, req, ep)

		// Create HTTP client with timeout
		client := &http.Client{
			Timeout: ep.Config.Timeout,
		}

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Set status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to copy response body: %w", err)
		}

		return nil
	}

	// Execute with retry logic
	lastErr = h.retryHandler.Execute(operation)
	
	// Store selected endpoint info in request context for logging
	if selectedEndpointName != "" {
		*r = *r.WithContext(context.WithValue(r.Context(), "selected_endpoint", selectedEndpointName))
	}
	
	if lastErr != nil {
		// If all retries failed, return error
		http.Error(w, "All endpoints failed: "+lastErr.Error(), http.StatusBadGateway)
	}
}

// copyHeaders copies headers from source to destination request
func (h *Handler) copyHeaders(src *http.Request, dst *http.Request, ep *endpoint.Endpoint) {
	// List of headers to skip/remove
	skipHeaders := map[string]bool{
		"host":          true, // We'll set this based on target endpoint
		"authorization": true, // We'll add our own if configured
		"x-api-key":     true, // Remove sensitive client API keys
	}
	
	// Copy all headers except those we want to skip
	for key, values := range src.Header {
		if skipHeaders[strings.ToLower(key)] {
			continue
		}
		
		for _, value := range values {
			dst.Header.Add(key, value)
		}
	}

	// Set Host header based on target endpoint URL
	if u, err := url.Parse(ep.Config.URL); err == nil {
		dst.Header.Set("Host", u.Host)
		// Also set the Host field directly on the request for proper HTTP/1.1 behavior
		dst.Host = u.Host
	}

	// Add or override Authorization header if token is configured
	if ep.Config.Token != "" {
		dst.Header.Set("Authorization", "Bearer "+ep.Config.Token)
	}

	// Add custom headers from endpoint configuration
	for key, value := range ep.Config.Headers {
		dst.Header.Set(key, value)
	}

	// Remove hop-by-hop headers
	hopByHopHeaders := []string{
		"Connection",
		"Keep-Alive", 
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range hopByHopHeaders {
		dst.Header.Del(header)
	}
}