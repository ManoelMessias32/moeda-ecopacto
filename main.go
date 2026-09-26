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
)

type Block struct {
	Index     int    `json:"index"`
	Timestamp string `json:"timestamp"`
	Data      string `json:"data"`
	PrevHash  string `json:"prev_hash"`
	Hash      string `json:"hash"`
	Nonce     int    `json:"nonce"`
}

type Blockchain struct {
	Chain      []Block `json:"chain"`
	Difficulty int     `json:"difficulty"`
}

type Tokenomics struct {
	TotalSupply    uint64 `json:"total_supply"`
	GameRewards    uint64 `json:"game_rewards"`
	NodeMining     uint64 `json:"node_mining"`
	DevEmergency   uint64 `json:"dev_emergency"`
	GeneralReserve uint64 `json:"general_reserve"`
	Treasury       uint64 `json:"treasury"`
	Liquidity      uint64 `json:"liquidity"`
	Symbol         string `json:"symbol"`
}

type Wallet struct {
	Address string `json:"address"`
	Balance uint64 `json:"balance"`
	Email   string `json:"email"`
}

type AppState struct {
	Blockchain Blockchain        `json:"blockchain"`
	Wallets    map[string]*Wallet `json:"wallets"`
	Tokenomics Tokenomics        `json:"tokenomics"`
	Emails     map[string]string `json:"emails"`
}

var state AppState
const DB_FILE = "blockchain.db"

func saveState() {
	data, _ := json.MarshalIndent(state, "", "  ")
	os.WriteFile(DB_FILE, data, 0644)
}

func loadState() {
	file, err := os.ReadFile(DB_FILE)
	if err == nil {
		json.Unmarshal(file, &state)
		fmt.Println("📦 Banco de dados carregado.")
	} else {
		initNewBlockchain()
	}
}

func initNewBlockchain() {
	state = AppState{
		Wallets: make(map[string]*Wallet),
		Emails:  make(map[string]string),
		Tokenomics: Tokenomics{
			TotalSupply: 1_200_000_000_000, GameRewards: 300_000_000_000, NodeMining: 300_000_000_000,
			DevEmergency: 200_000_000_000, Treasury: 150_000_000_000, GeneralReserve: 150_000_000_000,
			Liquidity: 100_000_000_000, Symbol: "ECO",
		},
		Blockchain: Blockchain{Difficulty: 4},
	}
	genesis := Block{Index: 0, Timestamp: time.Now().String(), Data: "Ecopacto Genesis", PrevHash: "0"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain.Chain = append(state.Blockchain.Chain, genesis)
	state.Wallets["ECO_FOUNDER_RESERVE"] = &Wallet{Address: "ECO_FOUNDER_RESERVE", Balance: state.Tokenomics.DevEmergency}
	saveState()
}

func calculateHash(b Block) string {
	record := fmt.Sprintf("%d%s%s%s%d", b.Index, b.Timestamp, b.Data, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// Middleware de segurança para permitir conexão da Vercel
func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ Email string `json:"email"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	email := strings.ToLower(req.Email)
	fmt.Printf("👤 Tentativa de login: %s\n", email)

	if addr, exists := state.Emails[email]; exists {
		json.NewEncoder(w).Encode(state.Wallets[addr])
		return
	}
	hash := sha256.Sum256([]byte(email + time.Now().String()))
	address := "ECO_" + hex.EncodeToString(hash[:])[:24]
	newWallet := &Wallet{Address: address, Balance: 0, Email: email}
	state.Wallets[address] = newWallet
	state.Emails[email] = address
	saveState()
	json.NewEncoder(w).Encode(newWallet)
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	var req struct{ From string `json:"from"`; To string `json:"to"`; Amount uint64 `json:"amount"` }
	json.NewDecoder(r.Body).Decode(&req)
	from := state.Wallets[req.From]; to := state.Wallets[req.To]
	if from == nil || to == nil { http.Error(w, "Carteira não encontrada", 404); return }

	fee := uint64(float64(req.Amount) * 0.01)
	if fee < 1 && req.Amount > 0 { fee = 1 }
	if from.Balance < req.Amount + fee { http.Error(w, "Saldo insuficiente", 400); return }

	from.Balance -= (req.Amount + fee)
	to.Balance += req.Amount
	state.Tokenomics.Treasury += fee
	saveState()
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "new_balance": from.Balance})
}

func main() {
	loadState()
	mux := http.NewServeMux()
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/transfer", transferHandler)
	mux.HandleFunc("/tokenomics", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(state.Tokenomics)
	})

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Printf("🚀 ECOPACTO ONLINE EM 0.0.0.0:%s\n", port)
	http.ListenAndServe("0.0.0.0:"+port, commonMiddleware(mux))
}
