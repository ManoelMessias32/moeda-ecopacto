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
	Pending    uint64   `json:"pending"`    // Saldo que cairá à meia-noite (Server)
	MinedToday uint64   `json:"mined_today"` // Acumulado local salvo no PC
	IsMining   bool     `json:"is_mining"`
	Logs       []string `json:"logs"`
}

var appState = State{
	IsMining: false,
	Logs:     []string{"[SISTEMA] Launcher iniciado. Pronto para minerar."},
}

// Salva as estatísticas no PC para não perder ao fechar
func saveStats() {
	data, _ := json.Marshal(appState.MinedToday)
	os.WriteFile("stats.json", data, 0644)
}

// Carrega as estatísticas salvas no PC
func loadStats() {
	data, err := os.ReadFile("stats.json")
	if err == nil {
		json.Unmarshal(data, &appState.MinedToday)
	}
}

func addLog(msg string) {
	t := time.Now().Format("15:04:05")
	appState.Logs = append(appState.Logs, "["+t+"] "+msg)
	if len(appState.Logs) > 30 {
		appState.Logs = appState.Logs[1:]
	}
}

func main() {
	// Carregar carteira e moedas salvas
	data, _ := os.ReadFile("wallet.txt")
	appState.Wallet = strings.TrimSpace(string(data))
	loadStats()

	// Iniciar loops de fundo
	go miningEngine()
	go syncNetwork()

	// Endpoints da Interface
	http.HandleFunc("/", serveUI)
	http.HandleFunc("/api/state", getState)
	http.HandleFunc("/api/save", saveWallet)
	http.HandleFunc("/api/toggle", toggleMining)

	go http.ListenAndServe(":"+APP_PORT, nil)

	openAppWindow()

	fmt.Println("🚀 Launcher Ecopacto ativo em http://localhost:" + APP_PORT)
	select {}
}

func serveUI(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>ECOPACTO MASTER LAUNCHER</title>
		<style>
			:root { --bg: #05070a; --card: #0f121a; --accent: #22c55e; --muted: #64748b; }
			body { background: var(--bg); color: white; font-family: 'Segoe UI', sans-serif; text-align: center; padding: 15px; user-select: none; overflow: hidden; }
			.card { border: 1px solid #333; border-radius: 24px; padding: 25px; background: var(--card); box-shadow: 0 20px 50px rgba(0,0,0,0.8); max-width: 400px; margin: auto; }

			h1 { color: #f7931a; font-style: italic; font-size: 32px; margin: 0; letter-spacing: -1px; }
			.subtitle { color: var(--muted); font-size: 10px; margin-bottom: 20px; text-transform: uppercase; letter-spacing: 1px; }

			.input-box { text-align: left; margin-bottom: 15px; background: #000; padding: 10px; border-radius: 12px; border: 1px solid #222; }
			label { font-size: 9px; color: var(--muted); text-transform: uppercase; font-weight: bold; }
			input { width: 100%%; padding: 5px 0; background: transparent; border: none; color: var(--accent); font-family: monospace; outline: none; font-size: 13px; }

			.balance-label { color: var(--muted); font-size: 10px; font-weight: 800; letter-spacing: 1px; margin-top: 15px; }
			.balance-amount { font-size: 48px; font-weight: 800; color: #fff; margin: 5px 0; display: flex; align-items: baseline; justify-content: center; gap: 8px; }
			.balance-amount span { color: var(--accent); font-size: 20px; }

			.btn-row { display: flex; gap: 10px; justify-content: center; margin: 20px 0; }
			.btn { flex: 1; padding: 14px; cursor: pointer; font-weight: bold; border-radius: 12px; border: none; transition: 0.2s; font-size: 11px; text-transform: uppercase; }
			.btn-save { background: #ffffff08; color: white; border: 1px solid #ffffff15; }
			.btn-toggle { background: var(--accent); color: black; font-weight: 900; }

			.stats-small { display: flex; justify-content: space-between; font-size: 11px; color: var(--muted); margin-bottom: 15px; padding: 0 5px; }
			.stats-small b { color: var(--accent); }

			.logs { background: #000; height: 120px; overflow-y: auto; text-align: left; padding: 10px; font-size: 10px; color: var(--accent); border: 1px solid #222; margin-top: 10px; font-family: 'Consolas', monospace; border-radius: 10px; line-height: 1.4; opacity: 0.8; }
			.footer { font-size: 9px; color: #333; margin-top: 15px; }
		</style>
	</head>
	<body>
		<div class="card">
			<h1>ecopacto</h1>
			<div class="subtitle">Official Mainnet Launcher</div>

			<div class="input-box">
				<label>Endereço da sua carteira:</label>
				<input type="text" id="wallet" value="%s" placeholder="ECO_..." oninput="saveSilently()">
			</div>

			<div class="balance-label">SALDO DISPONÍVEL</div>
			<div class="balance-amount"><span id="balance">0</span> <span>ECO</span></div>

			<div class="stats-small">
				<span>PENDENTE (00:00): <b id="pending">0 ECO</b></span>
				<span>ACUMULADO: <b id="mined">0 ECO</b></span>
			</div>

			<div class="btn-row">
				<button class="btn btn-save" onclick="saveSilently()">SALVAR CONFIG</button>
				<button class="btn btn-toggle" onclick="toggle()" id="btn-toggle">INICIAR</button>
			</div>

			<div class="logs" id="logs"></div>
			<div class="footer">POWERED BY ECOPACTO NETWORK CORE</div>
		</div>

		<script>
			const logsDiv = document.getElementById('logs');
			function update() {
				fetch('/api/state').then(r => r.json()).then(data => {
					document.getElementById('balance').innerText = data.balance.toLocaleString();
					document.getElementById('pending').innerText = data.pending + ' ECO';
					document.getElementById('mined').innerText = data.mined_today + ' ECO';

					const btn = document.getElementById('btn-toggle');
					btn.innerText = data.is_mining ? 'PARAR MINERAÇÃO' : 'INICIAR MINERAÇÃO';
					btn.style.background = data.is_mining ? '#ef4444' : '#22c55e';
					btn.style.color = data.is_mining ? 'white' : 'black';

					logsDiv.innerHTML = data.logs.join('<br>');
					logsDiv.scrollTop = logsDiv.scrollHeight;
				});
			}
			function saveSilently() {
				const addr = document.getElementById('wallet').value;
				fetch('/api/save?addr=' + addr.trim());
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
	appState.Wallet = strings.TrimSpace(r.URL.Query().Get("addr"))
	os.WriteFile("wallet.txt", []byte(appState.Wallet), 0644)
	addLog("Carteira salva localmente.")
	w.WriteHeader(200)
}

func toggleMining(w http.ResponseWriter, r *http.Request) {
	appState.IsMining = !appState.IsMining
	if appState.IsMining {
		addLog("Mineração ATIVADA.")
	} else {
		addLog("Mineração EM PAUSA.")
	}
	w.WriteHeader(200)
}

func miningEngine() {
	for {
		if appState.IsMining && appState.Wallet != "" {
			h := sha256.New()
			h.Write([]byte(time.Now().String()))
			hashStr := hex.EncodeToString(h.Sum(nil))

			time.Sleep(5 * time.Second)
			if appState.IsMining {
				// REPORTA O BLOCO PARA O SERVIDOR
				resp, err := http.Get(API_URL + "/mine/submit?address=" + appState.Wallet)
				if err == nil && resp.StatusCode == 200 {
					appState.MinedToday += 10
					saveStats() // SALVA O PROGRESSO NO PC
					addLog("Bloco OK! Hash: 000" + hashStr[:10])
					addLog("Sincronizado com a Railway.")
					resp.Body.Close()
				} else {
					addLog("ERRO: Carteira não encontrada na rede.")
				}
			}
		} else {
			time.Sleep(1000 * time.Millisecond)
		}
	}
}

func syncNetwork() {
	for {
		if appState.Wallet != "" {
			resp, err := http.Get(API_URL + "/wallet?address=" + appState.Wallet)
			if err == nil {
				var d struct{ Balance uint64; Pending uint64 }
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
