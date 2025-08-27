package proxy

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	
	"endpoint_forwarder/internal/monitor"
)

// UsageData represents the usage field in Claude API SSE events
type UsageData struct {
	InputTokens            int64 `json:"input_tokens"`
	OutputTokens           int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens    int64 `json:"cache_read_input_tokens"`
}

// MessageDelta represents the structure of message_delta events
type MessageDelta struct {
	Type      string     `json:"type"`
	Delta     interface{} `json:"delta"`
	Usage     *UsageData  `json:"usage,omitempty"`
}

// TokenParser handles parsing of SSE events for token usage extraction
type TokenParser struct {
	// Buffer to collect multi-line JSON data
	eventBuffer     strings.Builder
	currentEvent    string
	collectingData  bool
}

// NewTokenParser creates a new token parser instance
func NewTokenParser() *TokenParser {
	return &TokenParser{}
}

// ParseSSELine processes a single line from SSE stream and extracts token usage if found
func (tp *TokenParser) ParseSSELine(line string) *monitor.TokenUsage {
	line = strings.TrimSpace(line)
	
	
	// Handle event type lines
	if strings.HasPrefix(line, "event: ") {
		eventType := strings.TrimPrefix(line, "event: ")
		tp.currentEvent = eventType
		tp.collectingData = (eventType == "message_delta")
		tp.eventBuffer.Reset()
		return nil
	}
	
	// Handle data lines for message_delta events
	if strings.HasPrefix(line, "data: ") && tp.collectingData {
		dataContent := strings.TrimPrefix(line, "data: ")
		tp.eventBuffer.WriteString(dataContent)
		return nil
	}
	
	// Handle empty lines that signal end of SSE event
	if line == "" && tp.collectingData && tp.eventBuffer.Len() > 0 {
		return tp.parseMessageDelta()
	}
	
	return nil
}

// parseMessageDelta parses the collected message_delta JSON data
func (tp *TokenParser) parseMessageDelta() *monitor.TokenUsage {
	defer func() {
		tp.eventBuffer.Reset()
		tp.collectingData = false
		tp.currentEvent = ""
	}()
	
	jsonData := tp.eventBuffer.String()
	if jsonData == "" {
		return nil
	}
	
	// Parse the JSON data
	var messageDelta MessageDelta
	if err := json.Unmarshal([]byte(jsonData), &messageDelta); err != nil {
		return nil
	}
	
	// Check if this message_delta contains usage information
	if messageDelta.Usage == nil {
		return nil
	}
	
	// Convert to our TokenUsage format
	tokenUsage := &monitor.TokenUsage{
		InputTokens:            messageDelta.Usage.InputTokens,
		OutputTokens:           messageDelta.Usage.OutputTokens,
		CacheCreationTokens:    messageDelta.Usage.CacheCreationInputTokens,
		CacheReadTokens:        messageDelta.Usage.CacheReadInputTokens,
	}

	slog.Info(fmt.Sprintf("🪙 [Token Parser] Extracted token usage from SSE stream",
		"input_tokens", tokenUsage.InputTokens,
		"output_tokens", tokenUsage.OutputTokens,
		"cache_creation_tokens", tokenUsage.CacheCreationTokens,
		"cache_read_tokens", tokenUsage.CacheReadTokens))

	return tokenUsage
}

// Reset clears the parser state
func (tp *TokenParser) Reset() {
	tp.eventBuffer.Reset()
	tp.currentEvent = ""
	tp.collectingData = false
}