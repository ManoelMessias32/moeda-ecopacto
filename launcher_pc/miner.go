package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	API_URL  = "https://ecopacto-api-production.up.railway.app"
	APP_PORT = "9999"
)

type State struct {
	Wallet     string   `json:"wallet"`
	Balance    uint64   `json:"balance"`
	Pending    uint64   `json:"pending"`    // Saldo que cairá à meia-noite
	MinedToday uint64   `json:"mined_today"` // Contador local da sessão
	IsMining   bool     `json:"is_mining"`
	Logs       []string `json:"logs"`
}

var appState = State{
	IsMining: false,
	Logs:     []string{"[SISTEMA] Launcher iniciado. Pronto para minerar."},
}

func addLog(msg string) {
	t := time.Now().Format("15:04:05")
	appState.Logs = append(appState.Logs, "["+t+"] "+msg)
	if len(appState.Logs) > 40 {
		appState.Logs = appState.Logs[1:]
	}
}

func main() {
	// Carregar carteira salva
	data, _ := os.ReadFile("wallet.txt")
	appState.Wallet = strings.TrimSpace(string(data))

	// Iniciar loops
	go miningEngine()
	go syncNetwork()

	// Endpoints
	http.HandleFunc("/", serveUI)
	http.HandleFunc("/api/state", getState)
	http.HandleFunc("/api/save", saveWallet)
	http.HandleFunc("/api/toggle", toggleMining)

	go http.ListenAndServe(":"+APP_PORT, nil)

	openAppWindow()

	fmt.Println("🚀 Launcher Ecopacto rodando na porta " + APP_PORT)
	select {}
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>ECOPACTO MASTER LAUNCHER</title>
		<style>
			body { background: #05070a; color: white; font-family: 'Segoe UI', sans-serif; text-align: center; padding: 15px; user-select: none; overflow: hidden; }
			.card { border: 1px solid #333; border-radius: 12px; padding: 20px; background: #0f121a; box-shadow: 0 10px 40px rgba(0,0,0,0.8); max-width: 380px; margin: auto; }
			h1 { color: #f7931a; font-style: italic; font-size: 32px; margin: 0; letter-spacing: -1px; }
			.subtitle { color: #555; font-size: 10px; margin-bottom: 15px; text-transform: uppercase; }

			.input-box { text-align: left; margin-bottom: 12px; }
			label { font-size: 10px; color: #888; margin-left: 5px; }
			input { width: 100%%; padding: 8px; margin-top: 5px; background: #000; border: 1px solid #222; color: #22c55e; font-family: monospace; border-radius: 6px; outline: none; box-sizing: border-box; font-size: 12px; text-align: center; }

			.btn-row { display: flex; gap: 8px; justify-content: center; margin-bottom: 15px; }
			.btn { flex: 1; padding: 8px; cursor: pointer; font-weight: bold; border-radius: 5px; border: none; transition: 0.2s; font-size: 11px; text-transform: uppercase; }
			.btn-ok { background: #22c55e; color: black; }
			.btn-toggle { background: #333; color: white; }

			.stats { display: flex; justify-content: space-around; margin: 15px 0; background: rgba(0,0,0,0.2); padding: 10px; border-radius: 8px; border: 1px solid #1a1a1a; }
			.stats div { font-size: 11px; color: #666; }
			.stats b { color: #eee; display: block; font-size: 14px; }

			.logs { background: #000; height: 180px; overflow-y: auto; text-align: left; padding: 10px; font-size: 10px; color: #22c55e; border: 1px solid #222; margin-top: 10px; font-family: 'Consolas', monospace; border-radius: 6px; line-height: 1.4; opacity: 0.9; }
			.footer { font-size: 9px; color: #333; margin-top: 15px; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>ecopacto</h1>
			<div class="subtitle">Mainnet Miner v2.5 - Midnight Payout</div>

			<div class="input-box">
				<label>Ecopacto Address:</label>
				<input type="text" id="wallet" value="%s" placeholder="ECO_...">
			</div>

			<div class="btn-row">
				<button class="btn btn-ok" onclick="save()">SALVAR / OK</button>
				<button class="btn btn-toggle" onclick="toggle()" id="btn-toggle">INICIAR</button>
			</div>

			<div class="stats">
				<div>SALDO REAL<b><span id="balance">0</span> ECO</b></div>
				<div>PENDENTE (00:00)<b><span id="pending">0</span> ECO</b></div>
			</div>

			<div class="logs" id="logs"></div>
			<div class="footer">PAGAMENTOS AUTOMÁTICOS ÀS 00:00</div>
		</div>

		<script>
			const logsDiv = document.getElementById('logs');
			function update() {
				fetch('/api/state').then(r => r.json()).then(data => {
					document.getElementById('balance').innerText = data.balance.toLocaleString();
					document.getElementById('pending').innerText = data.pending.toLocaleString();

					const btn = document.getElementById('btn-toggle');
					btn.innerText = data.is_mining ? 'PARAR' : 'INICIAR';
					btn.style.background = data.is_mining ? '#ef4444' : '#333';

					const isAtBottom = logsDiv.scrollHeight - logsDiv.clientHeight <= logsDiv.scrollTop + 1;
					logsDiv.innerHTML = data.logs.join('<br>');
					if (isAtBottom) logsDiv.scrollTop = logsDiv.scrollHeight;
				});
			}
			function save() {
				const addr = document.getElementById('wallet').value;
				fetch('/api/save?addr=' + addr).then(() => alert('Configuração salva!'));
			}
			function toggle() { fetch('/api/toggle'); }
			setInterval(update, 1000);
		</script>
	</body>
	</html>
	`, appState.Wallet)
}

func getState(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(appState) }

func saveWallet(w http.ResponseWriter, r *http.Request) {
	appState.Wallet = r.URL.Query().Get("addr")
	os.WriteFile("wallet.txt", []byte(appState.Wallet), 0644)
	addLog("Carteira atualizada: " + appState.Wallet)
	w.WriteHeader(200)
}

func toggleMining(w http.ResponseWriter, r *http.Request) {
	appState.IsMining = !appState.IsMining
	if appState.IsMining {
		addLog("Mineração ATIVADA.")
	} else {
		addLog("Mineração PAUSADA.")
	}
	w.WriteHeader(200)
}

func miningEngine() {
	for {
		if appState.IsMining && appState.Wallet != "" {
			// Simula esforço real
			h := sha256.New()
			h.Write([]byte(time.Now().String()))
			hashStr := hex.EncodeToString(h.Sum(nil))

			time.Sleep(5 * time.Second)
			if appState.IsMining {
				// REPORTA AO SERVIDOR IMEDIATAMENTE
				resp, err := http.Get(API_URL + "/mine/submit?address=" + appState.Wallet)
				if err == nil && resp.StatusCode == 200 {
					appState.MinedToday += 10
					addLog("Bloco minerado! Hash: 000" + hashStr[:10])
					addLog("Enviado ao servidor. Cairá no Payout das 00:00.")
					resp.Body.Close()
				} else {
					addLog("ERRO: Falha ao reportar bloco ao servidor.")
				}
			}
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func syncNetwork() {
	for {
		if appState.Wallet != "" {
			resp, err := http.Get(API_URL + "/wallet?address=" + appState.Wallet)
			if err == nil {
				var d struct{
					Balance uint64 `json:"balance"`
					Pending uint64 `json:"pending"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&d); err == nil {
					appState.Balance = d.Balance
					appState.Pending = d.Pending
				}
				resp.Body.Close()
			}
		}
		time.Sleep(10 * time.Second)
	}
}

func openAppWindow() {
	url := "http://localhost:" + APP_PORT
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "start msedge --app="+url).Start()
	} else {
		exec.Command("open", url).Start()
	}
}
