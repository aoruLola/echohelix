package dashboard

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"echohelix/bridge/internal/auth"
)

// Handler handles Dashboard requests
type Handler struct {
	logger      *Logger
	authService *auth.Service
	tmpl        *template.Template
}

// NewHandler creates a new Dashboard handler
func NewHandler(logger *Logger, authService *auth.Service) *Handler {
	h := &Handler{
		logger:      logger,
		authService: authService,
	}
	h.tmpl = template.Must(template.New("dashboard").Parse(dashboardHTML))
	return h
}

// HandleDashboard renders the dashboard page
func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	pc := h.authService.GetActivePairingCode()
	var code string
	var expiresIn int64
	if pc != nil {
		code = pc.Code
		expiresIn = int64(time.Until(pc.ExpiresAt).Seconds())
	}

	data := map[string]interface{}{
		"PairingCode": code,
		"ExpiresIn":   expiresIn,
	}

	h.tmpl.Execute(w, data)
}

// HandleGetLogs returns log data
func (h *Handler) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	countStr := r.URL.Query().Get("count")
	count := 100
	if countStr != "" {
		if n, err := strconv.Atoi(countStr); err == nil && n > 0 {
			count = n
		}
	}

	logs := h.logger.GetLogs(count)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"logs":  logs,
		"total": h.logger.Count(),
	})
}

// HandleRefreshPairingCode refreshes the pairing code
func (h *Handler) HandleRefreshPairingCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pc, err := h.authService.GeneratePairingCode()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":       pc.Code,
		"expires_at": pc.ExpiresAt,
		"expires_in": int64(time.Until(pc.ExpiresAt).Seconds()),
	})
}

// Minimal HTML template
const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>EchoHelix Dashboard</title>
    <style>
        body { font-family: monospace; margin: 20px; background: #000; color: #0f0; }
        h1 { border-bottom: 2px solid #0f0; padding-bottom: 10px; }
        .section { margin: 30px 0; padding: 20px; border: 1px solid #0f0; }
        .code { font-size: 32px; letter-spacing: 8px; text-align: center; margin: 20px 0; }
        .timer { text-align: center; margin: 10px 0; }
        button { background: #0f0; color: #000; border: none; padding: 10px 20px; font-size: 14px; cursor: pointer; font-family: monospace; }
        button:hover { background: #0a0; }
        #logs { font-size: 12px; height: 400px; overflow-y: scroll; border: 1px solid #0f0; padding: 10px; }
        .log-entry { margin: 2px 0; }
        .info { color: #0ff; }
        .warn { color: #ff0; }
        .error { color: #f00; }
    </style>
</head>
<body>
    <h1>🌊 EchoHelix Bridge Dashboard</h1>
    
    <div class="section">
        <h2>📱 配对码</h2>
        <div class="code" id="code">{{.PairingCode}}</div>
        <div class="timer">剩余: <span id="timer">--:--</span></div>
        <center><button onclick="refresh()">🔄 刷新</button></center>
    </div>

    <div class="section">
        <h2>📋 服务器日志 <button onclick="loadLogs()" style="float:right">刷新</button></h2>
        <div id="logs">加载中...</div>
    </div>

    <script>
        let countdown = {{.ExpiresIn}};

        function updateTimer() {
            if (countdown <= 0) {
                document.getElementById('timer').textContent = '已过期';
                return;
            }
            const m = Math.floor(countdown / 60);
            const s = countdown % 60;
            document.getElementById('timer').textContent = m + ':' + s.toString().padStart(2, '0');
            countdown--;
        }

        async function refresh() {
            const res = await fetch('/dashboard/pairing/refresh', { method: 'POST' });
            const data = await res.json();
            if (data.code) {
                document.getElementById('code').textContent = data.code;
                countdown = data.expires_in;
                updateTimer();
            }
        }

        async function loadLogs() {
            const res = await fetch('/dashboard/logs?count=100');
            const data = await res.json();
            const container = document.getElementById('logs');
            if (!data.logs || data.logs.length === 0) {
                container.innerHTML = '暂无日志';
                return;
            }
            container.innerHTML = data.logs.map(log => {
                const time = new Date(log.timestamp).toLocaleTimeString();
                const cls = log.level.toLowerCase();
                return '<div class="log-entry ' + cls + '">[' + time + '] ' + log.level + ': ' + log.message + '</div>';
            }).join('');
        }

        setInterval(updateTimer, 1000);
        setInterval(loadLogs, 3000);
        loadLogs();
        updateTimer();
    </script>
</body>
</html>`
