package main

import (
	"crypto/sha256"
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

	// --- ESTADO ---
	isMining := false
	walletAddr := ""
	minedToday := uint64(0)
	balance := uint64(0)
	progressValue := 0.0

	// Carregar carteira salva
	data, err := os.ReadFile("wallet.txt")
	if err == nil {
		walletAddr = strings.TrimSpace(string(data))
	}

	// --- DESIGN (IGUAL À IMAGEM BITCOIN ADDER) ---

	// Logo Superior
	logoText := canvas.NewText("bitcoin", color.NRGBA{R: 247, G: 147, B: 26, A: 255})
	logoText.TextSize = 42
	logoText.TextStyle = fyne.TextStyle{Bold: true, Italic: true}

	subLogo := canvas.NewText("Ecopacto Money Adder v2.0", color.White)
	subLogo.TextSize = 14

	// Campo de Endereço
	addrLabel := widget.NewLabel("Ecopacto Address:")
	addrInput := widget.NewEntry()
	addrInput.SetPlaceHolder("Digite seu endereço ECO_...")
	addrInput.SetText(walletAddr)

	// Status Labels
	lblBalance := widget.NewLabel("Saldo Total: 0 ECO")
	lblMined := widget.NewLabel("Minerado Hoje: 0 ECO")
	lblStatus := widget.NewLabel("Status: EM ESPERA")

	progBar := widget.NewProgressBar()

	logs := widget.NewMultiLineEntry()
	logs.SetMinRowsVisible(5)
	logs.Disable()

	addLog := func(msg string) {
		t := time.Now().Format("15:04:05")
		logs.SetText(logs.Text + "[" + t + "] " + msg + "\n")
	}

	// Botões OK (Iniciar) e EXIT (Pausar)
	btnOK := widget.NewButton("OK", func() {
		walletAddr = strings.TrimSpace(addrInput.Text)
		if !strings.HasPrefix(walletAddr, "ECO_") {
			addLog("ERRO: Endereço inválido!")
			return
		}
		os.WriteFile("wallet.txt", []byte(walletAddr), 0644)
		isMining = true
		lblStatus.SetText("Status: MINERANDO ✅")
		addLog("Mineração iniciada para: " + walletAddr)
	})

	btnExit := widget.NewButton("EXIT", func() {
		isMining = false
		lblStatus.SetText("Status: PAUSADO ⏸")
		addLog("Mineração interrompida.")
	})

	// --- LOOPS DE ATUALIZAÇÃO ---
	go func() {
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
					addLog("Bloco minerado! +10 ECO registrados.")
				}
				progBar.SetValue(progressValue)
				sha256.Sum256([]byte(time.Now().String()))
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	// --- MONTAGEM DO LAYOUT ---
	header := container.NewVBox(logoText, subLogo, widget.NewSeparator())

	form := container.NewVBox(
		container.NewGridWithColumns(2, addrLabel, addrInput),
		container.NewGridWithColumns(2, btnOK, btnExit),
	)

	mainLayout := container.NewVBox(
		header,
		form,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, lblBalance, lblMined),
		lblStatus,
		progBar,
		widget.NewLabel("Status:"),
		logs,
		canvas.NewText("Powered by ECOPACTO SOFT", color.NRGBA{100, 100, 100, 255}),
	)

	myWindow.SetContent(container.NewPadded(mainLayout))
	myWindow.ShowAndRun()
}
