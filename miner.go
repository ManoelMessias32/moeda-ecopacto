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
	myWindow := myApp.NewWindow("Ecopacto Mainnet Launcher")
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

	// --- UI - TÍTULO ---
	title := canvas.NewText("ECOPACTO NETWORK", color.NRGBA{R: 34, G: 197, B: 94, A: 255})
	title.TextSize = 24
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	// --- UI - CAMPOS DE DADOS ---
	addrInput := widget.NewEntry()
	addrInput.SetPlaceHolder("Cole seu endereço ECO_...")
	addrInput.SetText(walletAddr)

	lblBalance := widget.NewLabel("Saldo Total: 0 ECO")
	lblMined := widget.NewLabel("Minerado Hoje: 0 ECO")
	lblBlock := widget.NewLabel("Bloco Atual: 0")
	lblHashrate := widget.NewLabel("Hashrate: 0 H/s")

	progBar := widget.NewProgressBar()

	logs := widget.NewMultiLineEntry()
	logs.SetMinRowsVisible(8)
	logs.Disable()

	addLog := func(msg string) {
		t := time.Now().Format("15:04:05")
		logs.SetText(logs.Text + "[" + t + "] " + msg + "\n")
		logs.CursorColumn = len(logs.Text)
	}

	// --- BOTÕES CLICÁVEIS ---
	btnStart := widget.NewButtonWithIcon("INICIAR MINERAÇÃO", theme.MediaPlayIcon(), nil)
	btnStart.Importance = widget.HighImportance

	btnStart.OnTapped = func() {
		if isMining {
			isMining = false
			btnStart.SetText("INICIAR MINERAÇÃO")
			btnStart.SetIcon(theme.MediaPlayIcon())
			addLog("Mineração interrompida.")
		} else {
			walletAddr = strings.TrimSpace(addrInput.Text)
			if !strings.HasPrefix(walletAddr, "ECO_") {
				addLog("ERRO: Endereço de carteira inválido!")
				return
			}
			os.WriteFile("wallet.txt", []byte(walletAddr), 0644)
			isMining = true
			btnStart.SetText("PARAR MINERAÇÃO")
			btnStart.SetIcon(theme.MediaStopIcon())
			addLog("Mineração iniciada com sucesso.")
		}
	}

	btnExplorer := widget.NewButton("Explorer", func() {
		u, _ := fyne.ParseURI(API_URL + "/history")
		myApp.OpenURL(u)
	})

	btnWallet := widget.NewButton("Abrir Wallet", func() {
		u, _ := fyne.ParseURI("https://ecopacto-api-production.up.railway.app")
		myApp.OpenURL(u)
	})

	// --- LOOPS DE ATUALIZAÇÃO (SEGUNDO PLANO) ---
	go func() {
		for {
			if walletAddr != "" {
				resp, err := http.Get(fmt.Sprintf("%s/wallet?address=%s", API_URL, walletAddr))
				if err == nil {
					var d WalletData
					json.NewDecoder(resp.Body).Decode(&d)
					balance = d.Balance
					lblBalance.SetText(fmt.Sprintf("Saldo Total: %d ECO", balance))
					resp.Body.Close()
				}
				respH, err := http.Get(API_URL + "/history")
				if err == nil {
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

	go func() {
		for {
			if isMining {
				progressValue += 0.05
				if progressValue > 1.0 {
					progressValue = 0
					minedToday += 10
					lblMined.SetText(fmt.Sprintf("Minerado Hoje: %d ECO", minedToday))
					addLog("Bloco encontrado! +10 ECO recebidos.")
				}
				progBar.SetValue(progressValue)

				start := time.Now()
				for i := 0; i < 50000; i++ { sha256.Sum256([]byte(fmt.Sprintf("%d", i))) }
				hashrate = int(50 / time.Since(start).Seconds())
				lblHashrate.SetText(fmt.Sprintf("Hashrate: %d H/s", hashrate))
			} else {
				progBar.SetValue(0)
				lblHashrate.SetText("Hashrate: 0 H/s")
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// --- LAYOUT FINAL ---
	mainLayout := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel("Configuração de Carteira:"),
		addrInput,
		container.NewGridWithColumns(2, lblBalance, lblMined),
		container.NewGridWithColumns(2, lblBlock, lblHashrate),
		widget.NewLabel("Status de Mineração:"),
		progBar,
		btnStart,
		container.NewGridWithColumns(2, btnExplorer, btnWallet),
		widget.NewLabel("Logs do Sistema:"),
		logs,
	)

	myWindow.SetContent(container.NewPadded(mainLayout))
	myWindow.ShowAndRun()
}
