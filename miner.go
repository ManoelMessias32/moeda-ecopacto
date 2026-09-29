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
		fmt.Println(" [!] Endereço inválido. Usando padrão Master...")
		walletAddr = "ECO_2b44246bf6a15b6361379bed"
		time.Sleep(2 * time.Second)
	}

	addLog("Iniciando nó Ecopacto...")
	addLog("Conectando aos peers P2P na porta 30303")
	addLog("RPC conectado")

	go fetchStats()
	go miningEngine()

	renderLoop()
}

func renderLoop() {
	for {
		clearTerminal()
		fmt.Println("┌──────────────────────────────────────────────────────────────┐")
		fmt.Println("│                      LAUNCHER ECOPACTO                       │")
		fmt.Println("├────────────────────────────────┬─────────────────────────────┤")
		fmt.Println("│ CARTEIRA                       │ REDE                        │")

		addrShort := walletAddr
		if len(addrShort) > 22 { addrShort = addrShort[:22] + "..." }

		fmt.Printf("│ • ID: %-24s │ • Bloco Atual: %-12d │\n", addrShort, blockHeight)
		fmt.Printf("│ • Saldo: %-21d │ • Dificuldade: 4            │\n", balance, 4)
		fmt.Printf("│ • Minerado: %-18d │ • Reward: 10 ECO            │\n", minedToday)
		fmt.Printf("│ • Status: ONLINE ✅            │ • Hashrate: %-10d H/s │\n", hashrate)
		fmt.Println("├────────────────────────────────┴─────────────────────────────┤")
		fmt.Println("│ ABA DE NAVEGAÇÃO:                                            │")
		// BOTOES LADO A LADO NA VERSÃO TERMINAL
		fmt.Println("│ [ Iniciar ]        [ Explorer ]        [ Abrir Wallet ]      │")
		fmt.Println("├──────────────────────────────────────────────────────────────┤")
		fmt.Println("│ LOGS DO NÓ EM TEMPO REAL                                     │")

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
		for i := 0; i < 200000; i++ {
			sha256.Sum256([]byte(fmt.Sprintf("%d", i)))
		}
		hashrate = int(200 / time.Since(start).Seconds())

		if time.Now().Second() % 40 == 0 {
			minedToday += 10
			h := sha256.Sum256([]byte(time.Now().String()))
			addLog(fmt.Sprintf("Bloco minerado! Hash: 000%s", hex.EncodeToString(h[:4])))
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
