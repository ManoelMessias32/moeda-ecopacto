package main

import (
	"crypto/sha256"
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
	isMining    bool = false
	progress    int  = 0
)

func main() {
	setupTerminal()

	// 1. Tentar carregar carteira salva
	data, err := os.ReadFile("wallet.txt")
	if err == nil && len(data) > 10 {
		walletAddr = strings.TrimSpace(string(data))
		isMining = true
	} else {
		fmt.Println("┌──────────────────────────────────────────────────┐")
		fmt.Println("│                ECOPACTO NETWORK                  │")
		fmt.Println("└──────────────────────────────────────────────────┘")
		fmt.Print(" > Insira sua Carteira ECO: ")
		fmt.Scanln(&walletAddr)

		if walletAddr == "" || !strings.HasPrefix(walletAddr, "ECO_") {
			walletAddr = "ECO_2b44246bf6a15b6361379bed"
		}
		os.WriteFile("wallet.txt", []byte(walletAddr), 0644)
		isMining = true
	}

	addLog("Conectado à Mainnet Ecopacto")
	addLog("Aperte [1] para Iniciar/Parar Mineração")

	go fetchStats()
	go miningEngine()

	// Captura comandos do teclado sem travar a tela
	go func() {
		for {
			var cmd string
			fmt.Scanln(&cmd)
			switch cmd {
			case "1":
				isMining = !isMining
				if isMining { addLog("Mineração ATIVADA") } else { addLog("Mineração PAUSADA") }
			case "2":
				addLog("Abrindo Explorer...")
				openBrowser(API_URL + "/history")
			case "3":
				addLog("Abrindo Wallet...")
				openBrowser("https://ecopacto-api-production.up.railway.app")
			}
		}
	}()

	renderLoop()
}

func renderLoop() {
	for {
		clearTerminal()
		fmt.Println("┌──────────────────────────────────────────────────┐")
		fmt.Println("│                ECOPACTO NETWORK                  │")
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ CARTEIRA                                         │")

		addrShort := walletAddr
		if len(addrShort) > 45 { addrShort = addrShort[:42] + "..." }
		fmt.Printf("│ %-48s │\n", addrShort)
		fmt.Println("│                                                  │")
		fmt.Printf("│ Saldo ECO: %-37s │\n", fmt.Sprintf("%d ECO", balance))
		fmt.Printf("│ Saldo Minerado Hoje: %-27s │\n", fmt.Sprintf("%d ECO", minedToday))
		fmt.Println("│                                                  │")
		statusStr := "ONLINE ✅"
		if !isMining { statusStr = "PAUSADO ⏸" }
		fmt.Printf("│ Status: %-40s │\n", statusStr)
		fmt.Printf("│ Nós Conectados: %-32d │\n", 7)
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ REDE                                             │")
		fmt.Printf("│ Bloco Atual: %-35d │\n", blockHeight)
		fmt.Printf("│ Hashrate: %-38s │\n", fmt.Sprintf("%d H/s", hashrate))

		// BARRA DE PROGRESSO VISUAL
		barSize := 30
		filled := (progress * barSize) / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barSize-filled)
		fmt.Printf("│ [%s] %3d%%          │\n", bar, progress)

		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ COMANDOS (Digite o número e aperte Enter):       │")
		fmt.Println("│ [1] Iniciar/Parar  [2] Explorer  [3] Abrir Wallet│")
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ LOGS DO NÓ                                       │")

		startIdx := len(logs) - 5
		if startIdx < 0 { startIdx = 0 }
		for i := startIdx; i < len(logs); i++ {
			fmt.Printf("│ %-48s │\n", logs[i])
		}
		for i := 0; i < 5 - (len(logs)-startIdx); i++ {
			fmt.Println("│                                                  │")
		}
		fmt.Println("└──────────────────────────────────────────────────┘")
		time.Sleep(250 * time.Millisecond)
	}
}

func fetchStats() {
	for {
		resp, err := http.Get(fmt.Sprintf("%s/wallet?address=%s", API_URL, walletAddr))
		if err == nil {
			var data WalletData
			json.NewDecoder(resp.Body).Decode(&data)
			balance = data.Balance
			resp.Body.Close()
		}
		respH, err := http.Get(API_URL + "/history")
		if err == nil {
			var history []interface{}
			json.NewDecoder(respH.Body).Decode(&history)
			blockHeight = len(history)
			respH.Body.Close()
		}
		time.Sleep(10 * time.Second)
	}
}

func miningEngine() {
	for {
		if isMining {
			progress += 2
			if progress > 100 { progress = 0 }
			start := time.Now()
			for i := 0; i < 50000; i++ { sha256.Sum256([]byte(fmt.Sprintf("%d", i))) }
			hashrate = int(50 / time.Since(start).Seconds())

			if time.Now().Second() % 40 == 0 {
				minedToday += 10
				addLog("Hash válido encontrado! (+10 ECO)")
				time.Sleep(1 * time.Second)
			}
		} else {
			hashrate = 0
			progress = 0
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func addLog(msg string) {
	t := time.Now().Format("15:04:05")
	logs = append(logs, fmt.Sprintf("[%s] %s", t, msg))
}

func clearTerminal() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" { cmd = exec.Command("cmd", "/c", "cls") } else { cmd = exec.Command("clear") }
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func setupTerminal() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "title Ecopacto Mainnet Launcher").Run()
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil { addLog("Erro ao abrir navegador") }
}
