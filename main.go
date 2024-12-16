package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"time"
)

type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	ISBN   string `json:"isbn"`
}

type BookCheckout struct {
	BookId       string `json:"book_id"`
	User         string `json:"user"`
	CheckoutDate string `json:"checkout_date"`
	IsGenesis    bool   `json:"is_genesis"`
}

type Block struct {
	Position     int          `json:"position"`
	Data         BookCheckout `json:"data"`
	TimeStamp    time.Time    `json:"timestamp"`
	Hash         string       `json:"hash"`
	PreviousHash string       `json:"prev_hash"`
}

func (b *Block) generateHash() {
	bytes, _ := json.Marshal(b)

	data := string(b.Position) + b.TimeStamp.String() + string(bytes) + b.Hash

	hash := sha256.New()

	hash.Write([]byte(data))

	b.Hash = hex.EncodeToString(hash.Sum(nil))
}

func (b *Block) validateHash(hash string) bool {
	b.generateHash()
	if b.Hash != hash {
		return false
	}

	return true
}

func CreateBlock(prev *Block, item BookCheckout) *Block {
	block := &Block{}
	block.Position = prev.Position + 1
	block.PreviousHash = prev.Hash
	block.Data = item
	block.TimeStamp = time.Now()
	block.generateHash()

	return block
}

type Blockchain struct {
	blocks []*Block
}

func ValidBlock(block *Block, prev *Block) bool {
	if prev.Hash != block.PreviousHash {
		return false
	}

	//if !block.validateHash(block.Hash) {
	//	fmt.Println("erro 2")
	//	return false
	//}

	if block.Position-1 != prev.Position {
		return false
	}

	return true
}

func (b *Blockchain) AddBlock(data BookCheckout) {
	previousBlock := b.blocks[len(b.blocks)-1]

	block := CreateBlock(previousBlock, data)

	if ValidBlock(block, previousBlock) {
		b.blocks = append(b.blocks, block)
	}
}

var Chain *Blockchain

func GetBlockchain(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.MarshalIndent(Chain.blocks, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	io.WriteString(w, string(bytes))
}

func WriteBlock(w http.ResponseWriter, r *http.Request) {
	var checkout BookCheckout

	if err := json.NewDecoder(r.Body).Decode(&checkout); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	Chain.AddBlock(checkout)
	resp, err := json.MarshalIndent(checkout, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not write block"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func CreateBook(w http.ResponseWriter, r *http.Request) {
	var book Book

	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	h := md5.New()
	io.WriteString(h, book.ISBN)
	book.ID = fmt.Sprintf("%x", h.Sum(nil))

	response, err := json.MarshalIndent(book, "", "  ")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println(err)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, BookCheckout{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func main() {
	Chain = NewBlockchain()
	r := mux.NewRouter()
	r.HandleFunc("/", GetBlockchain).Methods("GET")
	r.HandleFunc("/", WriteBlock).Methods("POST")
	r.HandleFunc("/book", CreateBook).Methods("POST")

	go func() {
		for _, block := range Chain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PreviousHash)
			bytes, _ := json.MarshalIndent(block, "", "  ")
			fmt.Println(string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Println()
		}
	}()

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
