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

// Blockchain é a estrutura que guarda a moeda "real"
type Blockchain struct {
	Chain      []Block `json:"chain"`
	Difficulty int     `json:"difficulty"`
}

// Tokenomics (12 Trilhões ECO)
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

var ecoPacto Blockchain
var ecopactoTokenomics Tokenomics

func calculateHash(b Block) string {
	record := fmt.Sprintf("%d%s%s%s%d", b.Index, b.Timestamp, b.Data, b.PrevHash, b.Nonce)
	h := sha256.New()
	h.Write([]byte(record))
	return hex.EncodeToString(h.Sum(nil))
}

func (bc *Blockchain) createNextBlock(data string) Block {
	prevBlock := bc.Chain[len(bc.Chain)-1]
	newBlock := Block{
		Index:     prevBlock.Index + 1,
		Timestamp: time.Now().String(),
		Data:      data,
		PrevHash:  prevBlock.Hash,
		Nonce:     0,
	}

	target := strings.Repeat("0", bc.Difficulty) + "7"
	fmt.Printf("⛏️  Minerando Bloco %d...\n", newBlock.Index)

	for {
		newBlock.Hash = calculateHash(newBlock)
		if strings.HasPrefix(newBlock.Hash, target) {
			fmt.Printf("✅ Bloco %d Minerado!\n", newBlock.Index)
			return newBlock
		}
		newBlock.Nonce++
	}
}

// Handler para ver a Blockchain completa
func getBlockchain(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ecoPacto)
}

// Handler para ver a distribuição dos tokens
func getTokenomics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ecopactoTokenomics)
}

func main() {
	// 1. Configurar Tokenomics
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

	// 2. Inicializar Blockchain com Bloco Gênesis
	ecoPacto = Blockchain{Difficulty: 4}
	genesis := Block{
		Index:     0,
		Timestamp: time.Now().String(),
		Data:      "Ecopacto Genesis Block",
		PrevHash:  "0",
	}
	genesis.Hash = calculateHash(genesis)
	ecoPacto.Chain = append(ecoPacto.Chain, genesis)

	// 3. Configurar Rotas do Servidor
	http.HandleFunc("/blockchain", getBlockchain)
	http.HandleFunc("/tokenomics", getTokenomics)

	// Iniciar servidor na porta 8080
	fmt.Println("🚀 Servidor Ecopacto Online em http://localhost:8080")
	fmt.Println("Endpoints:")
	fmt.Println("  - http://localhost:8080/blockchain")
	fmt.Println("  - http://localhost:8080/tokenomics")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Erro ao iniciar servidor: %s\n", err)
	}
}
