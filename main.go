package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// --- CONSTANTES ---
const (
	DB_FILE    = "blockchain.db"
	ADMIN_KEY  = "ECO_MASTER_888"
	FEE_RATE   = 0.005
)

type Transaction struct {
	ID        string `json:"id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	Fee       uint64 `json:"fee"`
	Timestamp int64  `json:"timestamp"`
}

type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PrevHash     string        `json:"prev_hash"`
	Hash         string        `json:"hash"`
	Nonce        int           `json:"nonce"`
	Miner        string        `json:"miner"`
}

type NodeIdentity struct {
	NodeID     string `json:"node_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey []byte `json:"-"`
}

type Wallet struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

type AppState struct {
	Identity   NodeIdentity       `json:"identity"`
	Blockchain []Block            `json:"blockchain"`
	Mempool    []Transaction      `json:"mempool"`
	Wallets    map[string]*Wallet `json:"wallets"`
	Emails     map[string]string  `json:"emails"`
}

var state AppState

func calculateHash(b Block) string {
	txData, _ := json.Marshal(b.Transactions)
	record := fmt.Sprintf("%d%d%s%s%d%s", b.Index, b.Timestamp, string(txData), b.PrevHash, b.Nonce, b.Miner)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
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
	if err == nil {
		json.Unmarshal(file, &state)
	} else {
		initSystem()
	}
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
	}
	genesis := Block{Index: 0, Timestamp: time.Now().Unix(), PrevHash: "0"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain = append(state.Blockchain, genesis)
	saveState()
}

func main() {
	loadState()

	mux := http.NewServeMux()
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		json.NewEncoder(w).Encode(state.Wallets[addr])
	})
	mux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(state.Blockchain)
	})

	// ROTA DE DOWNLOAD DO LAUNCHER
	mux.HandleFunc("/download-launcher", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename=ecopacto-launcher.exe")
		http.ServeFile(w, r, "./ecopacto-launcher.exe")
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
	fmt.Printf("🚀 API ECOPACTO ONLINE NA PORTA %s\n", port)
	http.ListenAndServe("0.0.0.0:"+port, handler)
}
