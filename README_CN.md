# Claude Code 请求转发器

一个高性能的 Go 应用程序，透明地将 Claude Code API 请求转发到多个端点，支持智能路由、健康检查和自动重试/故障转移功能。

[English](README.md) | 中文

## 功能特性

- **透明代理**: 透明地将所有 HTTP 请求转发到后端端点
- **SSE 流式支持**: 完整支持服务器发送事件（Server-Sent Events）流式传输
- **令牌管理**: 为每个端点覆盖或添加授权Bearer令牌
- **路由策略**: 基于优先级或最快响应的路由选择
- **健康检查**: 自动端点健康监控
- **重试和故障转移**: 指数退避算法和自动端点故障转移
- **监控功能**: 内置健康检查和Prometheus风格的指标
- **结构化日志**: 可配置的JSON或文本日志记录，支持多个级别
- **TUI界面**: 内置终端用户界面，实时监控（默认启用）

## 快速开始

1. **构建应用程序**:
   ```bash
   go build -o endpoint_forwarder
   ```

2. **复制并配置示例配置**:
   ```bash
   cp config/example.yaml config/config.yaml
   # 编辑 config.yaml 文件，添加您的端点和令牌
   ```

3. **运行转发器**:
   ```bash
   # 默认模式，启用 TUI 界面
   ./endpoint_forwarder -config config/config.yaml
   
   # 不使用 TUI 运行（传统控制台模式）
   ./endpoint_forwarder -config config/config.yaml --no-tui
   
   # 显式启用 TUI（默认行为）
   ./endpoint_forwarder -config config/config.yaml --tui
   ```

4. **在 Claude Code 中配置**:
   在 Claude Code 的 `settings.json` 中设置：
   ```json
   {
     "ANTHROPIC_BASE_URL": "http://localhost:8080"
   }
   ```

## 配置说明

### 服务器配置
```yaml
server:
  host: "0.0.0.0"  # 服务器绑定地址
  port: 8080        # 服务器端口
```

### 路由策略
```yaml
strategy:
  type: "priority"  # "priority"、"fastest" 或 "round-robin"
```

- **priority**: 按优先级顺序使用端点（数字越小优先级越高）
- **fastest**: 使用响应时间最短的端点
- **round-robin**: 轮询使用所有健康端点，实现负载均衡

### 重试配置
```yaml
retry:
  max_attempts: 3      # 每个端点最大重试次数
  base_delay: "1s"     # 重试之间的初始延迟
  max_delay: "30s"     # 最大延迟上限
  multiplier: 2.0      # 指数退避乘数
```

### 健康检查配置
```yaml
health:
  check_interval: "30s"     # 检查端点健康的频率
  timeout: "5s"             # 健康检查超时时间
  health_path: "/v1/models" # 健康检查端点路径
```

### 全局超时配置
```yaml
global_timeout: "300s"      # 所有非流式请求的默认超时时间（5分钟）
```

**用法说明:**
- 为未指定 `timeout` 的端点设置默认超时时间
- 仅适用于非流式请求
- 可通过各个端点的 `timeout` 设置进行覆盖

### 身份验证配置
```yaml
auth:
  enabled: false                    # 启用 Bearer 令牌身份验证（默认: false）
  token: "your-bearer-token"        # 身份验证的 Bearer 令牌（启用时必需）
```

### TUI 界面配置
```yaml
tui:
  enabled: true                     # 启用 TUI 界面（默认: true）
  update_interval: "1s"             # TUI 刷新间隔（默认: 1s）
```

**TUI 功能特性:**
- **实时监控**: 实时请求指标、响应时间和成功率
- **多标签界面**: 概览、端点、连接、日志和配置标签
- **交互式导航**: Tab/Shift+Tab 切换标签，1-5 直接访问
- **彩色状态编码**: 绿色=健康，黄色=警告，红色=错误
- **实时连接跟踪**: 监控活跃连接和流量
- **实时日志**: 显示实时的系统日志

**TUI 控制:**
- `Tab/Shift+Tab`: 在标签之间导航
- `1-5`: 直接跳转到标签（1=概览，2=端点等）
- `Ctrl+C`: 退出应用程序
- `方向键`: 在视图内导航

**用法说明:**
- 当 `enabled: false`（默认）时：不需要身份验证，请求直接通过
- 当 `enabled: true` 时：所有请求必须包含 `Authorization: Bearer <token>` 头部
- 头部中的令牌必须与配置的令牌完全匹配
- 对于缺失、格式错误或无效的令牌，返回HTTP 401未授权
- 仅适用于主要代理端点（健康检查端点保持开放）

**健康检查行为:**
- **端点**: 测试 `/v1/models` 端点（适用于 Claude API）
- **成功标准**: 接受 2xx（成功）和 4xx（客户端错误）状态码
  - 2xx 响应表示端点正常工作
  - 4xx 响应（401、403等）表示端点可达但可能需要适当的身份验证
- **失败标准**: 5xx 服务器错误表示端点有问题
- **策略日志**: 对于"fastest"策略，在每次选择之前记录端点延迟

### 端点配置
```yaml
endpoints:
  - name: "primary"
    url: "https://api.anthropic.com"
    priority: 1
    timeout: "30s"
    token: "sk-ant-your-token-here"  # 可选：覆盖/添加认证令牌
    headers:                         # 可选：附加头部
      X-Custom-Header: "value"
```

#### 参数继承
为了方便配置，后续端点可以从第一个端点继承未指定的配置参数：

**可继承参数:**
- `token`: 认证令牌
- `timeout`: 请求超时持续时间（如未指定则默认为 `global_timeout`）
- `headers`: HTTP 头部（智能合并）

```yaml
endpoints:
  - name: "primary"
    url: "https://api.anthropic.com"
    priority: 1
    timeout: "45s"                    # ⏱️ 将被继承
    token: "sk-ant-your-main-token"   # 🔑 将被继承
    headers:                          # 📋 将被继承并合并
      Authorization-Fallback: "Bearer fallback"
      X-API-Version: "v1"
      User-Agent: "Claude-Forwarder/1.0"
    
  - name: "secondary"
    url: "https://backup.anthropic.com" 
    priority: 2
    # ✅ 继承：timeout=45s, token=sk-ant-your-main-token
    headers:
      X-Custom-Header: "secondary"    # 🔄 与继承的头部合并
      # 最终头部：Authorization-Fallback, X-API-Version, User-Agent + X-Custom-Header
    
  - name: "custom"
    url: "https://custom.anthropic.com"
    priority: 3
    timeout: "30s"                    # 🚫 覆盖继承
    token: "sk-ant-different-token"   # 🚫 覆盖继承
    # ✅ 仍然从 primary 继承头部
    
  - name: "minimal"
    url: "https://minimal.anthropic.com"
    priority: 4
    # ✅ 从 primary 继承所有参数
```

**头部合并规则:**
- 如果未指定头部 → 继承第一个端点的所有头部
- 如果指定了头部 → 与第一个端点的头部合并（您的头部优先）
- 同名头部 → 您的值优先

### 代理配置
```yaml
proxy:
  enabled: true
  type: "http"  # "http", "https", 或 "socks5"
  
  # 选项1：完整代理 URL
  url: "http://proxy.example.com:8080"
  # url: "socks5://proxy.example.com:1080"
  
  # 选项2：主机和端口（作为 URL 的替代）
  host: "proxy.example.com"
  port: 8080
  
  # 可选身份验证
  username: "proxy_user"
  password: "proxy_pass"
```

**代理支持:**
- **HTTP/HTTPS 代理**: 标准 HTTP 代理，支持可选身份验证
- **SOCKS5 代理**: 完整的 SOCKS5 支持，支持可选身份验证
- **灵活配置**: 使用完整 URL 或单独的 host:port
- **安全性**: 代理凭据得到安全处理
- **性能**: 为所有代理类型优化的传输层

**使用说明:**
- 所有出站请求（健康检查、快速测试和 API 调用）使用配置的代理
- 代理设置全局应用于所有端点
- 对于企业环境，请确保代理允许 HTTPS CONNECT 方法
- SOCKS5 代理为高吞吐量场景提供更好的性能

## 监控端点

转发器提供几个监控端点：

- **GET /health**: 基本健康检查
- **GET /health/detailed**: 所有端点的详细健康信息
- **GET /metrics**: Prometheus 风格的指标

### 示例健康检查响应
```json
{
  "status": "healthy",
  "healthy_endpoints": 2,
  "total_endpoints": 3
}
```

## 使用示例

### 基本请求转发
```bash
# 常规 API 请求 - 将转发到最佳可用端点
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{"model": "claude-3-sonnet-20240229", "max_tokens": 100, "messages": [{"role": "user", "content": "你好"}]}'
```

### SSE 流式传输
```bash
# 流式请求 - 自动检测和处理
curl -X POST http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"model": "claude-3-sonnet-20240229", "max_tokens": 100, "messages": [{"role": "user", "content": "从1数到10"}], "stream": true}'
```

### 健康监控
```bash
# 检查整体健康状况
curl http://localhost:8080/health

# 获取详细的端点状态
curl http://localhost:8080/health/detailed

# 获取 Prometheus 指标
curl http://localhost:8080/metrics
```

## 工作原理

1. **请求接收**: 转发器在配置的端口上接收 HTTP 请求
2. **端点选择**: 基于配置的策略（优先级/最快），选择最佳可用的健康端点
3. **请求转发**: 透明地转发请求，正确处理头部：
   - **Host 头部**: 自动设置为匹配目标端点的主机名
   - **Authorization**: 根据配置覆盖/注入令牌，删除客户端令牌
   - **安全性**: 自动删除敏感的客户端头部（`X-API-Key`、`Authorization`）
   - **自定义头部**: 根据配置添加端点特定的头部
   - **原始头部**: 保留来自原始请求的所有其他头部
4. **响应处理**:
   - 常规请求：缓冲并转发完整的响应
   - SSE 请求：实时流式传输响应，正确处理事件
5. **错误处理**: 失败时，自动以指数退避方式重试，然后故障转移到下一个可用端点
6. **健康监控**: 持续监控端点健康状况并相应调整路由

## 命令行选项

```bash
./endpoint_forwarder [OPTIONS]
```

选项：
- `-config path/to/config.yaml`: 配置文件路径（默认："config/example.yaml"）
- `-version`: 显示版本信息
- `-tui`: 启用 TUI 界面（默认：true）
- `-no-tui`: 禁用 TUI 界面（在传统控制台模式下运行）

示例：
```bash
# 默认模式，启用 TUI
./endpoint_forwarder -config my-config.yaml

# 不使用 TUI 运行（传统控制台日志）
./endpoint_forwarder -config my-config.yaml -no-tui

# 显示版本信息
./endpoint_forwarder -version
```

## 日志记录

应用程序使用结构化日志，具有增强的格式以提高人类可读性：

```yaml
logging:
  level: "info"    # debug, info, warn, error
  format: "text"   # text（人类可读）或 json（机器可读）
```

### 日志功能

**增强可读性:**
- 🎯 不同日志类型和状态的表情符号指示器
- 📊 格式化的响应时间（μs/ms/s）和数据大小（B/KB/MB）
- 🚀 带有端点信息的请求生命周期跟踪
- ⏱️ 精确的时间戳格式（HH:MM:SS.mmm）

**请求日志记录:**
- 请求开始与所选端点名称
- 响应完成与状态指示器
- 具有适当严重级别的错误跟踪
- 性能监控（慢请求检测）

**日志示例:**
```
15:04:05.123 level=INFO msg="🚀 Request started" method=POST path=/v1/messages client_ip=192.168.1.100 user_agent="Claude-Client/1.0" content_length=245
15:04:05.456 level=INFO msg="🎯 Selected endpoint" endpoint=primary url=https://api.anthropic.com priority=1 attempt=1 total_endpoints=3  
15:04:06.789 level=INFO msg="✅ Request completed" method=POST path=/v1/messages endpoint=primary status_code=200 bytes_written=1.2KB duration=633.2ms client_ip=192.168.1.100
```

**安全功能:**
- 自动删除敏感的客户端头部（`X-API-Key`、`Authorization`）
- 替换为端点配置的令牌
- 防止客户端和后端之间的凭据泄露

## 生产环境考虑

- 为您的使用场景配置适当的超时时间
- 监控 `/health` 和 `/metrics` 端点
- 使用反向代理（nginx/Apache）进行 SSL 终端
- 为生产部署配置日志轮转
- 基于端点健康指标设置警报
- 考虑在反向代理级别进行速率限制

## 许可证

此项目按原样提供，用于教育和开发目的。