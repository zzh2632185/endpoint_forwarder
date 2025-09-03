# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Claude Request Forwarder is a high-performance Go application that transparently forwards Claude API requests to multiple endpoints with intelligent routing, health checking, and automatic retry/fallback capabilities. It includes a built-in Terminal User Interface (TUI) for real-time monitoring.

## Build and Development Commands

```bash
# Build the application
go build -o endpoint_forwarder

# Run with default configuration and TUI
./endpoint_forwarder -config config/config.yaml

# Run without TUI (console mode)
./endpoint_forwarder -config config/config.yaml --no-tui

# Run tests
go test ./...

# Test specific packages
go test ./internal/endpoint
go test ./internal/proxy
go test ./internal/middleware

# Check version
./endpoint_forwarder -version
```

## Architecture Overview

### Core Components

- **`main.go`**: Application entry point with TUI/console mode switching, graceful shutdown, and configuration management
- **`config/`**: Configuration management with hot-reloading via fsnotify
- **`internal/endpoint/`**: Endpoint management, health checking, and fast testing
- **`internal/proxy/`**: HTTP request forwarding, streaming support, and retry logic
- **`internal/middleware/`**: Authentication, logging, and monitoring middleware
- **`internal/tui/`**: Terminal User Interface using rivo/tview
- **`internal/transport/`**: HTTP/HTTPS/SOCKS5 proxy transport configuration

### Key Design Patterns

**Strategy Pattern**: Endpoint selection via "priority" or "fastest" strategies with optional pre-request fast testing

**Middleware Chain**: Request processing through authentication, logging, and monitoring layers

**Observer Pattern**: Configuration hot-reloading with callback-based component updates

**Circuit Breaker Pattern**: Health checking with automatic endpoint marking as healthy/unhealthy

### Request Flow

1. Request reception with middleware chain (auth → logging → monitoring)
2. Endpoint selection based on strategy and health status
3. Header transformation (strip client auth, inject endpoint tokens and API keys)
4. Request forwarding with timeout and retry handling
5. Response streaming (SSE) or buffered response handling
6. Error handling with automatic endpoint fallback

## Configuration

- **Primary config**: `config/config.yaml` (copy from `config/example.yaml`)
- **Hot-reloading**: Automatic configuration reload via fsnotify with 500ms debounce
- **Dynamic Token Resolution**: Tokens and API keys are resolved dynamically at runtime for group-based failover
- **Global timeout**: Default timeout for all non-streaming requests (5 minutes)
- **API Key support**: Endpoints can specify `api-key` field which is automatically passed as `X-Api-Key` header

### Group Management

The system supports endpoint grouping with automatic failover and cooldown mechanisms:

**Group Configuration**:
- Each endpoint can belong to a group using the `group` field
- Groups have priorities defined by `group-priority` (lower number = higher priority)
- Only one group is active at a time based on priority and cooldown status
- Groups inherit settings from the first endpoint in their group

**Group Behavior**:
- **Active Group**: The highest priority group not in cooldown
- **Cooldown**: When all endpoints in a group fail, the group enters cooldown mode
- **Automatic Failover**: System automatically switches to next priority group when active group fails
- **Cooldown Duration**: Configurable via `group.cooldown` (default: 10 minutes)
- **Retry Limit**: Groups have a maximum retry count via `group.max_retries` (default: 3). When exceeded, the group enters cooldown
- **Retry Tracking**: System tracks retry counts per group and resets on successful requests

**Group Inheritance Rules**:
- Endpoints inherit `group` and `group-priority` from previous endpoints if not specified
- Static inheritance: `timeout` and `headers` are inherited during configuration parsing
- Dynamic resolution: `token` and `api-key` are resolved at runtime from the first endpoint in the same group
- Groups can be mixed and matched with different priorities for failover scenarios

**Dynamic Token Resolution**:
- **Runtime Resolution**: Tokens and API keys are not inherited during config parsing but resolved dynamically at request time
- **Group-based Sharing**: All endpoints in a group share the token/api-key from the first endpoint that defines it
- **Override Support**: Individual endpoints can override group tokens by explicitly specifying their own `token` or `api-key`
- **Failover-friendly**: When groups switch during failover, the new active group's tokens are automatically used

**Example Group Configuration**:
```yaml
endpoints:
  # Primary group (highest priority) - defines group token
  - name: "primary"
    url: "https://api.openai.com"
    group: "main"
    group-priority: 1
    priority: 1
    token: "sk-main-group-token"        # 🔑 Shared by all main group endpoints
    
  # Backup for primary group - uses main group token
  - name: "primary_backup"
    url: "https://api.anthropic.com"
    priority: 2
    # 🔄 Inherits group: "main" and group-priority: 1
    # 🔑 Uses main group token dynamically at runtime
    
  # Secondary group (lower priority) - defines different group token
  - name: "secondary"
    url: "https://api.example.com"
    group: "backup"
    group-priority: 2
    priority: 1
    token: "sk-backup-group-token"      # 🔑 Shared by all backup group endpoints
    
  # Custom override within backup group
  - name: "secondary_special"
    url: "https://api.special.com"
    priority: 2
    token: "sk-special-override"        # 🔑 Overrides backup group token
    # 🔄 Still inherits group: "backup" and group-priority: 2
```

## Testing Approach

The codebase includes comprehensive unit tests:
- `*_test.go` files in each package
- Test configuration in `test_config.yaml`
- Health check testing with mock endpoints
- Fast tester functionality testing
- Proxy handler testing with various scenarios

## Key Features to Understand

**TUI Interface**: Real-time monitoring with tabs for Overview, Endpoints, Connections, Logs, and Configuration

**Streaming Support**: Automatic SSE detection and real-time streaming with proper event handling

**Proxy Support**: HTTP/HTTPS/SOCKS5 proxy configuration for all outbound requests

**Security**: Bearer token authentication with automatic header stripping and token injection. API key support with X-Api-Key header injection.

**Health Monitoring**: Continuous endpoint health checking with `/v1/models` endpoint testing

**Group Management**: Intelligent endpoint grouping with automatic failover, cooldown periods, and priority-based routing for high availability scenarios