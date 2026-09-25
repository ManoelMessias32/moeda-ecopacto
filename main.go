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
}

type TransferRequest struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Amount uint64 `json:"amount"`
}

var ecoPacto Blockchain
var ecopactoTokenomics Tokenomics
var wallets = make(map[string]*Wallet)

// Configurações de Mineração e Halving
const InitialMiningReward = 10
const HalvingIntervalBlocks = 2628000 // Aproximadamente 5 anos (considerando 1 bloco/minuto)

func calculateHash(b Block) string {
	record := fmt.Sprintf("%d%s%s%s%d", b.Index, b.Timestamp, b.Data, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

// Calcula a recompensa atual baseada no Halving (a cada 5 anos)
func getCurrentReward() uint64 {
	currentBlock := len(ecoPacto.Chain)
	halvings := currentBlock / HalvingIntervalBlocks
	reward := uint64(InitialMiningReward)

	for i := 0; i < halvings; i++ {
		reward = reward / 2
	}

	if reward < 1 { return 1 } // Recompensa mínima de 1 ECO
	return reward
}

func mineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	minerAddress := r.URL.Query().Get("address")

	if _, exists := wallets[minerAddress]; !exists {
		http.Error(w, "Carteira não encontrada", http.StatusNotFound)
		return
	}

	reward := getCurrentReward()

	if ecopactoTokenomics.NodeMining < reward {
		http.Error(w, "Reserva de mineração esgotada", http.StatusBadRequest)
		return
	}

	prevBlock := ecoPacto.Chain[len(ecoPacto.Chain)-1]
	newBlock := Block{
		Index:     prevBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data:      fmt.Sprintf("Reward: %d ECO to %s", reward, minerAddress),
		PrevHash:  prevBlock.Hash,
		Nonce:     0,
	}

	target := strings.Repeat("0", ecoPacto.Difficulty) + "7"
	for {
		newBlock.Hash = calculateHash(newBlock)
		if strings.HasPrefix(newBlock.Hash, target) {
			break
		}
		newBlock.Nonce++
	}

	ecoPacto.Chain = append(ecoPacto.Chain, newBlock)
	wallets[minerAddress].Balance += reward
	ecopactoTokenomics.NodeMining -= reward

	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Minerado!",
		"reward":  reward,
		"balance": wallets[minerAddress].Balance,
	})
}

func transferHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req TransferRequest
	json.NewDecoder(r.Body).Decode(&req)

	fromWallet := wallets[req.From]
	toWallet := wallets[req.To]

	if fromWallet == nil || toWallet == nil {
		http.Error(w, "Carteira inválida", http.StatusNotFound)
		return
	}

	// NOVA TAXA DE 1%
	fee := uint64(float64(req.Amount) * 0.01)
	if fee < 1 && req.Amount > 0 { fee = 1 } // Mínimo 1 ECO de taxa se o valor for > 0

	total := req.Amount + fee

	if fromWallet.Balance < total {
		http.Error(w, "Saldo insuficiente (Valor + 1% taxa)", http.StatusBadRequest)
		return
	}

	fromWallet.Balance -= total
	toWallet.Balance += req.Amount
	ecopactoTokenomics.Treasury += fee // Taxa vai para o tesouro de 150Bi

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "Sucesso",
		"fee": fee,
		"new_balance": fromWallet.Balance,
	})
}

func createWalletHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var request map[string]string
	json.NewDecoder(r.Body).Decode(&request)

	username := request["username"]
	hash := sha256.Sum256([]byte(username + time.Now().String()))
	address := "ECO_" + hex.EncodeToString(hash[:])[:24]

	newWallet := &Wallet{Address: address, Balance: 0}
	wallets[address] = newWallet
	json.NewEncoder(w).Encode(newWallet)
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
	// NOVO TOKENOMICS: 1.2 Trilhões
	ecopactoTokenomics = Tokenomics{
		TotalSupply:    1_200_000_000_000,
		GameRewards:    300_000_000_000,
		NodeMining:     300_000_000_000,
		DevEmergency:   200_000_000_000,
		Treasury:       150_000_000_000,
		GeneralReserve: 150_000_000_000,
		Liquidity:      100_000_000_000,
		Symbol:         "ECO",
	}

	ecoPacto = Blockchain{Difficulty: 4}
	genesis := Block{Index: 0, Timestamp: time.Now().String(), Data: "Ecopacto 1.2T Launch", PrevHash: "0"}
	genesis.Hash = calculateHash(genesis)
	ecoPacto.Chain = append(ecoPacto.Chain, genesis)

	// Carteira Fundadora (Seus 200 Bi)
	wallets["ECO_FOUNDER_RESERVE"] = &Wallet{Address: "ECO_FOUNDER_RESERVE", Balance: ecopactoTokenomics.DevEmergency}

	http.HandleFunc("/blockchain", getBlockchain)
	http.HandleFunc("/tokenomics", getTokenomics)
	http.HandleFunc("/wallet/create", createWalletHandler)
	http.HandleFunc("/mine", mineHandler)
	http.HandleFunc("/transfer", transferHandler)

	fmt.Println("🚀 Rede ECOPACTO v2 (1.2 Trilhões) Online!")
	fmt.Println("Taxa de 1% e Halving de 5 anos ativado.")
	http.ListenAndServe(":8080", nil)
}
