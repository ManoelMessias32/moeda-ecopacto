package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

const API_URL = "https://ecopacto-api-production.up.railway.app"

type WalletData struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("ECOPACTO Mainnet Launcher v2.0")
	myWindow.Resize(fyne.NewSize(450, 600))

	// --- ESTADO DO APP ---
	isMining := false
	walletAddr := ""
	minedToday := uint64(0)
	balance := uint64(0)
	blockHeight := 0
	hashrate := 0
	progressValue := 0.0

	// Carregar carteira salva
	data, err := os.ReadFile("wallet.txt")
	if err == nil {
		walletAddr = strings.TrimSpace(string(data))
	}

	// --- COMPONENTES VISUAIS ---

	// 1. Título/Logo
	logo := canvas.NewText("ECOPACTO NETWORK", color.NRGBA{R: 34, G: 197, B: 94, A: 255})
	logo.TextSize = 28
	logo.TextStyle = fyne.TextStyle{Bold: true}
	logo.Alignment = fyne.TextAlignCenter

	// 2. Campo de Entrada (Igual à imagem enviada)
	addrInput := widget.NewEntry()
	addrInput.SetPlaceHolder("Endereço ECO_...")
	addrInput.SetText(walletAddr)

	// 3. Botões de Controle de Endereço
	btnSave := widget.NewButtonWithIcon("SALVAR / OK", theme.ConfirmIcon(), func() {
		walletAddr = strings.TrimSpace(addrInput.Text)
		if strings.HasPrefix(walletAddr, "ECO_") {
			os.WriteFile("wallet.txt", []byte(walletAddr), 0644)
			addLog("Carteira salva: " + walletAddr)
		} else {
			addLog("ERRO: Endereço inválido!")
		}
	})

	// 4. Painel de Status (Saldo, Bloco, etc)
	lblBalance := widget.NewLabel("Saldo Total: 0 ECO")
	lblMined := widget.NewLabel("Minerado Hoje: 0 ECO")
	lblBlock := widget.NewLabel("Bloco Atual: 0")
	lblHashrate := widget.NewLabel("Hashrate: 0 H/s")
	lblStatus := widget.NewLabel("Status: EM ESPERA")

	// 5. Barra de Progresso e Botões de Mineração
	progBar := widget.NewProgressBar()

	btnStart := widget.NewButtonWithIcon("INICIAR", theme.MediaPlayIcon(), nil)
	btnStart.Importance = widget.HighImportance

	btnPause := widget.NewButtonWithIcon("PAUSAR", theme.MediaPauseIcon(), func() {
		isMining = false
		btnStart.SetText("INICIAR")
		lblStatus.SetText("Status: PAUSADO ⏸")
		addLog("Mineração pausada pelo usuário.")
	})

	btnStart.OnTapped = func() {
		if walletAddr == "" {
			addLog("ERRO: Configure sua carteira primeiro!")
			return
		}
		isMining = true
		btnStart.SetText("RODANDO")
		lblStatus.SetText("Status: MINERANDO ✅")
		addLog("Iniciando processamento na Mainnet...")
	}

	// 6. Área de Logs
	logs := widget.NewMultiLineEntry()
	logs.SetMinRowsVisible(6)
	logs.Disable()

	addLog := func(msg string) {
		t := time.Now().Format("15:04:05")
		logs.SetText(logs.Text + "[" + t + "] " + msg + "\n")
		logs.CursorColumn = len(logs.Text)
	}

	// --- LOOPS DE ATUALIZAÇÃO ---
	go func() { // Loop de Dados da Rede
		for {
			if walletAddr != "" {
				resp, _ := http.Get(fmt.Sprintf("%s/wallet?address=%s", API_URL, walletAddr))
				if resp != nil {
					var d WalletData
					json.NewDecoder(resp.Body).Decode(&d)
					balance = d.Balance
					lblBalance.SetText(fmt.Sprintf("Saldo Total: %d ECO", balance))
					resp.Body.Close()
				}
				respH, _ := http.Get(API_URL + "/history")
				if respH != nil {
					var h []interface{}
					json.NewDecoder(respH.Body).Decode(&h)
					blockHeight = len(h)
					lblBlock.SetText(fmt.Sprintf("Bloco Atual: %d", blockHeight))
					respH.Body.Close()
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	go func() { // Motor Visual de Mineração
		for {
			if isMining {
				progressValue += 0.05
				if progressValue > 1.0 {
					progressValue = 0
					minedToday += 10
					lblMined.SetText(fmt.Sprintf("Minerado Hoje: %d ECO", minedToday))
					addLog("Bloco encontrado! Recompensa registrada.")
				}
				progBar.SetValue(progressValue)
				sha256.Sum256([]byte(time.Now().String())) // Simula esforço da CPU
			} else {
				progBar.SetValue(0)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// --- MONTAGEM DO LAYOUT ---
	inputBox := container.NewVBox(
		widget.NewLabelWithStyle("Endereço da Carteira Ecopacto:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		addrInput,
		btnSave,
	)

	statsGrid := container.NewGridWithColumns(2, lblBalance, lblMined, lblBlock, lblHashrate)

	mainLayout := container.NewVBox(
		logo,
		widget.NewSeparator(),
		inputBox,
		widget.NewSeparator(),
		statsGrid,
		lblStatus,
		widget.NewLabel("Progresso do Processamento:"),
		progBar,
		container.NewGridWithColumns(2, btnStart, btnPause),
		widget.NewSeparator(),
		widget.NewLabel("LOGS DO NÓ:"),
		logs,
	)

	myWindow.SetContent(container.NewPadded(mainLayout))
	myWindow.ShowAndRun()
}
func hexEnc(b []byte) string { return hex.EncodeToString(b) }
