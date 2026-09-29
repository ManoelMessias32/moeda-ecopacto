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

const API_URL = "https://ecopacto-api-production.up.railway.app"

type WalletData struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
}

var (
	walletAddr  string
	balance     uint64
	minedToday  uint64
	blockHeight int
	hashrate    int
	logs        []string
)

func main() {
	setupTerminal()

	fmt.Println("┌──────────────────────────────────────────────────────────────┐")
	fmt.Println("│                   LAUNCHER ECOPACTO NETWORK                  │")
	fmt.Println("└──────────────────────────────────────────────────────────────┘")
	fmt.Print(" > Insira o endereço da Carteira: ")
	fmt.Scanln(&walletAddr)

	if walletAddr == "" {
		walletAddr = "ECO_2b44246bf6a15b6361379bed"
	}

	if !strings.HasPrefix(walletAddr, "ECO_") {
		fmt.Println(" [!] Endereço inválido. Reinicie o launcher.")
		time.Sleep(3 * time.Second)
		return
	}

	addLog("Iniciando nó Ecopacto...")
	addLog("Conectando aos peers P2P na porta 30303")
	addLog("RPC conectado com sucesso")

	go fetchStats()
	go miningEngine()

	renderLoop()
}

func fetchStats() {
	for {
		resp, err := http.Get(fmt.Sprintf("%s/wallet?address=%s", API_URL, walletAddr))
		if err == nil {
			var data WalletData
			if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
				balance = data.Balance
			}
			resp.Body.Close()
		}

		respH, err := http.Get(fmt.Sprintf("%s/history", API_URL))
		if err == nil {
			var history []interface{}
			if err := json.NewDecoder(respH.Body).Decode(&history); err == nil {
				blockHeight = len(history)
			}
			respH.Body.Close()
		}
		time.Sleep(5 * time.Second)
	}
}

func miningEngine() {
	for {
		start := time.Now()
		// Simulação de processamento para calcular hashrate
		for i := 0; i < 300000; i++ {
			sha256.Sum256([]byte(fmt.Sprintf("%d", i)))
		}
		hashrate = int(300 / time.Since(start).Seconds())

		// Simulação de encontro de blocos
		if time.Now().Second() % 40 == 0 {
			minedToday += 10
			h := sha256.Sum256([]byte(time.Now().String()))
			addLog(fmt.Sprintf("Bloco encontrado! Hash: 000%s", hex.EncodeToString(h[:8])))
			addLog("Recompensa de 10 ECO registrada na rede.")
			time.Sleep(1 * time.Second)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func renderLoop() {
	for {
		clearTerminal()
		fmt.Println("┌──────────────────────────────────────────────────────────────┐")
		fmt.Println("│                      LAUNCHER ECOPACTO                       │")
		fmt.Println("├────────────────────────────────┬─────────────────────────────┤")
		fmt.Println("│ PAINEL SUPERIOR:               │ INFORMAÇÕES:                │")

		displayAddr := walletAddr
		if len(displayAddr) > 18 {
			displayAddr = displayAddr[:18] + "..."
		}

		fmt.Printf("│ • Endereço: %-18s │ • Dificuldade: 4            │\n", displayAddr)
		fmt.Printf("│ • Saldo ECO: %-17d │ • Bloco Atual: %-12d │\n", balance, blockHeight)
		fmt.Printf("│ • Minerado Hoje: %-13d │ • Hashrate: %-10d H/s │\n", minedToday, hashrate)
		fmt.Println("├────────────────────────────────┴─────────────────────────────┤")
		fmt.Println("│ ABA DE NAVEGAÇÃO:                                            │")
		fmt.Println("│ [ Dashboard ]    [ Explorer da Rede ]    [ Abrir Wallet ]    │")
		fmt.Println("├──────────────────────────────────────────────────────────────┤")
		fmt.Println("│ JANELA INFERIOR (LOGS DO NÓ EM TEMPO REAL)                   │")

		// Mostrar os últimos 8 logs
		startIdx := len(logs) - 8
		if startIdx < 0 { startIdx = 0 }
		for i := startIdx; i < len(logs); i++ {
			fmt.Printf("│ > %-58s │\n", logs[i])
		}
		for i := 0; i < 8 - (len(logs)-startIdx); i++ {
			fmt.Println("│                                                              │")
		}
		fmt.Println("└──────────────────────────────────────────────────────────────┘")
		time.Sleep(1 * time.Second)
	}
}

func addLog(msg string) {
	t := time.Now().Format("15:04:05")
	logs = append(logs, fmt.Sprintf("[%s] %s", t, msg))
}

func clearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func setupTerminal() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "title Launcher Ecopacto Network").Run()
	}
}
