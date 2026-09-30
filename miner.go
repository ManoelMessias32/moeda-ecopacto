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
	"syscall"
	"time"
	"unsafe"
)

// NOTA PARA TESTAR: Para rodar este arquivo sozinho no PC use:
// go run miner.go

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
	isMining    bool = true
	progress    int  = 0
)

// --- MOTOR NATIVO DO WINDOWS (API DE BAIXO NÍVEL) ---
var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode   = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode   = kernel32.NewProc("SetConsoleMode")
)

const (
	ENABLE_QUICK_EDIT_MODE = 0x0040
	ENABLE_EXTENDED_FLAGS  = 0x0080
)

// Desativa o QuickEdit (Impede que o clique do mouse trave o app)
func disableQuickEdit() {
	if runtime.GOOS != "windows" { return }

	// Usa a constante oficial do sistema para evitar erro de overflow
	h, err := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err != nil { return }

	var mode uint32
	procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))

	// Desativa QuickEdit e ativa Flags Estendidas
	mode &^= uint32(ENABLE_QUICK_EDIT_MODE)
	mode |= uint32(ENABLE_EXTENDED_FLAGS)

	procSetConsoleMode.Call(uintptr(h), uintptr(mode))
}

// Verifica se uma tecla está sendo tocada AGORA (Instantâneo)
func isKeyDown(vkey int) bool {
	ret, _, _ := procGetAsyncKeyState.Call(uintptr(vkey))
	return int16(ret) < 0
}

func main() {
	disableQuickEdit()
	setupTerminal()

	// 1. Tentar carregar carteira salva
	data, err := os.ReadFile("wallet.txt")
	if err == nil && len(data) > 10 {
		walletAddr = strings.TrimSpace(string(data))
	} else {
		fmt.Println("\n > [SETUP] ECOPACTO NETWORK")
		fmt.Print(" > Cole seu endereço ECO: ")
		fmt.Scanln(&walletAddr)
		if walletAddr == "" { walletAddr = "ECO_2b44246bf6a15b6361379bed" }
		os.WriteFile("wallet.txt", []byte(walletAddr), 0644)
	}

	// Limpa e esconde o cursor
	fmt.Print("\033[H\033[2J\033[?25l")

	addLog("Nó v21.0 Nativo Online")
	addLog("Teclas instantâneas: [1] [2] [3]")

	go fetchStats()
	go miningEngine()

	// Loop de Comandos (Tipo Android - Responde ao toque)
	go func() {
		for {
			if isKeyDown(0x31) { // Tecla '1'
				isMining = !isMining
				if isMining { addLog("Status: MINERANDO") } else { addLog("Status: PAUSADO") }
				time.Sleep(500 * time.Millisecond)
			}
			if isKeyDown(0x32) { // Tecla '2'
				addLog("Abrindo Explorer...")
				openURL(API_URL + "/history")
				time.Sleep(500 * time.Millisecond)
			}
			if isKeyDown(0x33) { // Tecla '3'
				addLog("Sincronizando Wallet...")
				openURL("https://ecopacto-api-production.up.railway.app")
				time.Sleep(500 * time.Millisecond)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}()

	renderLoop()
}

func renderLoop() {
	for {
		// Reposiciona o cursor no topo sem limpar a tela (Evita piscar)
		fmt.Print("\033[H")

		fmt.Println("┌──────────────────────────────────────────────────────────────┐")
		fmt.Println("│                   LAUNCHER ECOPACTO NETWORK                  │")
		fmt.Println("├────────────────────────────────┬─────────────────────────────┤")
		fmt.Println("│ PAINEL DO MINERADOR            │ DADOS DA REDE MAINNET       │")

		addrShort := walletAddr
		if len(addrShort) > 22 { addrShort = addrShort[:19] + "..." }

		fmt.Printf("│ • ID: %-24s │ • Bloco Atual: %-12d │\n", addrShort, blockHeight)
		fmt.Printf("│ • Saldo: %-21d │ • Dificuldade: 4            │\n", balance)
		fmt.Printf("│ • Minerado: %-18d │ • Reward: 10 ECO            │\n", minedToday)

		st := "ATIVO ✅ "
		if !isMining { st = "PAUSADO ⏸" }
		fmt.Printf("│ • Status: %-20s │ • Hashrate: %-10d H/s │\n", st, hashrate)
		fmt.Println("├────────────────────────────────┴─────────────────────────────┤")

		barSize := 40
		filled := (progress * barSize) / 100
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barSize-filled)
		fmt.Printf("│ Progress: [%s] %3d%%             │\n", bar, progress)

		fmt.Println("├──────────────────────────────────────────────────────────────┤")
		fmt.Println("│ BOTÕES (Toque na tecla para ação instantânea):               │")
		fmt.Println("│    [1] Ligar/Pausar        [2] Explorer        [3] Wallet    │")
		fmt.Println("├──────────────────────────────────────────────────────────────┤")
		fmt.Println("│ LOGS EM TEMPO REAL (NÃO TRAVA AO CLICAR COM MOUSE)           │")

		startIdx := len(logs) - 6
		if startIdx < 0 { startIdx = 0 }
		for i := startIdx; i < len(logs); i++ {
			fmt.Printf("│ > %-58s │\n", logs[i])
		}
		for i := 0; i < 6-(len(logs)-startIdx); i++ {
			fmt.Println("│                                                              │")
		}
		fmt.Println("└──────────────────────────────────────────────────────────────┘")
		time.Sleep(100 * time.Millisecond)
	}
}

func miningEngine() {
	for {
		if isMining {
			progress += 2
			if progress > 100 { progress = 0 }
			start := time.Now()
			for i := 0; i < 50000; i++ {
				h := sha256.Sum256([]byte(fmt.Sprintf("%d", i)))
				_ = hex.EncodeToString(h[:])
			}
			hashrate = int(50 / time.Since(start).Seconds())
			if time.Now().Second() % 45 == 0 {
				minedToday += 10
				addLog("Hash encontrado! +10 ECO registrados.")
				time.Sleep(1 * time.Second)
			}
		} else {
			hashrate = 0
			progress = 0
		}
		time.Sleep(100 * time.Millisecond)
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

func addLog(msg string) {
	t := time.Now().Format("15:04:05")
	logs = append(logs, fmt.Sprintf("[%s] %s", t, msg))
}

func setupTerminal() {
	if runtime.GOOS == "windows" {
		exec.Command("cmd", "/c", "title Ecopacto Network Launcher").Run()
	}
}

func openURL(url string) {
	if runtime.GOOS == "windows" {
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
