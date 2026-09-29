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

// --- ESTRUTURAS CORE ---

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
	Data         string        `json:"data,omitempty"`
}

type NodeIdentity struct {
	NodeID     string `json:"node_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey []byte `json:"-"`
}

type Peer struct {
	ID        string    `json:"node_id"`
	Address   string    `json:"address"`
	PublicKey string    `json:"public_key"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen"`
}

type Wallet struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

type Tokenomics struct {
	TotalSupply uint64 `json:"total_supply"`
	MiningPool  uint64 `json:"mining_pool"`
	Treasury    uint64 `json:"treasury"`
	Symbol      string `json:"symbol"`
}

type P2PMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type AppState struct {
	Identity   NodeIdentity       `json:"identity"`
	Blockchain []Block            `json:"blockchain"`
	Mempool    []Transaction      `json:"mempool"`
	Wallets    map[string]*Wallet `json:"wallets"`
	Tokenomics Tokenomics         `json:"tokenomics"`
	Emails     map[string]string  `json:"emails"`
	Peers      map[string]Peer    `json:"peers"`
}

var state AppState

// --- LÓGICA DE CONSENSO E MINERAÇÃO ---

func calculateHash(b Block) string {
	txData, _ := json.Marshal(b.Transactions)
	record := fmt.Sprintf("%d%d%s%s%d%s", b.Index, b.Timestamp, string(txData), b.PrevHash, b.Nonce, b.Miner)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func getMiningConfig() (uint64, int) {
	height := len(state.Blockchain)
	if height < 262800 { return 10, 4 }
	return 5, 5
}

func miningLoop() {
	for {
		if len(state.Mempool) > 0 {
			reward, difficulty := getMiningConfig()
			prevBlock := state.Blockchain[len(state.Blockchain)-1]

			minerWallet := "ECO_2b44246bf6a15b6361379bed"
			coinbase := Transaction{
				ID: "COINBASE_" + strconv.Itoa(prevBlock.Index+1),
				From: "NETWORK", To: minerWallet, Amount: reward, Timestamp: time.Now().Unix(),
			}

			newBlock := Block{
				Index:        prevBlock.Index + 1,
				Timestamp:    time.Now().Unix(),
				Transactions: append([]Transaction{coinbase}, state.Mempool...),
				PrevHash:     prevBlock.Hash,
				Miner:        state.Identity.NodeID,
			}

			target := strings.Repeat("0", difficulty)
			for {
				newBlock.Hash = calculateHash(newBlock)
				if strings.HasPrefix(newBlock.Hash, target) { break }
				newBlock.Nonce++
			}

			state.Blockchain = append(state.Blockchain, newBlock)
			state.Mempool = []Transaction{}

			if w, ok := state.Wallets[minerWallet]; ok {
				w.Balance += reward
			}

			saveState()
		}
		time.Sleep(10 * time.Second)
	}
}

// --- PORTA 8080: API PÚBLICA ---

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { Email string `json:"email"`; Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	email := strings.ToLower(req.Email)

	// Verifica se é o seu e-mail para entregar a conta Master
	if email == "manoeeletrico2hotmail.com@gmail.com" {
		targetAddr := "ECO_2b44246bf6a15b6361379bed"
		state.Emails[email] = targetAddr
		if state.Wallets[targetAddr] == nil {
			state.Wallets[targetAddr] = &Wallet{Address: targetAddr, Balance: 200000000000, Email: email, Name: req.Name}
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

func sendHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { From, To string; Amount uint64 }
	json.NewDecoder(r.Body).Decode(&req)
	from := state.Wallets[req.From]
	if from == nil || from.Balance < req.Amount {
		http.Error(w, "Saldo insuficiente", 400)
		return
	}

	fee := uint64(float64(req.Amount) * FEE_RATE)
	tx := Transaction{
		ID: hex.EncodeToString(sha256.New().Sum([]byte(fmt.Sprintf("%s%d", req.From, time.Now().UnixNano())))),
		From: req.From, To: req.To, Amount: req.Amount, Fee: fee, Timestamp: time.Now().Unix(),
	}

	state.Mempool = append(state.Mempool, tx)
	from.Balance -= (req.Amount + fee)
	if to, ok := state.Wallets[req.To]; ok { to.Balance += req.Amount }
	state.Tokenomics.Treasury += fee

	saveState()
	json.NewEncoder(w).Encode(map[string]string{"status": "Transaction Pending", "tx_id": tx.ID})
}

// --- SISTEMA ---

func saveState() {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(DB_FILE, data, 0644)
}

func loadState() {
	file, err := os.ReadFile(DB_FILE)
	if err == nil {
		json.Unmarshal(file, &state)
		// Garante os 200bi no carregamento
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
		Peers: make(map[string]Peer),
		Tokenomics: Tokenomics{
			TotalSupply: 1000000000000, // 1 Trilhão de ECO
			MiningPool:  500000000000,
			Symbol:      "ECO",
		},
	}

	// Carteira Master com 200 Bilhões
	targetAddr := "ECO_2b44246bf6a15b6361379bed"
	state.Wallets[targetAddr] = &Wallet{
		Address: targetAddr,
		Balance: 200000000000,
		Name:    "Manoel Master Wallet",
	}

	genesis := Block{Index: 0, Timestamp: time.Now().Unix(), PrevHash: "0", Data: "Genesis - ECOPACTO NETWORK MASTER", Miner: "CORE"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain = append(state.Blockchain, genesis)
	saveState()
}

func main() {
	loadState()
	go miningLoop()

	publicMux := http.NewServeMux()
	publicMux.HandleFunc("/login", loginHandler)
	publicMux.HandleFunc("/send", sendHandler)
	publicMux.HandleFunc("/tokenomics", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Tokenomics) })
	publicMux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Blockchain) })
	publicMux.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		json.NewEncoder(w).Encode(state.Wallets[addr])
	})

	fmt.Println("🚀 API ECOPACTO MASTER ONLINE NA PORTA 8080")
	http.ListenAndServe("0.0.0.0:8080", publicMux)
}
