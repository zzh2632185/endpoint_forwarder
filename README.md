# Claude Code Request Forwarder

A high-performance Go application that transparently forwards Claude Code API requests to multiple endpoints with intelligent routing, health checking, and automatic retry/fallback capabilities.

[中文文档](README_CN.md) | English

## Features

- **Transparent Proxying**: Forward all HTTP requests transparently to backend endpoints
- **SSE Streaming Support**: Full support for Server-Sent Events streaming
- **Token Management**: Override or add Authorization Bearer tokens per endpoint  
- **Routing Strategies**: Priority-based or fastest-response routing
- **Health Checking**: Automatic endpoint health monitoring
- **Retry & Fallback**: Exponential backoff with automatic endpoint fallback
- **Monitoring**: Built-in health checks and Prometheus-style metrics
- **Structured Logging**: Configurable JSON or text logging with multiple levels
- **TUI Interface**: Built-in Terminal User Interface for real-time monitoring (enabled by default)

## Quick Start

1. **Build the application**:
   ```bash
   go build -o endpoint_forwarder
   ```

2. **Copy and configure the example config**:
   ```bash
   cp config/example.yaml config/config.yaml
   # Edit config.yaml with your endpoints and tokens
   ```

3. **Run the forwarder**:
   ```bash
   # Default mode with TUI interface
   ./endpoint_forwarder -config config/config.yaml
   
   # Run without TUI (traditional console mode)
   ./endpoint_forwarder -config config/config.yaml --no-tui
   
   # Explicitly enable TUI (default behavior)
   ./endpoint_forwarder -config config/config.yaml --tui
   ```

4. **Configure Claude Code**:
   Set in Claude Code's `settings.json`:
   ```json
   {
     "ANTHROPIC_BASE_URL": "http://localhost:8080"
   }
   ```

## Configuration

### Server Configuration
```yaml
server:
  host: "0.0.0.0"  # Server bind address
  port: 8080        # Server port
```

### Routing Strategy
```yaml
strategy:
  type: "priority"  # "priority", "fastest", or "round-robin"
```

- **priority**: Use endpoints in priority order (lower number = higher priority)
- **fastest**: Use endpoint with lowest response time
- **round-robin**: Rotate through all healthy endpoints for load balancing

### Retry Configuration
```yaml
retry:
  max_attempts: 3      # Maximum retry attempts per endpoint
  base_delay: "1s"     # Initial delay between retries
  max_delay: "30s"     # Maximum delay cap
  multiplier: 2.0      # Exponential backoff multiplier
```

### Health Check Configuration
```yaml
health:
  check_interval: "30s"     # How often to check endpoint health
  timeout: "5s"             # Health check timeout
  health_path: "/v1/models" # Health check endpoint path
```

### Global Timeout Configuration
```yaml
global_timeout: "300s"      # Default timeout for all non-streaming requests (5 minutes)
```

**Usage:**
- Sets the default timeout for all endpoints that don't specify their own `timeout`
- Only applies to non-streaming requests
- Can be overridden by individual endpoint `timeout` settings

### Authentication Configuration
```yaml
auth:
  enabled: false                    # Enable Bearer token authentication (default: false)
  token: "your-bearer-token"        # Bearer token for authentication (required when enabled)
```

### TUI Interface Configuration
```yaml
tui:
  enabled: true                     # Enable TUI interface (default: true)
  update_interval: "1s"             # TUI refresh interval (default: 1s)
```

**TUI Features:**
- **Real-time Monitoring**: Live request metrics, response times, and success rates
- **Multi-tab Interface**: Overview, Endpoints, Connections, Logs, and Configuration tabs
- **Interactive Navigation**: Tab/Shift+Tab to switch tabs, 1-5 for direct access
- **Color-coded Status**: Green=Healthy, Yellow=Warning, Red=Error
- **Live Connection Tracking**: Monitor active connections and traffic
- **Real-time Logs**: Real-time System logs

**TUI Controls:**
- `Tab/Shift+Tab`: Navigate between tabs
- `1-5`: Jump directly to tab (1=Overview, 2=Endpoints, etc.)
- `Ctrl+C`: Quit application
- `Arrow Keys`: Navigate within views

**Usage:**
- When `enabled: false` (default): No authentication is required, requests pass through directly
- When `enabled: true`: All requests must include `Authorization: Bearer <token>` header
- The token in the header must exactly match the configured token
- Returns HTTP 401 Unauthorized for missing, malformed, or invalid tokens
- Only applies to the main proxy endpoints (health check endpoints remain open)

**Health Check Behavior:**
- **Endpoint**: Tests the `/v1/models` endpoint (suitable for Claude API)
- **Success Criteria**: Accepts both 2xx (success) and 4xx (client error) status codes
  - 2xx responses indicate the endpoint is working correctly
  - 4xx responses (401, 403, etc.) indicate the endpoint is reachable but may need proper authentication
- **Failure Criteria**: 5xx server errors indicate endpoint problems
- **Strategy Logging**: For "fastest" strategy, logs endpoint latencies before each selection

### Endpoint Configuration
```yaml
endpoints:
  - name: "primary"
    url: "https://api.anthropic.com"
    priority: 1
    timeout: "30s"
    token: "sk-ant-your-token-here"  # Optional: Override/add auth token
    headers:                         # Optional: Additional headers
      X-Custom-Header: "value"
```

#### Parameter Inheritance
For convenience, subsequent endpoints can inherit configuration from the first endpoint for any unspecified parameters:

**Inheritable Parameters:**
- `token`: Authentication token
- `timeout`: Request timeout duration (defaults to `global_timeout` if not specified)
- `headers`: HTTP headers (with smart merging)

```yaml
endpoints:
  - name: "primary"
    url: "https://api.anthropic.com"
    priority: 1
    timeout: "45s"                    # ⏱️ Will be inherited
    token: "sk-ant-your-main-token"   # 🔑 Will be inherited
    headers:                          # 📋 Will be inherited & merged
      Authorization-Fallback: "Bearer fallback"
      X-API-Version: "v1"
      User-Agent: "Claude-Forwarder/1.0"
    
  - name: "secondary"
    url: "https://backup.anthropic.com" 
    priority: 2
    # ✅ Inherits: timeout=45s, token=sk-ant-your-main-token
    headers:
      X-Custom-Header: "secondary"    # 🔄 Merged with inherited headers
      # Final headers: Authorization-Fallback, X-API-Version, User-Agent + X-Custom-Header
    
  - name: "custom"
    url: "https://custom.anthropic.com"
    priority: 3
    timeout: "30s"                    # 🚫 Overrides inheritance
    token: "sk-ant-different-token"   # 🚫 Overrides inheritance
    # ✅ Still inherits headers from primary
    
  - name: "minimal"
    url: "https://minimal.anthropic.com"
    priority: 4
    # ✅ Inherits ALL parameters from primary
```

**Header Merging Rules:**
- If no headers specified → inherit all headers from first endpoint
- If headers specified → merge with first endpoint's headers (your headers override)
- Headers with same key → your value takes precedence

### Proxy Configuration
```yaml
proxy:
  enabled: true
  type: "http"  # "http", "https", or "socks5"
  
  # Option 1: Complete proxy URL
  url: "http://proxy.example.com:8080"
  # url: "socks5://proxy.example.com:1080"
  
  # Option 2: Host and port (alternative to URL)
  host: "proxy.example.com"
  port: 8080
  
  # Optional authentication
  username: "proxy_user"
  password: "proxy_pass"
```

**Proxy Support:**
- **HTTP/HTTPS Proxy**: Standard HTTP proxy with optional authentication
- **SOCKS5 Proxy**: Full SOCKS5 support with optional authentication  
- **Flexible Configuration**: Use complete URL or separate host:port
- **Security**: Proxy credentials are handled securely
- **Performance**: Optimized transport layer for all proxy types

**Usage Notes:**
- All outbound requests (health checks, fast tests, and API calls) use the configured proxy
- Proxy settings apply globally to all endpoints
- For corporate environments, ensure proxy allows HTTPS CONNECT method
- SOCKS5 proxies provide better performance for high-throughput scenarios

## Monitoring Endpoints

The forwarder provides several monitoring endpoints:

- **GET /health**: Basic health check
- **GET /health/detailed**: Detailed health information for all endpoints  
- **GET /metrics**: Prometheus-style metrics

### Example Health Check Response
```json
{
  "status": "healthy",
  "healthy_endpoints": 2,
  "total_endpoints": 3
}
```

## Usage Examples

### Basic Request Forwarding
```bash
# Regular API request - will be forwarded to the best available endpoint
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-3-sonnet-20240229", "max_tokens": 100, "messages": [{"role": "user", "content": "Hello"}]}'
```

### SSE Streaming
```bash
# Streaming request - automatically detected and handled
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"model": "claude-3-sonnet-20240229", "max_tokens": 100, "messages": [{"role": "user", "content": "Count to 10"}], "stream": true}'
```

### Health Monitoring
```bash
# Check overall health
curl http://localhost:8080/health

# Get detailed endpoint status
curl http://localhost:8080/health/detailed

# Get Prometheus metrics
curl http://localhost:8080/metrics
```

## How It Works

1. **Request Reception**: The forwarder receives HTTP requests on the configured port
2. **Endpoint Selection**: Based on the configured strategy (priority/fastest), selects the best available healthy endpoint
3. **Request Forwarding**: Transparently forwards the request with proper header handling:
   - **Host Header**: Automatically set to match the target endpoint's hostname
   - **Authorization**: Override/inject tokens as configured, remove client tokens
   - **Security**: Automatically strips sensitive client headers (`X-API-Key`, `Authorization`)
   - **Custom Headers**: Add endpoint-specific headers as configured
   - **Original Headers**: Preserve all other headers from the original request
4. **Response Handling**: 
   - Regular requests: Buffers and forwards the complete response
   - SSE requests: Streams response in real-time with proper event handling
5. **Error Handling**: On failure, automatically retries with exponential backoff, then falls back to the next available endpoint
6. **Health Monitoring**: Continuously monitors endpoint health and adjusts routing accordingly

## Command Line Options

```bash
./endpoint_forwarder [OPTIONS]
```

Options:
- `-config path/to/config.yaml`: Path to configuration file (default: "config/example.yaml")
- `-version`: Show version information
- `-tui`: Enable TUI interface (default: true)
- `-no-tui`: Disable TUI interface (run in traditional console mode)

Examples:
```bash
# Default mode with TUI
./endpoint_forwarder -config my-config.yaml

# Run without TUI (traditional console logging)
./endpoint_forwarder -config my-config.yaml -no-tui

# Show version information
./endpoint_forwarder -version
```

## Logging

The application uses structured logging with enhanced formatting for better human readability:

```yaml
logging:
  level: "info"    # debug, info, warn, error
  format: "text"   # text (human-readable) or json (machine-readable)
```

### Log Features

**Enhanced Readability:**
- 🎯 Emoji indicators for different log types and statuses
- 📊 Formatted response times (μs/ms/s) and data sizes (B/KB/MB)  
- 🚀 Request lifecycle tracking with endpoint information
- ⏱️  Precise timestamp formatting (HH:MM:SS.mmm)

**Request Logging:**
- Request start with selected endpoint name
- Response completion with status indicators
- Error tracking with appropriate severity levels
- Performance monitoring (slow request detection)

**Log Examples:**
```
15:04:05.123 level=INFO msg="🚀 Request started" method=POST path=/v1/messages client_ip=192.168.1.100 user_agent="Claude-Client/1.0" content_length=245
15:04:05.456 level=INFO msg="🎯 Selected endpoint" endpoint=primary url=https://api.anthropic.com priority=1 attempt=1 total_endpoints=3  
15:04:06.789 level=INFO msg="✅ Request completed" method=POST path=/v1/messages endpoint=primary status_code=200 bytes_written=1.2KB duration=633.2ms client_ip=192.168.1.100
```

**Security Features:**
- Automatically removes sensitive client headers (`X-API-Key`, `Authorization`) 
- Replaces with endpoint-configured tokens
- Prevents credential leakage between client and backend

## Production Considerations

- Configure appropriate timeouts for your use case
- Monitor the `/health` and `/metrics` endpoints
- Use a reverse proxy (nginx/Apache) for SSL termination
- Configure log rotation for production deployments
- Set up alerts based on endpoint health metrics
- Consider rate limiting at the reverse proxy level

## License

This project is provided as-is for educational and development purposes.