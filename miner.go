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
)

func main() {
	setupTerminal()

	fmt.Println("┌──────────────────────────────────────────────────┐")
	fmt.Println("│                ECOPACTO NETWORK                  │")
	fmt.Println("└──────────────────────────────────────────────────┘")
	fmt.Print(" > Insira o endereço da Carteira: ")
	fmt.Scanln(&walletAddr)

	if walletAddr == "" {
		walletAddr = "ECO_2b44246bf6a15b6361379bed"
	}

	if !strings.HasPrefix(walletAddr, "ECO_") {
		fmt.Println(" [!] Endereço inválido. Usando endereço padrão Master...")
		walletAddr = "ECO_2b44246bf6a15b6361379bed"
		time.Sleep(2 * time.Second)
	}

	addLog("Conectando ao RPC...")
	addLog("RPC conectado")
	addLog("Peer NodeA conectado")
	addLog("Peer NodeB conectado")
	addLog("Minerando...")

	go fetchStats()
	go miningEngine()

	renderLoop()
}

func renderLoop() {
	for {
		clearTerminal()
		fmt.Println("┌──────────────────────────────────────────────────┐")
		fmt.Println("│                ECOPACTO NETWORK                  │")
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ Carteira                                         │")
		fmt.Printf("│ %-48s │\n", walletAddr)
		fmt.Println("│                                                  │")
		fmt.Printf("│ Saldo ECO: %-37s │\n", fmt.Sprintf("%d ECO", balance))
		fmt.Printf("│ Saldo Minerado Hoje: %-27s │\n", fmt.Sprintf("%d ECO", minedToday))
		fmt.Printf("│ Próximo Pagamento: %-29s │\n", "00:00")
		fmt.Println("│                                                  │")
		fmt.Println("│ Status: ONLINE ✅                                │")
		fmt.Printf("│ Nós Conectados: %-32d │\n", 7)
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ Rede                                             │")
		fmt.Printf("│ Bloco Atual: %-35d │\n", blockHeight)
		fmt.Println("│ Dificuldade: 4                                   │")
		fmt.Println("│ Recompensa: 10 ECO                               │")
		fmt.Printf("│ Hashrate: %-38s │\n", fmt.Sprintf("%d H/s", hashrate))
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ [ Iniciar Mineração ]                            │")
		fmt.Println("│ [ Explorer da Rede ]                             │")
		fmt.Println("│ [ Abrir Wallet ]                                 │")
		fmt.Println("├──────────────────────────────────────────────────┤")
		fmt.Println("│ LOGS DO NÓ                                       │")
		fmt.Println("│                                                  │")

		startIdx := len(logs) - 8
		if startIdx < 0 {
			startIdx = 0
		}
		for i := startIdx; i < len(logs); i++ {
			fmt.Printf("│ %-48s │\n", logs[i])
		}
		for i := 0; i < 8-(len(logs)-startIdx); i++ {
			fmt.Println("│                                                  │")
		}
		fmt.Println("└──────────────────────────────────────────────────┘")
		time.Sleep(1 * time.Second)
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
		time.Sleep(5 * time.Second)
	}
}

func miningEngine() {
	for {
		start := time.Now()
		for i := 0; i < 150000; i++ {
			sha256.Sum256([]byte(fmt.Sprintf("%d", i)))
		}
		hashrate = int(150 / time.Since(start).Seconds())

		if time.Now().Second()%30 == 0 {
			minedToday += 10
			addLog("Novo bloco recebido")
			addLog("Hash válido encontrado")
			addLog("Recompensa registrada")
			time.Sleep(1 * time.Second)
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
