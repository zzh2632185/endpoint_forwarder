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
            <div class="status-bar">
                <span id="status-requests">Requests: 0</span>
                <span id="status-success">Success: 0.0%</span>
                <span id="status-connections">Connections: 0</span>
                <span id="last-update">Last Update: --:--:--</span>
            </div>
        </header>

        <nav class="nav-tabs">
            <button class="tab-button active" data-tab="overview">📊 Overview</button>
            <button class="tab-button" data-tab="endpoints">🎯 Endpoints</button>
            <button class="tab-button" data-tab="connections">🔌 Connections</button>
            <button class="tab-button" data-tab="logs">📝 Logs</button>
            <button class="tab-button" data-tab="config">⚙️ Config</button>
        </nav>

        <main class="main-content">
            <!-- Overview Tab -->
            <div id="overview" class="tab-content active">
                <div class="grid-2x2">
                    <div class="card">
                        <h3>📊 Request Metrics</h3>
                        <div id="metrics-content">
                            <div class="metric">
                                <span class="label">Total Requests:</span>
                                <span class="value" id="total-requests">0</span>
                            </div>
                            <div class="metric">
                                <span class="label">Successful:</span>
                                <span class="value success" id="successful-requests">0 (0.0%)</span>
                            </div>
                            <div class="metric">
                                <span class="label">Failed:</span>
                                <span class="value error" id="failed-requests">0 (0.0%)</span>
                            </div>
                            <div class="metric">
                                <span class="label">Avg Response Time:</span>
                                <span class="value" id="avg-response-time">0ms</span>
                            </div>
                            <div class="token-section">
                                <h4>🪙 Token Usage</h4>
                                <div class="metric">
                                    <span class="label">📥 Input Tokens:</span>
                                    <span class="value" id="input-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">📤 Output Tokens:</span>
                                    <span class="value" id="output-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">🆕 Cache Creation:</span>
                                    <span class="value" id="cache-creation-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">📖 Cache Read:</span>
                                    <span class="value" id="cache-read-tokens">0</span>
                                </div>
                                <div class="metric">
                                    <span class="label">🔢 Total Tokens:</span>
                                    <span class="value highlight" id="total-tokens">0</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <div class="card">
                        <h3>🪙 Historical Token Usage</h3>
                        <div id="token-history-content">
                            <p class="placeholder">Past connections with token consumption:</p>
                            <div id="token-history-list">
                                <div class="history-item">
                                    <span class="history-placeholder">No connections with token usage yet...</span>
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
                        <h3>🎯 Endpoints</h3>
                        <table id="endpoints-table">
                            <thead>
                                <tr>
                                    <th>Status</th>
                                    <th>Name</th>
                                    <th>URL</th>
                                    <th>Priority</th>
                                    <th>Response Time</th>
                                    <th>Requests</th>
                                </tr>
                            </thead>
                            <tbody id="endpoints-table-body">
                                <tr>
                                    <td colspan="6" class="placeholder">Loading endpoints...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                    <div class="endpoint-details">
                        <h3>📊 Details</h3>
                        <div id="endpoint-details-content">
                            <p class="placeholder">Select an endpoint to view details</p>
                        </div>
                    </div>
                </div>
            </div>

            <!-- Connections Tab -->
            <div id="connections" class="tab-content">
                <div class="card">
                    <h3>🔌 Active Connections</h3>
                    <div id="connections-stats">
                        <div class="metric">
                            <span class="label">Active:</span>
                            <span class="value" id="connections-active">0</span>
                            <span class="label">Historical:</span>
                            <span class="value" id="connections-historical">0</span>
                        </div>
                    </div>
                    <div id="connections-list">
                        <p class="placeholder">No active connections</p>
                    </div>
                </div>
            </div>

            <!-- Logs Tab -->
            <div id="logs" class="tab-content">
                <div class="card">
                    <h3>📝 System Logs</h3>
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
                        <h3>🌐 Server</h3>
                        <div id="config-server"></div>
                    </div>
                    <div class="card">
                        <h3>🎯 Strategy</h3>
                        <div id="config-strategy"></div>
                    </div>
                    <div class="card">
                        <h3>🔐 Authentication</h3>
                        <div id="config-auth"></div>
                    </div>
                    <div class="card">
                        <h3>🖥️ Interface</h3>
                        <div id="config-interface"></div>
                    </div>
                    <div class="card full-width">
                        <h3>🎯 Endpoints</h3>
                        <div id="config-endpoints"></div>
                    </div>
                </div>
            </div>
        </main>
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
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

.header {
    text-align: center;
    margin-bottom: 30px;
    padding: 20px;
    background: linear-gradient(135deg, #1e293b, #334155);
    border-radius: 12px;
    border: 1px solid #334155;
}

.header h1 {
    color: #60a5fa;
    margin-bottom: 15px;
    font-size: 2rem;
}

.status-bar {
    display: flex;
    justify-content: center;
    gap: 30px;
    flex-wrap: wrap;
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
}

.card h3 {
    color: #60a5fa;
    margin-bottom: 15px;
    font-size: 1.1rem;
}

.grid-2x2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
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
`

// appJS contains the JavaScript application code
const appJS = `
class WebUIApp {
    constructor() {
        this.currentTab = 'overview';
        this.selectedEndpoint = null;
        this.eventSource = null;
        this.init();
    }

    init() {
        this.setupTabs();
        this.setupEventSource();
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

    updateStatusBar(data) {
        document.getElementById('status-requests').textContent = 'Requests: ' + data.totalRequests;
        document.getElementById('status-success').textContent = 'Success: ' + data.successRate.toFixed(1) + '%';
        document.getElementById('status-connections').textContent = 'Connections: ' + data.activeConnections;
        document.getElementById('last-update').textContent = 'Last Update: ' + new Date().toLocaleTimeString();
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

            // Update token history
            this.updateTokenHistory(data.connectionHistory);

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
            div.innerHTML = '<span class="history-placeholder">No connections with token usage yet...</span>';
            historyList.appendChild(div);
        }
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

                row.innerHTML =
                    '<td><span class="status-icon">' + statusIcon + '</span></td>' +
                    '<td>' + endpoint.name + '</td>' +
                    '<td>' + this.truncateUrl(endpoint.url, 25) + '</td>' +
                    '<td>' + endpoint.priority + '</td>' +
                    '<td>' + endpoint.responseTime + 'ms</td>' +
                    '<td>' + requests + '</td>';

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
        event.currentTarget.classList.add('selected');

        // Update details panel
        this.updateEndpointDetails(endpoint);
    }

    updateEndpointDetails(endpoint) {
        const detailsContent = document.getElementById('endpoint-details-content');

        let html = '<h4 style="color: #60a5fa; margin-bottom: 15px;">🎯 ' + endpoint.name + '</h4>';

        // Basic Info
        html += '<div class="metric"><span class="label">URL:</span><span class="value">' + endpoint.url + '</span></div>';
        html += '<div class="metric"><span class="label">Priority:</span><span class="value">' + endpoint.priority + '</span></div>';
        html += '<div class="metric"><span class="label">Timeout:</span><span class="value">' + endpoint.timeout + '</span></div>';

        // Health Status
        const healthStatus = endpoint.healthy ? 'Healthy' : 'Unhealthy';
        const healthColor = endpoint.healthy ? '#10b981' : '#ef4444';
        html += '<div class="metric"><span class="label">Status:</span><span class="value" style="color: ' + healthColor + '">' + healthStatus + '</span></div>';
        html += '<div class="metric"><span class="label">Response Time:</span><span class="value">' + endpoint.responseTime + 'ms</span></div>';
        html += '<div class="metric"><span class="label">Consecutive Fails:</span><span class="value">' + endpoint.consecutiveFails + '</span></div>';
        html += '<div class="metric"><span class="label">Last Check:</span><span class="value">' + endpoint.lastCheck + '</span></div>';

        // Performance Metrics
        if (endpoint.stats && endpoint.stats.totalRequests > 0) {
            html += '<h5 style="color: #fbbf24; margin: 15px 0 10px 0;">📊 Performance</h5>';
            html += '<div class="metric"><span class="label">Total Requests:</span><span class="value">' + endpoint.stats.totalRequests.toLocaleString() + '</span></div>';
            html += '<div class="metric"><span class="label">Success Rate:</span><span class="value success">' + endpoint.stats.successRate.toFixed(1) + '%</span></div>';
            html += '<div class="metric"><span class="label">Retries:</span><span class="value">' + endpoint.stats.retryCount + '</span></div>';
            html += '<div class="metric"><span class="label">Avg Response:</span><span class="value">' + endpoint.stats.avgResponseTime + 'ms</span></div>';
            html += '<div class="metric"><span class="label">Min/Max Response:</span><span class="value">' + endpoint.stats.minResponseTime + 'ms / ' + endpoint.stats.maxResponseTime + 'ms</span></div>';
            html += '<div class="metric"><span class="label">Last Used:</span><span class="value">' + endpoint.stats.lastUsed + '</span></div>';

            // Token Usage
            const tokenUsage = endpoint.stats.tokenUsage;
            const hasTokens = tokenUsage.inputTokens > 0 || tokenUsage.outputTokens > 0 || tokenUsage.cacheCreationTokens > 0 || tokenUsage.cacheReadTokens > 0;
            if (hasTokens) {
                html += '<h5 style="color: #a855f7; margin: 15px 0 10px 0;">🪙 Tokens</h5>';
                html += '<div class="metric"><span class="label">📥 Input:</span><span class="value">' + tokenUsage.inputTokens.toLocaleString() + '</span></div>';
                html += '<div class="metric"><span class="label">📤 Output:</span><span class="value">' + tokenUsage.outputTokens.toLocaleString() + '</span></div>';
                if (tokenUsage.cacheCreationTokens > 0 || tokenUsage.cacheReadTokens > 0) {
                    html += '<div class="metric"><span class="label">🆕 Cache Create:</span><span class="value">' + tokenUsage.cacheCreationTokens.toLocaleString() + '</span></div>';
                    html += '<div class="metric"><span class="label">📖 Cache Read:</span><span class="value">' + tokenUsage.cacheReadTokens.toLocaleString() + '</span></div>';
                }
            }
        } else {
            html += '<h5 style="color: #fbbf24; margin: 15px 0 10px 0;">📊 Performance</h5>';
            html += '<p style="color: #64748b; font-style: italic;">No requests processed yet</p>';
        }

        detailsContent.innerHTML = html;
    }

    async loadConnections() {
        try {
            const response = await fetch('/api/connections');
            const data = await response.json();

            document.getElementById('connections-active').textContent = data.activeCount;
            document.getElementById('connections-historical').textContent = data.historicalCount;

            const connectionsList = document.getElementById('connections-list');
            connectionsList.innerHTML = '';

            if (data.activeConnections && data.activeConnections.length > 0) {
                data.activeConnections.forEach(conn => {
                    const div = document.createElement('div');
                    div.className = 'connection-item';

                    div.innerHTML =
                        '<div class="connection-info">' +
                        '<span style="color: #60a5fa">' + conn.clientIP + '</span>' +
                        '<span>' + conn.method + '</span>' +
                        '<span>' + this.truncateString(conn.path, 20) + '</span>' +
                        '<span style="color: #fbbf24">→ ' + conn.endpoint + '</span>' +
                        '<span style="color: #94a3b8">' + conn.retryInfo + '</span>' +
                        '</div>' +
                        '<div class="connection-duration">' + this.formatDuration(conn.duration) + '</div>';

                    connectionsList.appendChild(div);
                });
            } else {
                const div = document.createElement('div');
                div.innerHTML = '<p class="placeholder">No active connections</p>';
                connectionsList.appendChild(div);
            }

        } catch (error) {
            console.error('Error loading connections:', error);
        }
    }

    async loadLogs() {
        try {
            const response = await fetch('/api/logs');
            const data = await response.json();

            const logsContent = document.getElementById('logs-content');
            logsContent.innerHTML = '';

            if (data.logs && data.logs.length > 0) {
                data.logs.forEach(log => {
                    const div = document.createElement('div');
                    div.className = 'log-entry';

                    const levelClass = log.level.toLowerCase();

                    div.innerHTML =
                        '<span class="log-time">' + log.timestamp + '</span>' +
                        '<span class="log-level ' + levelClass + '">[' + log.level.substring(0, 3) + ']</span>' +
                        '<span class="log-source">' + log.source + '</span>' +
                        '<span class="log-message">' + log.message + '</span>';

                    logsContent.appendChild(div);
                });
            } else {
                const div = document.createElement('div');
                div.innerHTML = '<p class="placeholder">No logs available</p>';
                logsContent.appendChild(div);
            }

        } catch (error) {
            console.error('Error loading logs:', error);
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
}

// Initialize the app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new WebUIApp();
});
`
