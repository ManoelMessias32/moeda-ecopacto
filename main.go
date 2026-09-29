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
	ADMIN_KEY  = "ECO_MASTER_888" // Chave X-Admin-Key para porta 30304
	FEE_RATE   = 0.005           // 0.5% conforme solicitado no TEST
)

// --- ESTRUTURAS CORE ---

type Transaction struct {
	ID        string `json:"id"`
	From      string `json:"from"` // "NETWORK" para recompensas
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

type Peer struct {
	ID        string    `json:"node_id"`
	Address   string    `json:"address"` // IP:Porta
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
	// Baseado no Halving de 5 anos (~262.800 blocos se 10min/bloco)
	if height < 262800 {
		return 10, 4 // Ano 0-5
	} else if height < 525600 {
		return 5, 5 // Ano 5-10
	}
	return 2, 6 // Ano 10-15 (uint64 arredonda 2.5 para 2)
}

func miningLoop() {
	for {
		if len(state.Mempool) > 0 {
			reward, difficulty := getMiningConfig()
			prevBlock := state.Blockchain[len(state.Blockchain)-1]

			// Coinbase Transaction (Recompensa)
			minerWallet := "ECO_MINER_REWARD_ADDRESS" // Em produção seria a do dono do nó
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
			saveState()
			fmt.Printf("⛏️ Bloco #%d minerado! Hash: %s\n", newBlock.Index, newBlock.Hash)
		}
		time.Sleep(10 * time.Second)
	}
}

// --- PORTA 8080: API PÚBLICA (Wallets / Explorer) ---

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct { Email string `json:"email"`; Name string `json:"name"` }
	json.NewDecoder(r.Body).Decode(&req)
	email := strings.ToLower(req.Email)

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
	if to := state.Wallets[req.To]; to != nil { to.Balance += req.Amount }
	state.Tokenomics.Treasury += fee

	saveState()
	json.NewEncoder(w).Encode(map[string]string{"status": "Transaction Pending", "tx_id": tx.ID})
}

// --- PORTA 30303: REDE P2P (Somente Nós) ---

func p2pHandler(w http.ResponseWriter, r *http.Request) {
	var msg P2PMessage
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		// Resposta básica de status se acessado via GET
		json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id":    state.Identity.NodeID,
			"height":     len(state.Blockchain),
			"public_key": state.Identity.PublicKey,
		})
		return
	}

	switch msg.Type {
	case "HELLO":
		challenge := make([]byte, 8)
		rand.Read(challenge)
		w.Header().Set("X-Challenge", hex.EncodeToString(challenge))
		json.NewEncoder(w).Encode(map[string]string{"node_id": state.Identity.NodeID, "public_key": state.Identity.PublicKey})
	case "GET_CHAIN":
		json.NewEncoder(w).Encode(state.Blockchain)
	}
}

// --- PORTA 30304: ADMIN (Painel de Controle) ---

func adminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Admin-Key") != ADMIN_KEY {
		http.Error(w, "Acesso negado", 401)
		return
	}
	reward, diff := getMiningConfig()

	path := r.URL.Path
	if strings.HasSuffix(path, "/stats") {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"node_id": state.Identity.NodeID,
			"blocks":  len(state.Blockchain),
			"mempool": len(state.Mempool),
			"mining":  map[string]int{"reward": int(reward), "difficulty": diff},
			"peers":   len(state.Peers),
			"treasury": state.Tokenomics.Treasury,
		})
	} else if strings.HasSuffix(path, "/nodes") {
		json.NewEncoder(w).Encode(state.Peers)
	}
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
		fmt.Println("📦 Blockchain carregada.")
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
			TotalSupply: 1200000000,
			MiningPool: 300000000,
			Symbol: "ECO",
		},
	}
	genesis := Block{Index: 0, Timestamp: time.Now().Unix(), PrevHash: "0", Data: "Ecopacto Network Genesis", Miner: "CORE"}
	genesis.Hash = calculateHash(genesis)
	state.Blockchain = append(state.Blockchain, genesis)
	saveState()
	fmt.Println("✨ Sistema Ecopacto inicializado.")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Admin-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	loadState()
	go miningLoop()

	// 1. MUX API PÚBLICA (8080)
	publicMux := http.NewServeMux()
	publicMux.HandleFunc("/login", loginHandler)
	publicMux.HandleFunc("/send", sendHandler)
	publicMux.HandleFunc("/tokenomics", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Tokenomics) })
	publicMux.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Blockchain) })
	publicMux.HandleFunc("/wallet", func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		json.NewEncoder(w).Encode(state.Wallets[addr])
	})
	publicMux.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(state.Peers) })

	// 2. MUX REDE P2P (30303)
	p2pMux := http.NewServeMux()
	p2pMux.HandleFunc("/", p2pHandler)

	// 3. MUX ADMIN (30304)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/admin/", adminHandler)

	// Iniciar servidores em Goroutines
	go func() {
		fmt.Println("🔗 REDE P2P ATIVA NA PORTA 30303")
		http.ListenAndServe(":30303", p2pMux)
	}()
	go func() {
		fmt.Println("🛡️  PORTA ADMIN PROTEGIDA ATIVA NA PORTA 30304")
		http.ListenAndServe(":30304", corsMiddleware(adminMux))
	}()

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	fmt.Printf("🚀 API ECOPACTO ONLINE EM 0.0.0.0:%s | ID: %s\n", port, state.Identity.NodeID)
	http.ListenAndServe("0.0.0.0:"+port, corsMiddleware(publicMux))
}
