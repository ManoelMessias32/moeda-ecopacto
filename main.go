package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Block representa cada elo da corrente
type Block struct {
	Index     int    `json:"index"`
	Timestamp string `json:"timestamp"`
	Data      string `json:"data"`
	PrevHash  string `json:"prev_hash"`
	Hash      string `json:"hash"`
	Nonce     int    `json:"nonce"`
}

// Blockchain guarda a lista de blocos da moeda
type Blockchain struct {
	Chain      []Block `json:"chain"`
	Difficulty int     `json:"difficulty"`
}

// Tokenomics gerencia o suprimento fixo (12 Trilhões ECO)
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
}

// Estrutura para requisição de transferência
type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}

var ecoPacto Blockchain
var ecopactoTokenomics Tokenomics
var wallets = make(map[string]*Wallet)

// Recompensa aumentada para 100 ECO por bloco conforme solicitado
const MiningReward = 100

func calculateHash(b Block) string {
	record := fmt.Sprintf("%d%s%s%s%d", b.Index, b.Timestamp, b.Data, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// Handler de Mineração (Recompensa: 100 ECO)
func mineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	minerAddress := r.URL.Query().Get("address")

	if _, exists := wallets[minerAddress]; !exists {
		http.Error(w, "Carteira do minerador não encontrada", http.StatusNotFound)
		return
	}

	if ecopactoTokenomics.NodeMining < MiningReward {
		http.Error(w, "Reserva de mineração esgotada", http.StatusBadRequest)
		return
	}

	prevBlock := ecoPacto.Chain[len(ecoPacto.Chain)-1]
	newBlock := Block{
		Index:     prevBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data:      fmt.Sprintf("Recompensa de Mineração para %s", minerAddress),
		PrevHash:  prevBlock.Hash,
		Nonce:     0,
	}

	target := strings.Repeat("0", ecoPacto.Difficulty) + "7"
	fmt.Printf("⛏️  Node iniciando mineração para: %s\n", minerAddress)

	for {
		newBlock.Hash = calculateHash(newBlock)
		if strings.HasPrefix(newBlock.Hash, target) {
			break
		}
		newBlock.Nonce++
	}

	ecoPacto.Chain = append(ecoPacto.Chain, newBlock)
	wallets[minerAddress].Balance += MiningReward
	ecopactoTokenomics.NodeMining -= MiningReward

	fmt.Printf("✅ Bloco %d Minerado! Recompensa de %d ECO entregue.\n", newBlock.Index, MiningReward)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Mineração concluída!",
		"block":   newBlock,
		"reward":  MiningReward,
		"balance": wallets[minerAddress].Balance,
	})
}

// Handler de Transferência com Taxa de 20% em ECO que retorna para o Tesouro da moeda
func transferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requisição inválida", http.StatusBadRequest)
		return
	}

	fromWallet, existsFrom := wallets[req.From]
	toWallet, existsTo := wallets[req.To]

	if !existsFrom || !existsTo {
		http.Error(w, "Carteira de origem ou destino não encontrada", http.StatusNotFound)
		return
	}

	// Cálculo da Taxa de 20%
	fee := uint64(float64(req.Amount) * 0.20)
	totalRequired := req.Amount + fee

	if fromWallet.Balance < totalRequired {
		http.Error(w, "Saldo insuficiente para cobrir o valor + taxa de 20%", http.StatusBadRequest)
		return
	}

	// Executa a transação deduzindo o valor e a taxa da origem
	fromWallet.Balance -= totalRequired
	toWallet.Balance += req.Amount

	// A taxa retorna para o Tesouro do ecossistema, mantendo o suprimento total estável em 12T
	ecopactoTokenomics.Treasury += fee

	// Registra a transação em um novo bloco da Blockchain
	prevBlock := ecoPacto.Chain[len(ecoPacto.Chain)-1]
	txBlock := Block{
		Index:     prevBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data:      fmt.Sprintf("Tx: %s enviou %d ECO para %s (Taxa: %d ECO enviada ao Tesouro)", req.From, req.Amount, req.To, fee),
		PrevHash:  prevBlock.Hash,
		Hash:      "",
		Nonce:     0,
	}
	txBlock.Hash = calculateHash(txBlock)
	ecoPacto.Chain = append(ecoPacto.Chain, txBlock)

	fmt.Printf("💸 Transação efetuada: %s -> %s (%d ECO + %d Taxa)\n", req.From, req.To, req.Amount, fee)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "Sucesso",
		"amount_sent": req.Amount,
		"fee_charged": fee,
		"new_balance": fromWallet.Balance,
	})
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var request map[string]string
	json.NewDecoder(r.Body).Decode(&request)

	username := request["username"]
	if username == "" {
		http.Error(w, "Nome de usuário é obrigatório", http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(username + time.Now().String()))
	address := "ECO_" + hex.EncodeToString(hash[:])[:24]

	newWallet := &Wallet{Address: address, Balance: 0}
	wallets[address] = newWallet

	json.NewEncoder(w).Encode(newWallet)
}

func getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	wallet, exists := wallets[address]
	if !exists {
		http.Error(w, "Carteira não encontrada", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(wallet)
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ecoPacto)
}

func getTokenomics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ecopactoTokenomics)
}

func main() {
	ecopactoTokenomics = Tokenomics{
		TotalSupply:    12_000_000_000_000,
		GameRewards:    4_000_000_000_000,
		NodeMining:     3_000_000_000_000,
		DevEmergency:   1_000_000_000_000,
		GeneralReserve: 1_000_000_000_000,
		Treasury:       1_500_000_000_000,
		Liquidity:      1_000_000_000_000,
		Symbol:         "ECO",
	}

	ecoPacto = Blockchain{Difficulty: 4}
	genesis := Block{
		Index: 0, Timestamp: time.Now().String(), Data: "Ecopacto Genesis", PrevHash: "0",
	}
	genesis.Hash = calculateHash(genesis)
	ecoPacto.Chain = append(ecoPacto.Chain, genesis)

	wallets["ECO_DEV_MASTER"] = &Wallet{Address: "ECO_DEV_MASTER", Balance: ecopactoTokenomics.DevEmergency}

	http.HandleFunc("/blockchain", getBlockchain)
	http.HandleFunc("/tokenomics", getTokenomics)
	http.HandleFunc("/wallet/create", createWalletHandler)
	http.HandleFunc("/wallet/balance", getBalanceHandler)
	http.HandleFunc("/mine", mineHandler)
	http.HandleFunc("/transfer", transferHandler) // Novo endpoint de transferências

	fmt.Println("🚀 Rede ECOPACTO Atualizada com Sucesso!")
	fmt.Println("Regras de Consenso Aplicadas:")
	fmt.Println("  - Recompensa por Bloco: 100 ECO")
	fmt.Println("  - Taxa de Transferência: 20% (Destinada ao Tesouro)")
	fmt.Println("  - Limite de Suprimento: 12 Trilhões Imutáveis")

	http.ListenAndServe(":8080", nil)
}
