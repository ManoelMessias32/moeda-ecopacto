package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
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
	Signature string `json:"signature,omitempty"`
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

type Tokenomics struct {
	TotalSupply uint64 `json:"total_supply"`
	Symbol      string `json:"symbol"`
}

type AppState struct {
	Identity   NodeIdentity       `json:"identity"`
	Blockchain []Block            `json:"blockchain"`
	Mempool    []Transaction      `json:"mempool"`
	Wallets    map[string]*Wallet `json:"wallets"`
	Tokenomics Tokenomics         `json:"tokenomics"`
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

	// CORREÇÃO: O e-mail do print é 'manoeletrico2hotmail.com@gmail.com'
	isMaster := email == "manoeletrico2hotmail.com@gmail.com" || email == "manoeeletrico2hotmail.com@gmail.com"

	if isMaster {
		targetAddr := "ECO_2b44246bf6a15b6361379bed"
		state.Emails[email] = targetAddr
		if state.Wallets[targetAddr] == nil {
			state.Wallets[targetAddr] = &Wallet{Address: targetAddr, Balance: 200000000000, Email: email, Name: "Manoel Oliveira"}
		}
		// Garante que o nome não vá vazio
		if state.Wallets[targetAddr].Name == "" || state.Wallets[targetAddr].Name == "undefined" {
			state.Wallets[targetAddr].Name = "Manoel Oliveira"
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

	displayName := req.Name
	if displayName == "" || displayName == "undefined" { displayName = "Usuário ECO" }

	newWallet := &Wallet{Address: address, Balance: 0, Email: email, Name: displayName}
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
		// Garante saldo master no reload
		targetAddr := "ECO_2b44246bf6a15b6361379bed"
		if w, ok := state.Wallets[targetAddr]; ok {
			if w.Balance < 200000000000 { w.Balance = 200000000000 }
		}
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
		Tokenomics: Tokenomics{TotalSupply: 1000000000000, Symbol: "ECO"},
	}

	targetAddr := "ECO_2b44246bf6a15b6361379bed"
	state.Wallets[targetAddr] = &Wallet{Address: targetAddr, Balance: 200000000000, Name: "Manoel Oliveira", Email: "manoeletrico2hotmail.com@gmail.com"}
	state.Emails["manoeletrico2hotmail.com@gmail.com"] = targetAddr

	genesis := Block{Index: 0, Timestamp: time.Now().Unix(), PrevHash: "0", Miner: "CORE"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain = append(state.Blockchain, genesis)
	saveState()
}

func main() {
	loadState()
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		json.NewEncoder(w).Encode(state.Wallets[addr])
	})
	fmt.Println("🚀 API ECOPACTO ONLINE NA PORTA 8080")
	http.ListenAndServe("0.0.0.0:8080", nil)
}
