package webui

// indexHTML contains the main HTML page
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Claude EndPoints Forwarder WebUI</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <header class="header">
            <h1>🚀 Claude EndPoints Forwarder WebUI</h1>
            <div class="header-controls">
                <div class="status-bar">
                    <span id="status-requests">请求数: 0</span>
                    <span id="status-success">成功率: 0.0%</span>
                    <span id="status-connections">连接数: 0</span>
                    <span id="last-update">最后更新: --:--:--</span>
                </div>
                <div class="auth-controls">
                    <a href="/logout" class="logout-btn" title="退出登录">🚪</a>
                </div>
            </div>
        </header>

        <nav class="nav-tabs">
            <button class="tab-button active" data-tab="overview">📊 概览</button>
            <button class="tab-button" data-tab="endpoints">🎯 端点</button>
            <button class="tab-button" data-tab="connections">🔌 连接</button>
            <button class="tab-button" data-tab="logs">📝 日志</button>
            <button class="tab-button" data-tab="config">⚙️ 配置</button>
        </nav>

        <main class="main-content">
            <!-- Overview Tab -->
            <div id="overview" class="tab-content active">
                <div class="grid-2x2">
                    <div class="card">
                        <h3>📊 Request Metrics</h3>
                        <div id="metrics-content">
                            <div class="metric">
                                <span class="label">总请求数:</span>
                                <span class="value" id="total-requests">0</span>
                            </div>
                            <div class="metric">
                                <span class="label">成功:</span>
                                <span class="value success" id="successful-requests">0 (0.0%)</span>
                            </div>
                            <div class="metric">
                                <span class="label">失败:</span>
                                <span class="value error" id="failed-requests">0 (0.0%)</span>
                            </div>
                            <div class="metric">
                                <span class="label">平均响应时间:</span>
                                <span class="value" id="avg-response-time">0ms</span>
                            </div>
                            <div class="token-section">
                                <h4>🪙 令牌使用情况</h4>
                                <div class="metric">
                                    <span class="label">📥 输入令牌:</span>
                                    <span class="value" id="input-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">📤 输出令牌:</span>
                                    <span class="value" id="output-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">🆕 缓存创建:</span>
                                    <span class="value" id="cache-creation-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">📖 缓存读取:</span>
                                    <span class="value" id="cache-read-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">🔢 总令牌数:</span>
                                    <span class="value highlight" id="total-tokens">0</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3>🪙 Historical Token Usage</h3>
                        <div id="token-history-content">
                            <div id="token-chart" class="chart-area">
                                <div class="loading">正在加载令牌历史...</div>
                            </div>
                            <div class="chart-legend">
                                <div class="legend-item">
                                    <span class="legend-color input"></span>
                                    <span class="legend-label">输入令牌</span>
                                </div>
                                <div class="legend-item">
                                    <span class="legend-color output"></span>
                                    <span class="legend-label">输出令牌</span>
                                </div>
                                <div class="legend-item">
                                    <span class="legend-color cache"></span>
                                    <span class="legend-label">缓存令牌</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3>🎯 Endpoints Status</h3>
                        <div id="endpoints-status-content">
                            <div class="metric">
                                <span class="label">Total:</span>
                                <span class="value" id="endpoints-total">0</span>
                                <span class="label">Healthy:</span>
                                <span class="value success" id="endpoints-healthy">0</span>
                            </div>
                            <div id="endpoints-list"></div>
                        </div>
                    </div>

                    <div class="card">
                        <h3>💻 System Info</h3>
                        <div id="system-info-content">
                            <div class="metric">
                                <span class="label">Active Connections:</span>
                                <span class="value" id="active-connections">0</span>
                            </div>
                            <div class="metric">
                                <span class="label">Total Connections:</span>
                                <span class="value" id="total-connections">0</span>
                            </div>
                            <div class="metric">
                                <span class="label">Uptime:</span>
                                <span class="value" id="uptime">0s</span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Endpoints Tab -->
            <div id="endpoints" class="tab-content">
                <div class="endpoints-layout">
                    <div class="endpoints-table-container">
                        <div class="endpoints-header">
                            <h3 id="endpoints-title">🎯 Endpoints</h3>
                            <div class="endpoints-controls">
                                <button id="edit-mode-btn" class="btn btn-primary">✏️ 编辑模式</button>
                                <button id="save-config-btn" class="btn btn-success" style="display: none;">💾 保存</button>
                                <button id="cancel-edit-btn" class="btn btn-secondary" style="display: none;">❌ 取消</button>
                            </div>
                        </div>
                        <table id="endpoints-table">
                            <thead>
                                <tr>
                                    <th>状态</th>
                                    <th>名称</th>
                                    <th>URL</th>
                                    <th>优先级</th>
                                    <th>响应时间</th>
                                    <th>请求数</th>
                                    <th>失败数</th>
                                </tr>
                            </thead>
                            <tbody id="endpoints-table-body">
                                <tr>
                                    <td colspan="7" class="placeholder">正在加载端点...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="endpoint-details">
                        <h3>📊 详细信息</h3>
                        <div id="endpoint-details-content">
                            <p class="placeholder">选择一个端点查看详细信息</p>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Connections Tab -->
            <div id="connections" class="tab-content">
                <div class="card">
                    <h3>🔌 Connection Statistics</h3>
                    <div id="connections-stats">
                        <div class="metric">
                            <span class="label">Active:</span>
                            <span class="value" id="connections-active">0</span>
                            <span class="label">Historical:</span>
                            <span class="value" id="connections-historical">0</span>
                        </div>
                    </div>
                </div>

                <div class="card">
                    <h3>🔗 Active Connections</h3>
                    <div class="connections-header">
                        <div class="connections-legend">
                            <span class="legend-item">
                                <span class="connection-status active"></span>
                                <span>Active</span>
                            </span>
                            <span class="legend-item">
                                <span class="connection-status completed"></span>
                                <span>Completed</span>
                            </span>
                            <span class="legend-item">
                                <span class="connection-status failed"></span>
                                <span>Failed</span>
                            </span>
                            <span class="legend-item">
                                <span class="connection-status streaming"></span>
                                <span>Streaming</span>
                            </span>
                        </div>
                    </div>
                    <div id="connections-list" class="connections-container">
                        <div class="connections-table-header">
                            <div class="conn-col-client">客户端IP</div>
                            <div class="conn-col-method">方法</div>
                            <div class="conn-col-path">路径</div>
                            <div class="conn-col-endpoint">端点</div>
                            <div class="conn-col-group">分组</div>
                            <div class="conn-col-retry">重试</div>
                            <div class="conn-col-duration">持续时间</div>
                        </div>
                        <div id="connections-table-body">
                            <div class="placeholder">无活动连接</div>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Logs Tab -->
            <div id="logs" class="tab-content">
                <div class="card">
                    <h3>📝 系统日志</h3>
                    <div id="logs-content">
                        <div class="log-entry">
                            <span class="log-time">--:--:--</span>
                            <span class="log-level info">[INF]</span>
                            <span class="log-source">webui</span>
                            <span class="log-message">WebUI服务器正在运行</span>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Config Tab -->
            <div id="config" class="tab-content">
                <div class="config-grid">
                    <div class="card">
                        <h3>🌐 服务器</h3>
                        <div id="config-server"></div>
                    </div>
                    <div class="card">
                        <h3>🎯 策略</h3>
                        <div id="config-strategy"></div>
                    </div>
                    <div class="card">
                        <h3>🔐 身份验证</h3>
                        <div id="config-auth"></div>
                    </div>
                    <div class="card">
                        <h3>🖥️ 界面</h3>
                        <div id="config-interface"></div>
                    </div>
                    <div class="card full-width">
                        <h3>🎯 端点配置</h3>
                        <div id="config-endpoints"></div>
                    </div>
                    <div class="card full-width">
                        <h3>📁 配置管理</h3>
                        <div class="config-manager">
                            <!-- 当前活动配置显示 -->
                            <div class="active-config">
                                <span class="label">当前配置：</span>
                                <strong id="current-config-name">加载中...</strong>
                                <button id="refresh-configs" onclick="app.loadConfigs()">🔄 刷新</button>
                                <button id="export-all-configs" onclick="app.exportAllConfigs()">📦 批量导出</button>
                            </div>

                            <!-- 配置导入区域 -->
                            <div class="import-section">
                                <h4>导入新配置</h4>
                                <div class="import-form">
                                    <input type="file" id="config-file" accept=".yaml,.yml" />
                                    <input type="text" id="config-name" placeholder="配置名称" />
                                    <button onclick="app.importConfig()">导入配置</button>
                                </div>
                            </div>

                            <!-- 配置列表 -->
                            <div class="config-list-section">
                                <h4>可用配置</h4>
                                <div class="config-list" id="config-list">
                                    <!-- 动态生成配置列表 -->
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </main>
    </div>

    <!-- 配置编辑器模态框 -->
    <div id="config-editor-modal" class="modal" style="display:none;">
        <div class="modal-content">
            <div class="modal-header">
                <h3 id="config-editor-title">编辑配置</h3>
                <button class="modal-close" onclick="app.closeConfigEditor()">×</button>
            </div>
            <div class="modal-body">
                <textarea id="config-editor-content" spellcheck="false" style="width:100%;height:360px;font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; background:#0b1220; color:#e2e8f0; border:1px solid #334155; border-radius:8px; padding:12px; line-height:1.4;"></textarea>
                <div id="config-editor-error" style="display:none;color:#ef4444;margin-top:8px;"></div>
            </div>
            <div class="modal-footer">
                <button class="btn btn-secondary" onclick="app.closeConfigEditor()">取消</button>
                <button class="btn btn-success" onclick="app.saveConfigEditor()">💾 保存并应用</button>
            </div>
        </div>
    </div>

    <script src="/static/app.js"></script>
</body>
</html>`

// styleCSS contains the CSS styles
const styleCSS = `
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0f172a;
    color: #e2e8f0;
    line-height: 1.6;
    overflow-x: hidden;
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
    overflow-x: hidden;
    width: 100%;
}

/* Modal styles */
.modal {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(15, 23, 42, 0.75);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
}
.modal-content {
    width: 80%;
    max-width: 900px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 10px;
    box-shadow: 0 10px 30px rgba(0,0,0,0.4);
}
.modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid #334155;
}
.modal-header h3 { margin: 0; }
.modal-close {
    background: transparent;
    border: none;
    color: #94a3b8;
    font-size: 24px;
    cursor: pointer;
}
.modal-footer {
    display: flex; gap: 10px; justify-content: flex-end;
    padding: 12px 16px;
    border-top: 1px solid #334155;
}

.header {
    text-align: center;
    margin-bottom: 30px;
    padding: 20px;
    background: linear-gradient(135deg, #1e293b, #334155);
    border-radius: 12px;
    border: 1px solid #334155;
    position: relative;
}

.header h1 {
    color: #60a5fa;
    margin-bottom: 15px;
    font-size: 2rem;
}

.header-controls {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 20px;
    flex-wrap: wrap;
}

.status-bar {
    display: flex;
    justify-content: center;
    gap: 30px;
    flex-wrap: wrap;
}

.auth-controls {
    position: absolute;
    top: 20px;
    right: 20px;
}

.logout-btn {
    display: inline-block;
    padding: 8px 12px;
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    text-decoration: none;
    border-radius: 6px;
    border: 1px solid rgba(239, 68, 68, 0.3);
    transition: all 0.2s;
    font-size: 1.2rem;
}

.logout-btn:hover {
    background: rgba(239, 68, 68, 0.2);
    border-color: rgba(239, 68, 68, 0.5);
    transform: translateY(-1px);
}

.status-bar span {
    padding: 8px 16px;
    background: #1e293b;
    border-radius: 6px;
    border: 1px solid #475569;
    font-size: 0.9rem;
}

.nav-tabs {
    display: flex;
    gap: 5px;
    margin-bottom: 30px;
    background: #1e293b;
    padding: 5px;
    border-radius: 12px;
    border: 1px solid #334155;
}

.tab-button {
    flex: 1;
    padding: 12px 20px;
    background: transparent;
    border: none;
    color: #94a3b8;
    cursor: pointer;
    border-radius: 8px;
    transition: all 0.2s;
    font-size: 0.95rem;
}

.tab-button:hover {
    background: #334155;
    color: #e2e8f0;
}

.tab-button.active {
    background: #3b82f6;
    color: white;
}

.main-content {
    min-height: 600px;
}

.tab-content {
    display: none;
}

.tab-content.active {
    display: block;
}

.card {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 20px;
    margin-bottom: 20px;
    min-width: 0;
    overflow: hidden;
}

.card h3 {
    color: #60a5fa;
    margin-bottom: 15px;
    font-size: 1.1rem;
}

.grid-2x2 {
    display: grid;
    grid-template-columns: minmax(400px, 1fr) minmax(400px, 1fr);
    gap: 20px;
    width: 100%;
}

@media (max-width: 768px) {
    .grid-2x2 {
        grid-template-columns: 1fr;
    }
}

.metric {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid #334155;
}

.metric:last-child {
    border-bottom: none;
}

.metric .label {
    color: #94a3b8;
    font-size: 0.9rem;
}

.metric .value {
    font-weight: 600;
    color: #60a5fa;
}

.metric .value.success {
    color: #10b981;
}

.metric .value.error {
    color: #ef4444;
}

.metric .value.highlight {
    color: #a855f7;
    font-size: 1.1rem;
}

.token-section {
    margin-top: 15px;
    padding-top: 15px;
    border-top: 1px solid #334155;
}

.token-section h4 {
    color: #fbbf24;
    margin-bottom: 10px;
    font-size: 1rem;
}

.placeholder {
    color: #64748b;
    font-style: italic;
    text-align: center;
    padding: 20px;
}

.endpoints-layout {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 20px;
}

@media (max-width: 1024px) {
    .endpoints-layout {
        grid-template-columns: 1fr;
    }
}

.endpoints-table-container {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 20px;
}

.endpoint-details {
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 12px;
    padding: 20px;
}

table {
    width: 100%;
    border-collapse: collapse;
    margin-top: 15px;
}

th, td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid #334155;
}

th {
    background: #334155;
    color: #94a3b8;
    font-weight: 600;
    font-size: 0.9rem;
}

tr:hover {
    background: #334155;
    cursor: pointer;
}

.status-icon {
    font-size: 1.2rem;
}

.config-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;
}

.config-grid .full-width {
    grid-column: 1 / -1;
}

.log-entry {
    display: flex;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid #334155;
    font-family: 'Courier New', monospace;
    font-size: 0.9rem;
}

.log-time {
    color: #64748b;
    min-width: 80px;
}

.log-level {
    min-width: 50px;
    font-weight: 600;
}

.log-level.info {
    color: #60a5fa;
}

.log-level.warn {
    color: #fbbf24;
}

.log-level.error {
    color: #ef4444;
}

.log-source {
    color: #94a3b8;
    min-width: 80px;
}

.log-message {
    color: #e2e8f0;
    flex: 1;
}

.history-item {
    padding: 10px 0;
    border-bottom: 1px solid #334155;
}

.history-placeholder {
    color: #64748b;
    font-style: italic;
}

.connection-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 0;
    border-bottom: 1px solid #334155;
    font-family: 'Courier New', monospace;
    font-size: 0.9rem;
}

.connection-info {
    display: flex;
    gap: 15px;
}

.connection-duration {
    color: #64748b;
}

/* Loading animation */
@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

.loading {
    animation: pulse 2s infinite;
}

/* Chart styles */
.chart-area {
    height: 200px;
    max-height: 200px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    padding: 10px;
    margin-bottom: 10px;
    position: relative;
    overflow: auto;
}

.chart-legend {
    display: flex;
    justify-content: center;
    gap: 15px;
    flex-wrap: wrap;
}

.legend-item {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 0.85rem;
}

.legend-color {
    width: 12px;
    height: 12px;
    border-radius: 2px;
}

.legend-color.input {
    background: #60a5fa;
}

.legend-color.output {
    background: #34d399;
}

.legend-color.cache {
    background: #fbbf24;
}

.legend-label {
    color: #cbd5e1;
}

/* Table selection styles */
#endpoints-table tbody tr {
    cursor: pointer;
    transition: background-color 0.2s ease;
}

#endpoints-table tbody tr:hover {
    background-color: #334155;
}

#endpoints-table tbody tr.selected {
    background-color: #1e40af;
}

#endpoints-table tbody tr.selected:hover {
    background-color: #1d4ed8;
}

/* Endpoints header and controls */
.endpoints-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
}

.endpoints-controls {
    display: flex;
    gap: 10px;
}

.btn {
    padding: 8px 16px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    transition: all 0.2s ease;
    display: inline-flex;
    align-items: center;
    gap: 5px;
}

.btn:hover {
    transform: translateY(-1px);
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.2);
}

.btn-primary {
    background: #3b82f6;
    color: white;
}

.btn-primary:hover {
    background: #2563eb;
}

.btn-success {
    background: #10b981;
    color: white;
}

.btn-success:hover {
    background: #059669;
}

.btn-secondary {
    background: #6b7280;
    color: white;
}

.btn-secondary:hover {
    background: #4b5563;
}

/* Edit mode styles */
.edit-mode .priority-cell {
    position: relative;
}

.priority-input {
    background: #374151;
    border: 1px solid #60a5fa;
    border-radius: 4px;
    color: white;
    padding: 4px 8px;
    width: 60px;
    text-align: center;
    font-size: 0.9rem;
}

.priority-input:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

.unsaved-changes {
    color: #fbbf24 !important;
}

.edit-mode-indicator {
    background: #1e40af;
    color: white;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.8rem;
    margin-left: 10px;
}

/* Message toast styles */
.message-toast {
    position: fixed;
    top: 20px;
    right: 20px;
    padding: 12px 20px;
    border-radius: 8px;
    color: white;
    font-weight: 500;
    z-index: 1000;
    animation: slideIn 0.3s ease-out;
    max-width: 400px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.message-success {
    background: #10b981;
}

.message-error {
    background: #ef4444;
}

.message-info {
    background: #3b82f6;
}

@keyframes slideIn {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

/* Connections styles */
.connections-header {
    margin-bottom: 15px;
}

.connections-legend {
    display: flex;
    gap: 20px;
    flex-wrap: wrap;
    justify-content: center;
    padding: 10px;
    background: #0f172a;
    border-radius: 6px;
}

.connections-legend .legend-item {
    display: flex;
    align-items: center;
    gap: 5px;
    font-size: 0.85rem;
    color: #cbd5e1;
}

.connection-status {
    width: 10px;
    height: 10px;
    border-radius: 50%;
}

.connection-status.active {
    background: #10b981;
}

.connection-status.completed {
    background: #3b82f6;
}

.connection-status.failed {
    background: #ef4444;
}

.connection-status.streaming {
    background: #f59e0b;
    animation: pulse 2s infinite;
}

.connections-container {
    font-family: 'Courier New', monospace;
    font-size: 0.85rem;
}

.connections-table-header {
    display: grid;
    grid-template-columns: 1.2fr 0.6fr 1.8fr 1fr 1.2fr 0.8fr 1fr;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 2px solid #334155;
    font-weight: 600;
    color: #60a5fa;
    background: #0f172a;
    border-radius: 6px 6px 0 0;
    padding-left: 10px;
    padding-right: 10px;
}

.connection-row {
    display: grid;
    grid-template-columns: 1.2fr 0.6fr 1.8fr 1fr 1.2fr 0.8fr 1fr;
    gap: 10px;
    padding: 6px 10px;
    border-bottom: 1px solid #334155;
    align-items: center;
    transition: background-color 0.2s ease;
}

.connection-row:hover {
    background: #1e293b;
}

.conn-col-client,
.conn-col-method,
.conn-col-path,
.conn-col-endpoint,
.conn-col-group,
.conn-col-retry,
.conn-col-duration {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.conn-col-method {
    color: #fbbf24;
    font-weight: 600;
}

.conn-col-endpoint {
    color: #34d399;
}

.conn-col-group {
    color: #a855f7;
}

.conn-col-retry {
    color: #f87171;
}

.conn-col-duration {
    color: #64748b;
}

/* Log entry animations */
.log-entry {
    display: flex;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid #334155;
    font-family: 'Courier New', monospace;
    font-size: 0.9rem;
    animation: logFadeIn 0.3s ease-in;
}

@keyframes logFadeIn {
    from { 
        opacity: 0; 
        transform: translateY(-10px);
        background-color: rgba(96, 165, 250, 0.2);
    }
    to { 
        opacity: 1; 
        transform: translateY(0);
        background-color: transparent;
    }
}

/* Scrollable log container */
#logs-content {
    max-height: 500px;
    overflow-y: auto;
    padding: 10px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
}

/* Custom scrollbar */
#logs-content::-webkit-scrollbar {
    width: 8px;
}

#logs-content::-webkit-scrollbar-track {
    background: #1e293b;
    border-radius: 4px;
}

#logs-content::-webkit-scrollbar-thumb {
    background: #475569;
    border-radius: 4px;
}

#logs-content::-webkit-scrollbar-thumb:hover {
    background: #64748b;
}

/* Configuration Management Styles */
.config-manager {
    display: flex;
    flex-direction: column;
    gap: 20px;
}

.active-config {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 15px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
}

.active-config .label {
    color: #94a3b8;
    font-weight: 500;
}

.active-config strong {
    color: #10b981;
    font-size: 1.1em;
}

.active-config button {
    margin-left: auto;
    padding: 5px 10px;
    background: #374151;
    color: #e5e7eb;
    border: 1px solid #4b5563;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.9em;
    transition: background-color 0.2s;
}

.active-config button:hover {
    background: #4b5563;
}

.import-section {
    padding: 15px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
}

.import-section h4 {
    color: #e2e8f0;
    margin-bottom: 15px;
    font-size: 1.1em;
}

.import-form {
    display: flex;
    gap: 10px;
    align-items: center;
    flex-wrap: wrap;
}

.import-form input[type="file"] {
    flex: 1;
    min-width: 200px;
    padding: 8px;
    background: #1e293b;
    color: #e2e8f0;
    border: 1px solid #475569;
    border-radius: 4px;
}

.import-form input[type="text"] {
    flex: 1;
    min-width: 150px;
    padding: 8px;
    background: #1e293b;
    color: #e2e8f0;
    border: 1px solid #475569;
    border-radius: 4px;
}

.import-form input[type="text"]:focus,
.import-form input[type="file"]:focus {
    outline: none;
    border-color: #10b981;
}

.import-form button {
    padding: 8px 16px;
    background: #10b981;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
    transition: background-color 0.2s;
}

.import-form button:hover {
    background: #059669;
}

.config-list-section {
    padding: 15px;
    background: #0f172a;
    border: 1px solid #334155;
    border-radius: 8px;
}

.config-list-section h4 {
    color: #e2e8f0;
    margin-bottom: 15px;
    font-size: 1.1em;
}

.config-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.config-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px;
    background: #1e293b;
    border: 1px solid #475569;
    border-radius: 6px;
    transition: border-color 0.2s;
}

.config-item:hover {
    border-color: #64748b;
}

.config-item.active {
    border-color: #10b981;
    background: rgba(16, 185, 129, 0.1);
}

.config-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.config-name {
    color: #e2e8f0;
    font-weight: 500;
    font-size: 1em;
}

.config-details {
    color: #94a3b8;
    font-size: 0.85em;
}

.config-actions {
    display: flex;
    gap: 8px;
}

.config-actions button {
    padding: 6px 12px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 0.85em;
    font-weight: 500;
    transition: background-color 0.2s;
}

.config-actions .switch-btn {
    background: #3b82f6;
    color: white;
}

.config-actions .switch-btn:hover {
    background: #2563eb;
}

.config-actions .switch-btn:disabled {
    background: #6b7280;
    cursor: not-allowed;
}

.config-actions .rename-btn {
    background: #f59e0b;
    color: white;
}

.config-actions .rename-btn:hover {
    background: #d97706;
}

.config-actions .delete-btn {
    background: #ef4444;
    color: white;
}

.config-actions .delete-btn:hover {
    background: #dc2626;
}

.config-actions .delete-btn:disabled {
    background: #6b7280;
    cursor: not-allowed;
}
`

// loginHTML contains the login page
const loginHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebUI 登录 - Claude EndPoints Forwarder</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
        }

        .login-header {
            text-align: center;
            margin-bottom: 2rem;
        }

        .login-header h1 {
            color: #333;
            margin-bottom: 0.5rem;
        }

        .login-header p {
            color: #666;
            font-size: 0.9rem;
        }

        .form-group {
            margin-bottom: 1.5rem;
        }

        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }

        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e1e5e9;
            border-radius: 5px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }

        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }

        .login-button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: transform 0.2s;
        }

        .login-button:hover {
            transform: translateY(-1px);
        }

        .login-button:active {
            transform: translateY(0);
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1>🚀 WebUI 登录</h1>
            <p>Claude EndPoints Forwarder</p>
        </div>
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="password">密码:</label>
                <input type="password" id="password" name="password" required autofocus>
            </div>
            <button type="submit" class="login-button">登录</button>
        </form>
    </div>
</body>
</html>`

// loginHTMLWithError contains the login page with error message
const loginHTMLWithError = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WebUI 登录 - Claude EndPoints Forwarder</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 10px;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
            width: 100%;
            max-width: 400px;
        }

        .login-header {
            text-align: center;
            margin-bottom: 2rem;
        }

        .login-header h1 {
            color: #333;
            margin-bottom: 0.5rem;
        }

        .login-header p {
            color: #666;
            font-size: 0.9rem;
        }

        .error-message {
            background: #fee;
            color: #c33;
            padding: 0.75rem;
            border-radius: 5px;
            margin-bottom: 1.5rem;
            text-align: center;
            border: 1px solid #fcc;
        }

        .form-group {
            margin-bottom: 1.5rem;
        }

        .form-group label {
            display: block;
            margin-bottom: 0.5rem;
            color: #333;
            font-weight: 500;
        }

        .form-group input {
            width: 100%;
            padding: 0.75rem;
            border: 2px solid #e1e5e9;
            border-radius: 5px;
            font-size: 1rem;
            transition: border-color 0.3s;
        }

        .form-group input:focus {
            outline: none;
            border-color: #667eea;
        }

        .login-button {
            width: 100%;
            padding: 0.75rem;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 5px;
            font-size: 1rem;
            font-weight: 500;
            cursor: pointer;
            transition: transform 0.2s;
        }

        .login-button:hover {
            transform: translateY(-1px);
        }

        .login-button:active {
            transform: translateY(0);
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="login-header">
            <h1>🚀 WebUI 登录</h1>
            <p>Claude EndPoints Forwarder</p>
        </div>
        <div class="error-message">
            ❌ 密码错误，请重试
        </div>
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="password">密码:</label>
                <input type="password" id="password" name="password" required autofocus>
            </div>
            <button type="submit" class="login-button">登录</button>
        </form>
    </div>
</body>
</html>`

// appJS contains the JavaScript application code
const appJS = `
class WebUIApp {
    constructor() {
        this.currentTab = 'overview';
        this.selectedEndpoint = null;
        this.eventSource = null;
        this.logEventSource = null;

        // Edit mode state
        this.editMode = false;
        this.originalPriorities = {};
        this.currentPriorities = {};
        this.hasUnsavedChanges = false;
        this.editingConfigName = null; // for config editor

        this.init();
    }

    init() {
        this.setupTabs();
        this.setupEventSource();
        this.setupLogStream();
        this.setupEditMode();
        this.loadAllData();

        // Refresh data every 5 seconds as fallback
        setInterval(() => this.loadAllData(), 5000);
    }

    setupTabs() {
        const tabButtons = document.querySelectorAll('.tab-button');
        const tabContents = document.querySelectorAll('.tab-content');

        tabButtons.forEach(button => {
            button.addEventListener('click', () => {
                const tabName = button.dataset.tab;

                // Update active tab button
                tabButtons.forEach(b => b.classList.remove('active'));
                button.classList.add('active');

                // Update active tab content
                tabContents.forEach(content => content.classList.remove('active'));
                document.getElementById(tabName).classList.add('active');

                this.currentTab = tabName;
                this.loadTabData(tabName);
            });
        });
    }

    setupEventSource() {
        if (this.eventSource) {
            this.eventSource.close();
        }

        this.eventSource = new EventSource('/api/events');

        this.eventSource.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                this.updateStatusBar(data);
            } catch (e) {
                console.error('Error parsing SSE data:', e);
            }
        };

        this.eventSource.onerror = (error) => {
            console.error('SSE connection error:', error);
            // Reconnect after 5 seconds
            setTimeout(() => this.setupEventSource(), 5000);
        };
    }

    setupLogStream() {
        if (this.logEventSource) {
            this.logEventSource.close();
        }

        this.logEventSource = new EventSource('/api/log-stream');

        this.logEventSource.onmessage = (event) => {
            try {
                const logEntry = JSON.parse(event.data);
                this.addLogToUI(logEntry);
            } catch (e) {
                console.error('Error parsing log stream data:', e);
            }
        };

        this.logEventSource.onerror = (error) => {
            console.error('Log stream connection error:', error);
            // Reconnect after 3 seconds
            setTimeout(() => this.setupLogStream(), 3000);
        };
    }

    setupEditMode() {
        // Edit mode button
        const editModeBtn = document.getElementById('edit-mode-btn');
        const saveConfigBtn = document.getElementById('save-config-btn');
        const cancelEditBtn = document.getElementById('cancel-edit-btn');

        editModeBtn.addEventListener('click', () => this.enterEditMode());
        saveConfigBtn.addEventListener('click', () => this.saveConfiguration());
        cancelEditBtn.addEventListener('click', () => this.cancelEditMode());

        // Keyboard shortcuts (similar to TUI)
        document.addEventListener('keydown', (event) => {
            this.handleGlobalKeyboard(event);
        });
    }

    handleGlobalKeyboard(event) {
        // Don't handle shortcuts if user is typing in an input field
        if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
            // Only handle specific shortcuts in input fields
            if (event.key === 'Escape') {
                event.target.blur(); // Remove focus from input
                if (this.editMode) {
                    event.preventDefault();
                    this.cancelEditMode();
                }
            } else if (event.ctrlKey && event.key === 's' && this.editMode) {
                event.preventDefault();
                this.saveConfiguration();
            }
            return;
        }

        // Global tab switching shortcuts (similar to TUI)
        if (event.key >= '1' && event.key <= '5') {
            event.preventDefault();
            const tabIndex = parseInt(event.key) - 1;
            const tabs = ['overview', 'endpoints', 'connections', 'logs', 'config'];
            if (tabs[tabIndex]) {
                this.switchToTab(tabs[tabIndex]);
            }
        }

        // Tab navigation with Tab/Shift+Tab
        else if (event.key === 'Tab' && !event.ctrlKey && !event.altKey) {
            event.preventDefault();
            const tabs = ['overview', 'endpoints', 'connections', 'logs', 'config'];
            const currentIndex = tabs.indexOf(this.currentTab);

            if (event.shiftKey) {
                // Previous tab
                const prevIndex = currentIndex > 0 ? currentIndex - 1 : tabs.length - 1;
                this.switchToTab(tabs[prevIndex]);
            } else {
                // Next tab
                const nextIndex = currentIndex < tabs.length - 1 ? currentIndex + 1 : 0;
                this.switchToTab(tabs[nextIndex]);
            }
        }

        // Endpoints tab specific shortcuts
        else if (this.currentTab === 'endpoints') {
            if (event.key === 'Enter' && !this.editMode) {
                event.preventDefault();
                this.enterEditMode();
            } else if (event.key === 'Escape' && this.editMode) {
                event.preventDefault();
                this.cancelEditMode();
            } else if (event.ctrlKey && event.key === 's' && this.editMode) {
                event.preventDefault();
                this.saveConfiguration();
            }
            // Priority shortcuts in edit mode (1-9 keys)
            else if (this.editMode && event.key >= '1' && event.key <= '9' && this.selectedEndpoint) {
                event.preventDefault();
                const priority = parseInt(event.key);
                this.setPriorityForSelected(priority);
            }
        }

        // Global shortcuts
        else if (event.key === 'F5') {
            event.preventDefault();
            this.loadAllData();
        }
    }

    switchToTab(tabName) {
        // Find and click the corresponding tab button
        const tabButton = document.querySelector('[data-tab="' + tabName + '"]');
        if (tabButton) {
            tabButton.click();
        }
    }

    setPriorityForSelected(priority) {
        if (!this.selectedEndpoint || !this.editMode) return;

        const endpointName = this.selectedEndpoint.name;
        const input = document.querySelector('input[data-endpoint="' + endpointName + '"]');

        if (input) {
            input.value = priority;
            input.dispatchEvent(new Event('input')); // Trigger the change handler
        }
    }

    enterEditMode() {
        this.editMode = true;
        this.hasUnsavedChanges = false;

        // Store original priorities
        this.originalPriorities = {};
        this.currentPriorities = {};

        const rows = document.querySelectorAll('#endpoints-table tbody tr');
        rows.forEach(row => {
            const nameCell = row.querySelector('td:nth-child(2)');
            const priorityCell = row.querySelector('td:nth-child(4)');

            if (nameCell && priorityCell) {
                const endpointName = nameCell.textContent;
                const priority = parseInt(priorityCell.textContent);
                this.originalPriorities[endpointName] = priority;
                this.currentPriorities[endpointName] = priority;

                // Replace priority text with input
                priorityCell.innerHTML = '<input type="number" class="priority-input" value="' + priority + '" min="0" max="999" data-endpoint="' + endpointName + '">';

                // Add event listener for changes
                const input = priorityCell.querySelector('.priority-input');
                input.addEventListener('input', (e) => this.onPriorityChange(endpointName, parseInt(e.target.value)));
            }
        });

        // Update UI
        document.querySelector('#endpoints-table').classList.add('edit-mode');
        this.updateEditModeUI();
    }

    onPriorityChange(endpointName, newPriority) {
        this.currentPriorities[endpointName] = newPriority;

        // Check if there are unsaved changes
        this.hasUnsavedChanges = Object.keys(this.originalPriorities).some(name =>
            this.originalPriorities[name] !== this.currentPriorities[name]
        );

        this.updateEditModeUI();
    }

    updateEditModeUI() {
        const title = document.getElementById('endpoints-title');
        const editModeBtn = document.getElementById('edit-mode-btn');
        const saveConfigBtn = document.getElementById('save-config-btn');
        const cancelEditBtn = document.getElementById('cancel-edit-btn');

        if (this.editMode) {
            let titleText = '🎯 Endpoints [Edit Mode';
            if (this.hasUnsavedChanges) {
                titleText += ' *';
            }
            titleText += ']';
            title.innerHTML = titleText + '<span class="edit-mode-indicator">ESC to Exit | Ctrl+S to Save</span>';

            editModeBtn.style.display = 'none';
            saveConfigBtn.style.display = 'inline-flex';
            cancelEditBtn.style.display = 'inline-flex';

            // Update save button state
            if (this.hasUnsavedChanges) {
                saveConfigBtn.classList.remove('btn-secondary');
                saveConfigBtn.classList.add('btn-success');
                saveConfigBtn.textContent = '💾 Save Changes';
            } else {
                saveConfigBtn.classList.remove('btn-success');
                saveConfigBtn.classList.add('btn-secondary');
                saveConfigBtn.textContent = '💾 No Changes';
            }
        } else {
            title.textContent = '🎯 Endpoints';
            editModeBtn.style.display = 'inline-flex';
            saveConfigBtn.style.display = 'none';
            cancelEditBtn.style.display = 'none';
        }
    }

    async saveConfiguration() {
        if (!this.hasUnsavedChanges) {
            return;
        }

        try {
            // Save each changed priority
            for (const endpointName of Object.keys(this.currentPriorities)) {
                if (this.originalPriorities[endpointName] !== this.currentPriorities[endpointName]) {
                    const response = await fetch('/api/endpoints/priority', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            endpointName: endpointName,
                            priority: this.currentPriorities[endpointName]
                        })
                    });

                    if (!response.ok) {
                        throw new Error('Failed to update priority for ' + endpointName);
                    }
                }
            }

            // Save configuration to file
            const saveResponse = await fetch('/api/config/save', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({})
            });

            if (!saveResponse.ok) {
                throw new Error('Failed to save configuration');
            }

            const saveResult = await saveResponse.json();

            // Show success message
            this.showMessage('✅ Configuration saved successfully' + (saveResult.savedToFile ? ' to file' : ' to memory'), 'success');

            // Update original priorities to current ones
            this.originalPriorities = { ...this.currentPriorities };
            this.hasUnsavedChanges = false;

            // Exit edit mode
            this.exitEditMode();

            // Reload endpoints to reflect changes
            await this.loadEndpoints();

        } catch (error) {
            console.error('Error saving configuration:', error);
            this.showMessage('❌ Failed to save configuration: ' + error.message, 'error');
        }
    }

    cancelEditMode() {
        // Restore original priorities
        this.currentPriorities = { ...this.originalPriorities };
        this.hasUnsavedChanges = false;

        this.exitEditMode();
    }

    exitEditMode() {
        this.editMode = false;

        // Remove edit mode class
        document.querySelector('#endpoints-table').classList.remove('edit-mode');

        // Restore priority cells to text
        const rows = document.querySelectorAll('#endpoints-table tbody tr');
        rows.forEach(row => {
            const nameCell = row.querySelector('td:nth-child(2)');
            const priorityCell = row.querySelector('td:nth-child(4)');

            if (nameCell && priorityCell) {
                const endpointName = nameCell.textContent;
                const priority = this.originalPriorities[endpointName] || 0;
                priorityCell.textContent = priority;
            }
        });

        this.updateEditModeUI();
    }

    showMessage(message, type = 'info') {
        // Create a temporary message element
        const messageDiv = document.createElement('div');
        messageDiv.className = 'message-toast message-' + type;
        messageDiv.textContent = message;

        // Add to page
        document.body.appendChild(messageDiv);

        // Remove after 3 seconds
        setTimeout(() => {
            if (messageDiv.parentNode) {
                messageDiv.parentNode.removeChild(messageDiv);
            }
        }, 3000);
    }

    updateStatusBar(data) {
        document.getElementById('status-requests').textContent = 'Requests: ' + data.totalRequests;
        document.getElementById('status-success').textContent = 'Success: ' + data.successRate.toFixed(1) + '%';
        document.getElementById('status-connections').textContent = 'Connections: ' + data.activeConnections;
        document.getElementById('last-update').textContent = 'Last Update: ' + new Date().toLocaleTimeString();
    }

    addLogToUI(logEntry) {
        // Only update if we're on the logs tab
        if (this.currentTab !== 'logs') {
            return;
        }

        const logsContent = document.getElementById('logs-content');
        if (!logsContent) {
            return;
        }

        // Create new log entry element
        const logDiv = document.createElement('div');
        logDiv.className = 'log-entry';

        const levelClass = logEntry.level.toLowerCase();
        const levelText = logEntry.level.substring(0, 3);

        logDiv.innerHTML = 
            '<span class="log-time">' + logEntry.timestamp + '</span>' +
            '<span class="log-level ' + levelClass + '">[' + levelText + ']</span>' +
            '<span class="log-source">' + logEntry.source + '</span>' +
            '<span class="log-message">' + logEntry.message + '</span>';

        // Insert at the top (most recent first)
        const firstChild = logsContent.firstChild;
        if (firstChild) {
            logsContent.insertBefore(logDiv, firstChild);
        } else {
            logsContent.appendChild(logDiv);
        }

        // Keep only latest 500 log entries in UI to prevent memory issues
        const logEntries = logsContent.querySelectorAll('.log-entry');
        if (logEntries.length > 500) {
            for (let i = 500; i < logEntries.length; i++) {
                logEntries[i].remove();
            }
        }

        // Auto-scroll to top if user is already at the top
        if (logsContent.scrollTop < 50) {
            logsContent.scrollTop = 0;
        }
    }

    async loadAllData() {
        await this.loadTabData(this.currentTab);
    }

    async loadTabData(tabName) {
        switch (tabName) {
            case 'overview':
                await this.loadOverview();
                break;
            case 'endpoints':
                await this.loadEndpoints();
                break;
            case 'connections':
                await this.loadConnections();
                break;
            case 'logs':
                await this.loadLogs();
                break;
            case 'config':
                await this.loadConfig();
                break;
        }
    }

    async loadOverview() {
        try {
            const response = await fetch('/api/overview');
            const data = await response.json();

            // Update metrics
            document.getElementById('total-requests').textContent = data.metrics.totalRequests;
            document.getElementById('successful-requests').textContent =
                data.metrics.successfulRequests + ' (' + data.metrics.successRate.toFixed(1) + '%)';
            document.getElementById('failed-requests').textContent =
                data.metrics.failedRequests + ' (' + (100 - data.metrics.successRate).toFixed(1) + '%)';
            document.getElementById('avg-response-time').textContent = data.metrics.averageResponseTime + 'ms';

            // Update token usage
            document.getElementById('input-tokens').textContent = data.tokens.inputTokens.toLocaleString();
            document.getElementById('output-tokens').textContent = data.tokens.outputTokens.toLocaleString();
            document.getElementById('cache-creation-tokens').textContent = data.tokens.cacheCreationTokens.toLocaleString();
            document.getElementById('cache-read-tokens').textContent = data.tokens.cacheReadTokens.toLocaleString();
            document.getElementById('total-tokens').textContent = data.tokens.totalTokens.toLocaleString();

            // Update endpoints status
            document.getElementById('endpoints-total').textContent = data.endpoints.total;
            document.getElementById('endpoints-healthy').textContent = data.endpoints.healthy;

            const endpointsList = document.getElementById('endpoints-list');
            endpointsList.innerHTML = '';
            data.endpoints.statuses.slice(0, 6).forEach(ep => {
                const div = document.createElement('div');
                div.className = 'metric';
                div.innerHTML =
                    '<span class="status-icon">' + (ep.healthy ? '🟢' : '🔴') + '</span>' +
                    '<span class="label">' + ep.name + '</span>' +
                    '<span class="value">(' + ep.responseTime + 'ms)</span>';
                endpointsList.appendChild(div);
            });

            // Update system info
            document.getElementById('active-connections').textContent = data.system.activeConnections;
            document.getElementById('total-connections').textContent = data.system.totalConnections;
            document.getElementById('uptime').textContent = this.formatUptime(data.system.uptime);

            // Load and update token history chart
            await this.loadTokenHistoryChart();

        } catch (error) {
            console.error('Error loading overview:', error);
        }
    }

    updateTokenHistory(history) {
        const historyList = document.getElementById('token-history-list');
        historyList.innerHTML = '';

        if (history && history.length > 0) {
            history.forEach((conn, index) => {
                const div = document.createElement('div');
                div.className = 'history-item';
                const statusIcon = conn.status === 'success' ? '✓' : '✗';
                const statusColor = conn.status === 'success' ? '#10b981' : '#ef4444';

                div.innerHTML =
                    '<div style="display: flex; justify-content: space-between; align-items: center;">' +
                    '<div>' +
                    '<span style="color: ' + statusColor + '">' + statusIcon + '</span> ' +
                    '<span style="color: #60a5fa">' + conn.clientIP + '</span> → ' +
                    '<span style="color: #fbbf24">' + conn.endpoint + '</span>' +
                    '</div>' +
                    '<div style="font-size: 0.9rem; color: #94a3b8">' +
                    '📥' + conn.tokenUsage.inputTokens + ' 📤' + conn.tokenUsage.outputTokens + ' ' +
                    '🔢' + conn.tokenUsage.totalTokens +
                    '</div>' +
                    '</div>';
                historyList.appendChild(div);
            });
        } else {
            const div = document.createElement('div');
            div.className = 'history-item';
            div.innerHTML = '<span class="history-placeholder">暂无令牌使用记录...</span>';
            historyList.appendChild(div);
        }
    }

    async loadTokenHistoryChart() {
        try {
            const response = await fetch('/api/overview/token-history');
            const data = await response.json();

            this.renderTokenChart(data);
        } catch (error) {
            console.error('Error loading token history:', error);
            document.getElementById('token-chart').innerHTML =
                '<div style="color: #ef4444; text-align: center; padding: 20px;">加载令牌历史失败</div>';
        }
    }

    renderTokenChart(data) {
        const chartContainer = document.getElementById('token-chart');

        if (!data.history || data.history.length === 0) {
            chartContainer.innerHTML =
                '<div style="color: #64748b; text-align: center; padding: 20px;">No token usage data available</div>';
            return;
        }

        // Simple ASCII-style chart rendering (similar to TUI)
        let chartHtml = '<div style="font-family: monospace; font-size: 0.8rem; line-height: 1.2;">';

        // Get the last 20 data points for display
        const displayData = data.history.slice(-20);
        const maxTokens = Math.max(...displayData.map(d => d.totalTokens));

        if (maxTokens === 0) {
            chartContainer.innerHTML =
                '<div style="color: #64748b; text-align: center; padding: 20px;">No token usage recorded</div>';
            return;
        }

        // Chart header
        chartHtml += '<div style="color: #60a5fa; margin-bottom: 10px; text-align: center;">令牌使用时间趋势</div>';

        // Simple bar chart
        displayData.forEach((point, index) => {
            const percentage = (point.totalTokens / maxTokens) * 100;
            const barWidth = Math.max(1, Math.floor(percentage / 2)); // Scale to fit

            const inputPerc = point.totalTokens > 0 ? (point.inputTokens / point.totalTokens) * barWidth : 0;
            const outputPerc = point.totalTokens > 0 ? (point.outputTokens / point.totalTokens) * barWidth : 0;
            const cachePerc = point.totalTokens > 0 ? ((point.cacheCreationTokens + point.cacheReadTokens) / point.totalTokens) * barWidth : 0;

            chartHtml += '<div style="display: flex; align-items: center; margin: 2px 0;">';
            chartHtml += '<span style="color: #64748b; width: 60px; font-size: 0.7rem;">' + point.timestamp + '</span>';
            chartHtml += '<div style="display: flex; margin-left: 10px;">';

            // Input tokens (blue)
            if (inputPerc > 0) {
                chartHtml += '<div style="background: #60a5fa; height: 12px; width: ' + Math.floor(inputPerc) + 'px;"></div>';
            }
            // Output tokens (green)
            if (outputPerc > 0) {
                chartHtml += '<div style="background: #34d399; height: 12px; width: ' + Math.floor(outputPerc) + 'px;"></div>';
            }
            // Cache tokens (yellow)
            if (cachePerc > 0) {
                chartHtml += '<div style="background: #fbbf24; height: 12px; width: ' + Math.floor(cachePerc) + 'px;"></div>';
            }

            chartHtml += '</div>';
            chartHtml += '<span style="color: #94a3b8; margin-left: 10px; font-size: 0.7rem;">' + point.totalTokens.toLocaleString() + '</span>';
            chartHtml += '</div>';
        });

        chartHtml += '</div>';
        chartContainer.innerHTML = chartHtml;
    }

    async loadEndpoints() {
        try {
            const response = await fetch('/api/endpoints');
            const data = await response.json();

            const tbody = document.getElementById('endpoints-table-body');
            tbody.innerHTML = '';

            data.endpoints.forEach((endpoint, index) => {
                const row = document.createElement('tr');
                row.dataset.index = index;
                row.addEventListener('click', () => this.selectEndpoint(endpoint));

                const statusIcon = endpoint.healthy ? '🟢' : '🔴';
                const requests = endpoint.stats ? endpoint.stats.totalRequests : 0;
                const failedRequests = endpoint.failedRequests || 0;  // Use new failedRequests field

                row.innerHTML =
                    '<td><span class="status-icon">' + statusIcon + '</span></td>' +
                    '<td>' + endpoint.name + '</td>' +
                    '<td>' + this.truncateUrl(endpoint.url, 25) + '</td>' +
                    '<td>' + endpoint.priority + '</td>' +
                    '<td>' + endpoint.responseTime + 'ms</td>' +
                    '<td>' + requests + '</td>' +
                    '<td>' + failedRequests + '</td>';

                tbody.appendChild(row);
            });

            // Auto-select first endpoint if none selected
            if (data.endpoints.length > 0 && !this.selectedEndpoint) {
                this.selectEndpoint(data.endpoints[0]);
            }

        } catch (error) {
            console.error('Error loading endpoints:', error);
        }
    }

    selectEndpoint(endpoint) {
        this.selectedEndpoint = endpoint;

        // Update table selection
        document.querySelectorAll('#endpoints-table-body tr').forEach(row => {
            row.classList.remove('selected');
        });

        // Find and highlight the selected row
        const rows = document.querySelectorAll('#endpoints-table-body tr');
        rows.forEach(row => {
            if (row.querySelector('td:nth-child(2)') &&
                row.querySelector('td:nth-child(2)').textContent === endpoint.name) {
                row.classList.add('selected');
            }
        });

        // Update details panel (now async)
        this.updateEndpointDetails(endpoint);
    }

    async updateEndpointDetails(endpoint) {
        const detailsContent = document.getElementById('endpoint-details-content');

        // Show loading state
        detailsContent.innerHTML = '<div class="loading">正在加载端点详情...</div>';

        try {
            // Fetch detailed endpoint information from new API
            const response = await fetch('/api/endpoints/details?name=' + encodeURIComponent(endpoint.name));
            const details = await response.json();

            this.renderEndpointDetails(details);
        } catch (error) {
            console.error('Error loading endpoint details:', error);
            // Fallback to basic details if API fails
            this.renderBasicEndpointDetails(endpoint);
        }
    }

    renderEndpointDetails(details) {
        const detailsContent = document.getElementById('endpoint-details-content');

        let html = '<h4 style="color: #60a5fa; margin-bottom: 15px;">🎯 ' + details.name + '</h4>';

        // Basic Info
        html += '<div class="metric"><span class="label">URL:</span><span class="value">' + details.url + '</span></div>';
        html += '<div class="metric"><span class="label">Priority:</span><span class="value">' + details.priority + '</span></div>';

        // Group information (similar to TUI)
        if (details.group) {
            html += '<div class="metric"><span class="label">Group:</span><span class="value">' + details.group + '</span></div>';
            if (details.groupPriority !== undefined) {
                html += '<div class="metric"><span class="label">Group Priority:</span><span class="value">' + details.groupPriority + '</span></div>';
            }
        }

        html += '<div class="metric"><span class="label">Timeout:</span><span class="value">' + details.timeout + '</span></div>';

        // Health Status
        const healthStatus = details.healthy ? 'Healthy' : 'Unhealthy';
        const healthColor = details.healthy ? '#10b981' : '#ef4444';
        html += '<div class="metric"><span class="label">Status:</span><span class="value" style="color: ' + healthColor + '">' + healthStatus + '</span></div>';
        html += '<div class="metric"><span class="label">Response Time:</span><span class="value">' + details.responseTime + 'ms</span></div>';
        html += '<div class="metric"><span class="label">Last Check:</span><span class="value">' + details.lastCheck + '</span></div>';

        // Performance Metrics (enhanced with detailed stats)
        if (details.stats && details.stats.totalRequests > 0) {
            html += '<h5 style="color: #fbbf24; margin: 15px 0 10px 0;">📊 Performance</h5>';
            html += '<div class="metric"><span class="label">Total Requests:</span><span class="value">' + details.stats.totalRequests.toLocaleString() + '</span></div>';
            html += '<div class="metric"><span class="label">Successful:</span><span class="value success">' + details.stats.successfulRequests.toLocaleString() + '</span></div>';
            html += '<div class="metric"><span class="label">Failed:</span><span class="value error">' + details.stats.failedRequests.toLocaleString() + '</span></div>';

            const successRate = details.stats.totalRequests > 0 ? (details.stats.successfulRequests / details.stats.totalRequests * 100) : 0;
            html += '<div class="metric"><span class="label">Success Rate:</span><span class="value success">' + successRate.toFixed(1) + '%</span></div>';

            html += '<div class="metric"><span class="label">Avg Response:</span><span class="value">' + details.stats.averageResponseTime + 'ms</span></div>';
            html += '<div class="metric"><span class="label">Min Response:</span><span class="value">' + details.stats.minResponseTime + 'ms</span></div>';
            html += '<div class="metric"><span class="label">Max Response:</span><span class="value">' + details.stats.maxResponseTime + 'ms</span></div>';

            // Token Usage (enhanced)
            const tokenUsage = details.stats.tokenUsage;
            const hasTokens = tokenUsage.inputTokens > 0 || tokenUsage.outputTokens > 0 || tokenUsage.cacheCreationTokens > 0 || tokenUsage.cacheReadTokens > 0;
            if (hasTokens) {
                html += '<h5 style="color: #a855f7; margin: 15px 0 10px 0;">🪙 Token Usage</h5>';
                html += '<div class="metric"><span class="label">📥 Input:</span><span class="value">' + tokenUsage.inputTokens.toLocaleString() + '</span></div>';
                html += '<div class="metric"><span class="label">📤 Output:</span><span class="value">' + tokenUsage.outputTokens.toLocaleString() + '</span></div>';
                if (tokenUsage.cacheCreationTokens > 0 || tokenUsage.cacheReadTokens > 0) {
                    html += '<div class="metric"><span class="label">🆕 Cache Create:</span><span class="value">' + tokenUsage.cacheCreationTokens.toLocaleString() + '</span></div>';
                    html += '<div class="metric"><span class="label">📖 Cache Read:</span><span class="value">' + tokenUsage.cacheReadTokens.toLocaleString() + '</span></div>';
                }
                const totalTokens = tokenUsage.inputTokens + tokenUsage.outputTokens;
                html += '<div class="metric"><span class="label">🔢 Total:</span><span class="value highlight">' + totalTokens.toLocaleString() + '</span></div>';
            }
        } else {
            html += '<h5 style="color: #fbbf24; margin: 15px 0 10px 0;">📊 Performance</h5>';
            html += '<p style="color: #64748b; font-style: italic;">No requests processed yet</p>';
        }

        // Headers (if any)
        if (details.headers && Object.keys(details.headers).length > 0) {
            html += '<h5 style="color: #06b6d4; margin: 15px 0 10px 0;">📋 Headers</h5>';
            Object.entries(details.headers).forEach(([key, value]) => {
                html += '<div class="metric"><span class="label">' + key + ':</span><span class="value" style="font-family: monospace; font-size: 0.9rem;">' + value + '</span></div>';
            });
        }

        detailsContent.innerHTML = html;
    }

    renderBasicEndpointDetails(endpoint) {
        // Fallback method using basic endpoint data (original implementation)
        const detailsContent = document.getElementById('endpoint-details-content');

        let html = '<h4 style="color: #60a5fa; margin-bottom: 15px;">🎯 ' + endpoint.name + '</h4>';
        html += '<div class="metric"><span class="label">URL:</span><span class="value">' + endpoint.url + '</span></div>';
        html += '<div class="metric"><span class="label">Priority:</span><span class="value">' + endpoint.priority + '</span></div>';

        const healthStatus = endpoint.healthy ? 'Healthy' : 'Unhealthy';
        const healthColor = endpoint.healthy ? '#10b981' : '#ef4444';
        html += '<div class="metric"><span class="label">Status:</span><span class="value" style="color: ' + healthColor + '">' + healthStatus + '</span></div>';
        html += '<div class="metric"><span class="label">Response Time:</span><span class="value">' + endpoint.responseTime + 'ms</span></div>';

        html += '<p style="color: #ef4444; font-style: italic; margin-top: 15px;">⚠️ Detailed information unavailable</p>';

        detailsContent.innerHTML = html;
    }

    async loadConnections() {
        try {
            const response = await fetch('/api/connections');
            const data = await response.json();

            document.getElementById('connections-active').textContent = data.activeCount;
            document.getElementById('connections-historical').textContent = data.historicalCount;

            const connectionsTableBody = document.getElementById('connections-table-body');
            connectionsTableBody.innerHTML = '';

            if (data.activeConnections && data.activeConnections.length > 0) {
                // Sort connections by start time (most recent first)
                const sortedConnections = data.activeConnections.sort((a, b) =>
                    new Date(b.startTime) - new Date(a.startTime)
                );

                // Show up to 15 connections (similar to TUI)
                sortedConnections.slice(0, 15).forEach(conn => {
                    const row = document.createElement('div');
                    row.className = 'connection-row';

                    // Determine connection status and styling
                    let statusClass = 'active';
                    if (conn.status === 'completed') statusClass = 'completed';
                    else if (conn.status === 'failed') statusClass = 'failed';
                    else if (conn.isStreaming) statusClass = 'streaming';

                    // Calculate duration
                    const duration = this.calculateConnectionDuration(conn.startTime);

                    // Get endpoint group information
                    const endpointDisplay = conn.endpoint || 'pending';
                    const groupName = this.getEndpointGroup(endpointDisplay);

                    // Format retry information
                    let retryDisplay = '';
                    if (conn.retryCount > 0) {
                        retryDisplay = conn.retryCount + '/3'; // Assuming max 3 retries
                    } else {
                        retryDisplay = '-';
                    }

                    row.innerHTML =
                        '<div class="conn-col-client">' +
                        '<span class="connection-status ' + statusClass + '"></span> ' +
                        this.truncateString(conn.clientIP, 12) +
                        '</div>' +
                        '<div class="conn-col-method">' + conn.method + '</div>' +
                        '<div class="conn-col-path">' + this.truncateString(conn.path, 18) + '</div>' +
                        '<div class="conn-col-endpoint">' + this.truncateString(endpointDisplay, 8) + '</div>' +
                        '<div class="conn-col-group">' + this.truncateString(groupName, 12) + '</div>' +
                        '<div class="conn-col-retry">' + retryDisplay + '</div>' +
                        '<div class="conn-col-duration">' + this.formatDurationShort(duration) + '</div>';

                    connectionsTableBody.appendChild(row);
                });

                // Fill remaining rows to maintain consistent height (similar to TUI)
                const remainingRows = Math.max(0, 15 - sortedConnections.length);
                for (let i = 0; i < remainingRows; i++) {
                    const emptyRow = document.createElement('div');
                    emptyRow.className = 'connection-row';
                    emptyRow.innerHTML =
                        '<div class="conn-col-client"></div>' +
                        '<div class="conn-col-method"></div>' +
                        '<div class="conn-col-path"></div>' +
                        '<div class="conn-col-endpoint"></div>' +
                        '<div class="conn-col-group"></div>' +
                        '<div class="conn-col-retry"></div>' +
                        '<div class="conn-col-duration"></div>';
                    connectionsTableBody.appendChild(emptyRow);
                }
            } else {
                // Show "No active connections" message
                const emptyRow = document.createElement('div');
                emptyRow.className = 'connection-row';
                emptyRow.innerHTML = '<div style="grid-column: 1 / -1; text-align: center; color: #64748b; font-style: italic;">无活动连接</div>';
                connectionsTableBody.appendChild(emptyRow);

                // Fill remaining rows
                for (let i = 0; i < 14; i++) {
                    const emptyRow = document.createElement('div');
                    emptyRow.className = 'connection-row';
                    emptyRow.innerHTML =
                        '<div class="conn-col-client"></div>' +
                        '<div class="conn-col-method"></div>' +
                        '<div class="conn-col-path"></div>' +
                        '<div class="conn-col-endpoint"></div>' +
                        '<div class="conn-col-group"></div>' +
                        '<div class="conn-col-retry"></div>' +
                        '<div class="conn-col-duration"></div>';
                    connectionsTableBody.appendChild(emptyRow);
                }
            }

        } catch (error) {
            console.error('Error loading connections:', error);
        }
    }

    calculateConnectionDuration(startTime) {
        const start = new Date(startTime);
        const now = new Date();
        return now - start;
    }

    getEndpointGroup(endpointName) {
        // This would ideally come from the endpoint data
        // For now, return a default group name
        if (endpointName === 'pending' || endpointName === 'unknown') {
            return 'Unknown';
        }
        // In a real implementation, you'd look up the endpoint's group
        return 'Default';
    }

    formatDurationShort(milliseconds) {
        if (milliseconds < 1000) {
            return milliseconds + 'ms';
        } else if (milliseconds < 60000) {
            return Math.floor(milliseconds / 1000) + 's';
        } else if (milliseconds < 3600000) {
            const minutes = Math.floor(milliseconds / 60000);
            const seconds = Math.floor((milliseconds % 60000) / 1000);
            return minutes + 'm' + (seconds > 0 ? seconds + 's' : '');
        } else {
            const hours = Math.floor(milliseconds / 3600000);
            const minutes = Math.floor((milliseconds % 3600000) / 60000);
            return hours + 'h' + (minutes > 0 ? minutes + 'm' : '');
        }
    }

    async loadLogs() {
        try {
            const response = await fetch('/api/logs');
            const data = await response.json();

            const logsContent = document.getElementById('logs-content');
            logsContent.innerHTML = '';

            if (data.logs && data.logs.length > 0) {
                // Display logs in reverse order (most recent first)
                const reversedLogs = data.logs.slice().reverse();
                
                reversedLogs.forEach(log => {
                    const div = document.createElement('div');
                    div.className = 'log-entry';

                    const levelClass = log.level.toLowerCase();
                    const levelText = log.level.substring(0, 3);

                    div.innerHTML =
                        '<span class="log-time">' + log.timestamp + '</span>' +
                        '<span class="log-level ' + levelClass + '">[' + levelText + ']</span>' +
                        '<span class="log-source">' + log.source + '</span>' +
                        '<span class="log-message">' + log.message + '</span>';

                    logsContent.appendChild(div);
                });
            } else {
                const div = document.createElement('div');
                div.innerHTML = '<p class="placeholder">暂无日志...</p>';
                logsContent.appendChild(div);
            }

        } catch (error) {
            console.error('Error loading logs:', error);
            const logsContent = document.getElementById('logs-content');
            logsContent.innerHTML = '<p class="placeholder" style="color: #ef4444;">加载日志失败: ' + error.message + '</p>';
        }
    }

    async loadConfig() {
        try {
            const response = await fetch('/api/config');
            const data = await response.json();

            // Server config
            document.getElementById('config-server').innerHTML =
                '<div class="metric"><span class="label">Host:</span><span class="value">' + data.server.host + '</span></div>' +
                '<div class="metric"><span class="label">Port:</span><span class="value">' + data.server.port + '</span></div>';

            // Strategy config
            document.getElementById('config-strategy').innerHTML =
                '<div class="metric"><span class="label">Type:</span><span class="value">' + data.strategy.type + '</span></div>' +
                '<div class="metric"><span class="label">Fast Test:</span><span class="value">' + (data.strategy.fastTestEnabled ? 'Enabled' : 'Disabled') + '</span></div>';

            // Auth config
            const authStatus = data.auth.enabled ? 'Enabled' : 'Disabled';
            const authColor = data.auth.enabled ? '#10b981' : '#ef4444';
            document.getElementById('config-auth').innerHTML =
                '<div class="metric"><span class="label">Status:</span><span class="value" style="color: ' + authColor + '">' + authStatus + '</span></div>';

            // Interface config
            document.getElementById('config-interface').innerHTML =
                '<div class="metric"><span class="label">TUI Update Interval:</span><span class="value">' + data.tui.updateInterval + '</span></div>' +
                '<div class="metric"><span class="label">WebUI Host:</span><span class="value">' + data.webui.host + '</span></div>' +
                '<div class="metric"><span class="label">WebUI Port:</span><span class="value">' + data.webui.port + '</span></div>';

            // Endpoints config
            let endpointsHtml = '';
            data.endpoints.forEach((ep, index) => {
                endpointsHtml +=
                    '<div class="metric">' +
                    '<span class="label">' + (index + 1) + '. ' + ep.name + ':</span>' +
                    '<span class="value">' + this.truncateUrl(ep.url, 30) + ' (P:' + ep.priority + ')</span>' +
                    '</div>';
            });
            document.getElementById('config-endpoints').innerHTML = endpointsHtml;

            // Load configuration management data
            await this.loadConfigs();

        } catch (error) {
            console.error('Error loading config:', error);
        }
    }

    // Utility functions
    formatUptime(seconds) {
        if (seconds < 60) {
            return Math.floor(seconds) + 's';
        } else if (seconds < 3600) {
            const minutes = Math.floor(seconds / 60);
            const secs = Math.floor(seconds % 60);
            return minutes + 'm' + secs + 's';
        } else if (seconds < 86400) {
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            return hours + 'h' + minutes + 'm';
        } else {
            const days = Math.floor(seconds / 86400);
            const hours = Math.floor((seconds % 86400) / 3600);
            return days + 'd' + hours + 'h';
        }
    }

    formatDuration(seconds) {
        if (seconds < 1) {
            return Math.floor(seconds * 1000) + 'ms';
        } else if (seconds < 60) {
            return seconds.toFixed(1) + 's';
        } else {
            const minutes = Math.floor(seconds / 60);
            const secs = Math.floor(seconds % 60);
            return minutes + 'm' + secs + 's';
        }
    }

    truncateString(str, maxLen) {
        if (str.length <= maxLen) {
            return str;
        }
        return str.substring(0, maxLen - 3) + '...';
    }

    truncateUrl(url, maxLen) {
        if (url.length <= maxLen) {
            return url;
        }

        // Try to preserve protocol and domain
        const protocolEnd = url.indexOf('://');
        if (protocolEnd === -1) {
            return this.truncateString(url, maxLen);
        }

        const domainStart = protocolEnd + 3;
        const pathStart = url.indexOf('/', domainStart);
        if (pathStart === -1) {
            return this.truncateString(url, maxLen);
        }

        const domain = url.substring(0, pathStart);
        const path = url.substring(pathStart);

        if (domain.length >= maxLen - 3) {
            return this.truncateString(url, maxLen);
        }

        const remaining = maxLen - domain.length - 3;
        if (remaining <= 0) {
            return domain + '...';
        }

        if (path.length <= remaining) {
            return url;
        }

        return domain + this.truncateString(path, remaining);
    }

    // Configuration Management Methods
    async loadConfigs() {
        try {
            // Load all configurations
            const configsResponse = await fetch('/api/configs');
            const configsData = await configsResponse.json();

            // Load active configuration
            const activeResponse = await fetch('/api/configs/active');
            const activeData = await activeResponse.json();

            // Update current config display
            const currentConfigName = document.getElementById('current-config-name');
            if (activeData.activeConfig) {
                currentConfigName.textContent = activeData.activeConfig.name;
                currentConfigName.style.color = '#10b981';
            } else {
                currentConfigName.textContent = '未知';
                currentConfigName.style.color = '#ef4444';
            }

            // Render config list
            this.renderConfigList(configsData.configs, activeData.activeConfig);

        } catch (error) {
            console.error('Error loading configs:', error);
            document.getElementById('current-config-name').textContent = '加载失败';
            document.getElementById('current-config-name').style.color = '#ef4444';
        }
    }

    renderConfigList(configs, activeConfig) {
        const configList = document.getElementById('config-list');

        if (!configs || configs.length === 0) {
            configList.innerHTML = '<p style="color: #94a3b8; text-align: center; padding: 20px;">暂无配置文件</p>';
            return;
        }

        let html = '';
        configs.forEach(config => {
            const isActive = activeConfig && activeConfig.name === config.name;
            const createdAt = new Date(config.createdAt).toLocaleString('zh-CN');

            html += ` + "`" + `
                <div class="config-item ` + "${isActive ? 'active' : ''}" + `">
                    <div class="config-info">
                        <div class="config-name">` + "${this.escapeHtml(config.name)}" + ` ` + "${isActive ? '(当前)' : ''}" + `</div>
                        <div class="config-details">
                            ` + "${this.escapeHtml(config.description)}" + ` • 创建于 ` + "${createdAt}" + `
                        </div>
                    </div>
                    <div class="config-actions">
                        <button class="switch-btn" onclick="app.switchConfig('` + "${this.escapeHtml(config.name)}" + `')"
                                ` + "${isActive ? 'disabled' : ''}" + `>
                            ` + "${isActive ? '当前配置' : '切换'}" + `
                        </button>
                        <button class="rename-btn" onclick="app.openConfigEditor('` + "${this.escapeHtml(config.name)}" + `')">编辑</button>
                        <button class="rename-btn" onclick="app.exportConfig('` + "${this.escapeHtml(config.name)}" + `')">导出</button>
                        <button class="rename-btn" onclick="app.renameConfig('` + "${this.escapeHtml(config.name)}" + `')">
                            重命名
                        </button>
                        <button class="delete-btn" onclick="app.deleteConfig('` + "${this.escapeHtml(config.name)}" + `')"
                                ` + "${isActive ? 'disabled' : ''}" + `>
                            删除
                        </button>
                    </div>
                </div>
            ` + "`" + `;
        });

        configList.innerHTML = html;
    }

    async importConfig() {
        const fileInput = document.getElementById('config-file');
        const nameInput = document.getElementById('config-name');

        const file = fileInput.files[0];
        const configName = nameInput.value.trim();

        if (!file) {
            this.showMessage('❌ 请选择配置文件', 'error');
            return;
        }

        if (!configName) {
            this.showMessage('❌ 请输入配置名称', 'error');
            return;
        }

        try {
            const formData = new FormData();
            formData.append('configFile', file);
            formData.append('configName', configName);

            const response = await fetch('/api/configs/import', {
                method: 'POST',
                body: formData
            });

            const result = await response.json();

            if (response.ok) {
                this.showMessage('✅ 配置导入成功', 'success');
                fileInput.value = '';
                nameInput.value = '';
                await this.loadConfigs();
            } else {
                this.showMessage('❌ 导入失败: ' + result.message, 'error');
            }

        } catch (error) {
            console.error('Error importing config:', error);
            this.showMessage('❌ 导入失败: ' + error.message, 'error');
        }
    }

    async switchConfig(configName) {
        if (!confirm('确定要切换到配置 "' + configName + '" 吗？')) {
            return;
        }

        try {
            const response = await fetch('/api/configs/switch', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ configName: configName })
            });

            const result = await response.json();

            if (response.ok) {
                this.showMessage('✅ 配置切换成功', 'success');
                await this.loadConfigs();

                // Wait a moment for backend configuration to fully switch
                await new Promise(resolve => setTimeout(resolve, 1000));

                // Force reload all tab data to reflect new configuration
                await this.loadOverview();
                await this.loadEndpoints();
                await this.loadConfig();

                // Also reload current tab data
                await this.loadTabData(this.currentTab);

                this.showMessage('🔄 数据已更新', 'success');
            } else {
                this.showMessage('❌ 切换失败: ' + result.message, 'error');
            }

        } catch (error) {
            console.error('Error switching config:', error);
            this.showMessage('❌ 切换失败: ' + error.message, 'error');
        }
    }

    async deleteConfig(configName) {
        if (!confirm('确定要删除配置 "' + configName + '" 吗？此操作不可撤销。')) {
            return;
        }

        try {
            const response = await fetch('/api/configs/delete', {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ configName: configName })
            });

            const result = await response.json();

            if (response.ok) {
                this.showMessage('✅ 配置删除成功', 'success');
                await this.loadConfigs();
            } else {
                this.showMessage('❌ 删除失败: ' + result.message, 'error');
            }

        } catch (error) {
            console.error('Error deleting config:', error);
            this.showMessage('❌ 删除失败: ' + error.message, 'error');
        }
    }

    async renameConfig(oldName) {
        const newName = prompt('请输入新的配置名称:', oldName);
        if (!newName || newName.trim() === '' || newName === oldName) {
            return;
        }

        try {
            const response = await fetch('/api/configs/rename', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    oldName: oldName,
                    newName: newName.trim()
                })
            });

            const result = await response.json();

            if (response.ok) {
                this.showMessage('✅ 配置重命名成功', 'success');
                await this.loadConfigs();
            } else {
                this.showMessage('❌ 重命名失败: ' + result.message, 'error');
            }

        } catch (error) {
            console.error('Error renaming config:', error);
            this.showMessage('❌ 重命名失败: ' + error.message, 'error');
        }
    }

    async openConfigEditor(name) {
        try {
            const resp = await fetch('/api/configs/content?name=' + encodeURIComponent(name));
            if (!resp.ok) {
                const t = await resp.text();
                this.showMessage('读取配置失败: ' + t, 'error');
                return;
            }
            const data = await resp.json();
            this.editingConfigName = name;
            document.getElementById('config-editor-title').textContent = '编辑配置: ' + name;
            document.getElementById('config-editor-content').value = data.content || '';
            document.getElementById('config-editor-error').style.display = 'none';
            document.getElementById('config-editor-modal').style.display = 'flex';
        } catch (e) {
            this.showMessage('读取配置失败: ' + e.message, 'error');
        }
    }

    closeConfigEditor() {
        document.getElementById('config-editor-modal').style.display = 'none';
        this.editingConfigName = null;
    }

    async saveConfigEditor() {
        const name = this.editingConfigName;
        const content = document.getElementById('config-editor-content').value;
        const errorBox = document.getElementById('config-editor-error');
        errorBox.style.display = 'none';
        try {
            const resp = await fetch('/api/configs/content', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name, content })
            });
            if (!resp.ok) {
                const msg = await resp.text();
                errorBox.textContent = msg;
                errorBox.style.display = 'block';
                return;
            }
            const result = await resp.json();
            this.showMessage('配置保存成功' + (result.active ? '（已实时生效）' : ''), 'success');
            this.closeConfigEditor();
            await this.loadConfigs();
        } catch (e) {
            errorBox.textContent = '保存失败: ' + e.message;
            errorBox.style.display = 'block';
        }
    }

    async exportConfig(name) {
        try {
            const resp = await fetch('/api/configs/export?name=' + encodeURIComponent(name));
            if (!resp.ok) {
                this.showMessage('导出失败', 'error');
                return;
            }
            const blob = await resp.blob();
            const a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = name + '.yaml';
            document.body.appendChild(a);
            a.click();
            a.remove();
            URL.revokeObjectURL(a.href);
        } catch (e) {
            this.showMessage('导出失败: ' + e.message, 'error');
        }
    }

    async exportAllConfigs() {
        try {
            const resp = await fetch('/api/configs/export-all');
            if (!resp.ok) {
                this.showMessage('批量导出失败', 'error');
                return;
            }
            const blob = await resp.blob();
            const a = document.createElement('a');
            a.href = URL.createObjectURL(blob);
            a.download = 'configs_' + Date.now() + '.zip';
            document.body.appendChild(a);
            a.click();
            a.remove();
            URL.revokeObjectURL(a.href);
        } catch (e) {
            this.showMessage('批量导出失败: ' + e.message, 'error');
        }
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}

// Initialize the app when DOM is loaded
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new WebUIApp();
});
`
