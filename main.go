package main

import (
	"archive/zip"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DB_FILE    = "blockchain.db"
	ADMIN_KEY  = "ECO_MASTER_888"
)

type AppState struct {
	Identity   NodeIdentity       `json:"identity"`
	Blockchain []Block            `json:"blockchain"`
	Mempool    []Transaction      `json:"mempool"`
	Wallets    map[string]*Wallet `json:"wallets"`
	Emails     map[string]string  `json:"emails"`
	Tokenomics Tokenomics         `json:"tokenomics"`
}

type Tokenomics struct {
	TotalSupply uint64 `json:"total_supply"`
	MiningPool  uint64 `json:"mining_pool"` // Reserva para pagamentos
	Burned      uint64 `json:"burned"`      // Moedas destruídas para valorização
	Symbol      string `json:"symbol"`
}

type Wallet struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Pending uint64 `json:"pending"` // Minerado aguardando 00:00
	Email   string `json:"email"`
	Name    string `json:"name"`
}

type Transaction struct {
	ID string `json:"id"`; From string `json:"from"`; To string `json:"to"`; Amount uint64 `json:"amount"`; Fee uint64 `json:"fee"`; Timestamp int64 `json:"timestamp"`
}

type Block struct {
	Index int `json:"index"`; Timestamp int64 `json:"timestamp"`; Transactions []Transaction `json:"transactions"`; PrevHash string `json:"prev_hash"`; Hash string `json:"hash"`; Nonce int `json:"nonce"`; Miner string `json:"miner"`
}

type NodeIdentity struct {
	NodeID string `json:"node_id"`; PublicKey string `json:"public_key"`; PrivateKey []byte `json:"-"`
}

var state AppState

func calculateHash(b Block) string {
	txData, _ := json.Marshal(b.Transactions)
	record := fmt.Sprintf("%d%d%s%s%d%s", b.Index, b.Timestamp, string(txData), b.PrevHash, b.Nonce, b.Miner)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// SISTEMA DE PAGAMENTO À MEIA-NOITE
func midnightPayoutTask() {
	for {
		now := time.Now()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		time.Sleep(time.Until(nextMidnight))

		fmt.Println("💰 [PAYOUT] Iniciando distribuição diária...")
		for _, w := range state.Wallets {
			if w.Pending > 0 {
				if state.Tokenomics.MiningPool >= w.Pending {
					w.Balance += w.Pending
					state.Tokenomics.MiningPool -= w.Pending
					w.Pending = 0
				}
			}
		}
		saveState()
	}
}

// ENVIO COM TAXA DE 5% (2% QUEIMA / 3% POOL)
func sendHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { From, To string; Amount uint64 }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { return }

	from := state.Wallets[req.From]
	if from == nil || from.Balance < req.Amount {
		http.Error(w, "Saldo insuficiente", 400)
		return
	}

	fee := uint64(float64(req.Amount) * 0.05)
	burn := uint64(float64(req.Amount) * 0.02)
	pool := fee - burn

	from.Balance -= (req.Amount + fee)
	if to, ok := state.Wallets[req.To]; ok { to.Balance += req.Amount }

	state.Tokenomics.Burned += burn
	state.Tokenomics.TotalSupply -= burn
	state.Tokenomics.MiningPool += pool

	saveState()
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "Transação Concluída", "queimado": burn, "reabastecido_pool": pool})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { Email string `json:"email"`; Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	email := strings.ToLower(req.Email)
	isMaster := email == "manoeletrico2hotmail.com@gmail.com" || email == "manoeeletrico2hotmail.com@gmail.com"

	if isMaster {
		targetAddr := "ECO_2b44246bf6a15b6361379bed"
		state.Emails[email] = targetAddr
		if state.Wallets[targetAddr] == nil {
			state.Wallets[targetAddr] = &Wallet{Address: targetAddr, Balance: 200000000000, Email: email, Name: "Manoel Oliveira"}
		}
		json.NewEncoder(w).Encode(state.Wallets[targetAddr])
		return
	}

	if addr, exists := state.Emails[email]; exists {
		json.NewEncoder(w).Encode(state.Wallets[addr])
		return
	}

	hash := sha256.Sum256([]byte(email + time.Now().String()))
	address := "ECO_" + hex.EncodeToString(hash[:])[:24]
	newWallet := &Wallet{Address: address, Balance: 0, Email: email, Name: req.Name}
	state.Wallets[address] = newWallet
	state.Emails[email] = address
	saveState()
	json.NewEncoder(w).Encode(newWallet)
}

func saveState() {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(DB_FILE, data, 0644)
}

func loadState() {
	file, err := os.ReadFile(DB_FILE)
	if err == nil { json.Unmarshal(file, &state) } else { initSystem() }
}

func initSystem() {
	pub, priv, _ := ed25519.GenerateKey(nil)
	state = AppState{
		Identity: NodeIdentity{
			NodeID: "NODE_" + hex.EncodeToString(sha256.New().Sum([]byte(time.Now().String())))[:12],
			PublicKey: hex.EncodeToString(pub),
			PrivateKey: priv,
		},
		Wallets: make(map[string]*Wallet),
		Emails: make(map[string]string),
		Tokenomics: Tokenomics{
			TotalSupply: 1000000000000,
			MiningPool:  300000000000,
			Burned:      0,
			Symbol:      "ECO",
		},
	}
	genesis := Block{Index: 0, Timestamp: time.Now().Unix(), PrevHash: "0"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain = append(state.Blockchain, genesis)
	saveState()
}

func main() {
	loadState()
	go midnightPayoutTask()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/send", sendHandler)
	mux.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		json.NewEncoder(w).Encode(state.Wallets[addr])
	})
	mux.HandleFunc("/tokenomics", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Tokenomics) })
	mux.HandleFunc("/mine/submit", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		if wallet, ok := state.Wallets[addr]; ok {
			wallet.Pending += 10
			saveState()
			w.WriteHeader(200)
		}
	})
	mux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Blockchain) })

	mux.HandleFunc("/download-launcher", func(w http.ResponseWriter, r *http.Request) {
		exePath := "./ecopacto-launcher.exe"
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", "attachment; filename=ecopacto-network.zip")
		zw := zip.NewWriter(w)
		f, _ := zw.Create("bin/ecopacto-launcher.exe")
		file, err := os.Open(exePath)
		if err == nil {
			io.Copy(f, file)
			file.Close()
		}
		zw.Close()
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" { return }
		mux.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Printf("🚀 ECOPACTO ONLINE NA PORTA %s\n", port)
	http.ListenAndServe("0.0.0.0:"+port, handler)
}
