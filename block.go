package ethchain

import (
	"fmt"
	"github.com/ethereum/ethutil-go"
	"math/big"
	"time"
)

type BlockInfo struct {
	Number *big.Int
}

func (bi *BlockInfo) RlpDecode(data []byte) {
	decoder := ethutil.NewRlpDecoder(data)
	bi.Number = decoder.Get(0).AsBigInt()
}

func (bi *BlockInfo) RlpEncode() []byte {
	return ethutil.Encode([]interface{}{bi.Number})
}

type Block struct {
	// Hash to the previous block
	PrevHash string
	// Uncles of this block
	Uncles   []*Block
	UncleSha []byte
	// The coin base address
	Coinbase string
	// Block Trie state
	state *ethutil.Trie
	// Difficulty for the current block
	Difficulty *big.Int
	// Creation time
	Time int64
	// Block Nonce for verification
	Nonce *big.Int
	// List of transactions and/or contracts
	transactions []*Transaction
	TxSha        []byte
	// Extra (unused)
	extra string
}

// New block takes a raw encoded string
func NewBlockFromData(raw []byte) *Block {
	block := &Block{}
	block.RlpDecode(raw)

	return block
}

// New block takes a raw encoded string
func NewBlockFromRlpValue(rlpValue *ethutil.RlpValue) *Block {
	block := &Block{}
	block.RlpValueDecode(rlpValue)

	return block
}

// Creates a new block. This is currently for testing
func CreateTestBlock( /* TODO use raw data */ transactions []*Transaction) *Block {
	block := &Block{
		// Slice of transactions to include in this block
		transactions: transactions,
		PrevHash:     "1234",
		Coinbase:     "me",
		Difficulty:   big.NewInt(10),
		Nonce:        ethutil.BigInt0,
		Time:         time.Now().Unix(),
	}

	return block
}

func CreateBlock(root string,
	prevHash []byte,
	base string,
	Difficulty *big.Int,
	Nonce *big.Int,
	extra string,
	txes []*Transaction) *Block {

	block := &Block{
		// Slice of transactions to include in this block
		transactions: txes,
		PrevHash:     string(prevHash),
		Coinbase:     base,
		Difficulty:   Difficulty,
		Nonce:        Nonce,
		Time:         time.Now().Unix(),
		extra:        extra,
		// TODO
		Uncles: []*Block{},
	}
	block.state = ethutil.NewTrie(ethutil.Config.Db, root)

	for _, tx := range txes {
		block.MakeContract(tx)
	}

	return block
}

func (block *Block) State() *ethutil.Trie {
	return block.state
}

func (block *Block) Transactions() []*Transaction {
	return block.transactions
}

func (block *Block) GetContract(addr []byte) *Contract {
	data := block.state.Get(string(addr))
	if data == "" {
		return nil
	}

	contract := &Contract{}
	contract.RlpDecode([]byte(data))

	return contract
}

func (block *Block) UpdateContract(addr []byte, contract *Contract) {
	block.state.Update(string(addr), string(contract.RlpEncode()))
}

func (block *Block) PayFee(addr []byte, fee *big.Int) bool {
	contract := block.GetContract(addr)
	// If we can't pay the fee return
	if contract == nil || contract.Amount.Cmp(fee) < 0 /* amount < fee */ {
		fmt.Println("Contract has insufficient funds", contract.Amount, fee)

		return false
	}

	base := new(big.Int)
	contract.Amount = base.Sub(contract.Amount, fee)
	block.state.Update(string(addr), string(contract.RlpEncode()))

	data := block.state.Get(string(block.Coinbase))

	// Get the ether (Coinbase) and add the fee (gief fee to miner)
	ether := NewAddressFromData([]byte(data))

	base = new(big.Int)
	ether.Amount = base.Add(ether.Amount, fee)

	block.state.Update(string(block.Coinbase), string(ether.RlpEncode()))

	return true
}

func (block *Block) BlockInfo() BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Returns a hash of the block
func (block *Block) Hash() []byte {
	header, _, _ := block.Make()
	return ethutil.Sha3Bin(ethutil.Encode(header))
}

func (block *Block) Make() (interface{}, []string, interface{}) {
	// Marshal the transactions of this block
	encTx := make([]string, len(block.transactions))
	for i, tx := range block.transactions {
		// Cast it to a string (safe)
		encTx[i] = string(tx.RlpEncode())
	}
	tsha := ethutil.Sha3Bin([]byte(ethutil.Encode(encTx)))

	uncles := make([]interface{}, len(block.Uncles))
	for i, uncle := range block.Uncles {
		uncles[i] = uncle.uncleHeader()
	}

	// Sha of the concatenated uncles
	usha := ethutil.Sha3Bin(ethutil.Encode(uncles))

	return block.header(tsha, usha), encTx, uncles
}

func (block *Block) RlpData() interface{} {
	// The block header
	header, encTx, uncles := block.Make()

	return []interface{}{header, encTx, uncles}
}

func (block *Block) RlpEncode() []byte {

	// Encode a slice interface which contains the header and the list of
	// transactions.
	return ethutil.Encode(block.RlpData())
}

func (block *Block) RlpDecode(data []byte) {
	rlpValue := ethutil.NewRlpDecoder(data)
	block.RlpValueDecode(rlpValue)
}

func (block *Block) RlpValueDecode(decoder *ethutil.RlpValue) {
	header := decoder.Get(0)

	block.PrevHash = header.Get(0).AsString()
	block.UncleSha = header.Get(1).AsBytes()
	block.Coinbase = header.Get(2).AsString()
	block.state = ethutil.NewTrie(ethutil.Config.Db, header.Get(3).AsString())
	block.TxSha = header.Get(4).AsBytes()
	block.Difficulty = header.Get(5).AsBigInt()
	block.Time = int64(header.Get(6).AsUint())
	block.Nonce = header.Get(7).AsBigInt()

	// Tx list might be empty if this is an uncle. Uncles only have their
	// header set.
	if decoder.Get(1).IsNil() == false { // Yes explicitness
		txes := decoder.Get(1)
		block.transactions = make([]*Transaction, txes.Length())
		for i := 0; i < txes.Length(); i++ {
			tx := &Transaction{}
			tx.RlpDecode(txes.Get(i).AsBytes())
			block.transactions[i] = tx

			if ethutil.Config.Debug {
				ethutil.Config.Db.Put(tx.Hash(), ethutil.Encode(tx))
			}
		}

	}

	if decoder.Get(2).IsNil() == false { // Yes explicitness
		uncles := decoder.Get(2)
		block.Uncles = make([]*Block, uncles.Length())
		for i := 0; i < uncles.Length(); i++ {
			block := &Block{}
			// This is terrible but it's the way it has to be since
			// I'm going by now means doing it by hand (the data is in it's raw format in interface form)
			block.RlpDecode(ethutil.Encode(uncles.Get(i).AsRaw()))
			block.Uncles[i] = block
		}
	}

}

func (block *Block) MakeContract(tx *Transaction) {
	// Create contract if there's no recipient
	if tx.IsContract() {
		addr := tx.Hash()

		value := tx.Value
		contract := NewContract(value, []byte(""))
		block.state.Update(string(addr), string(contract.RlpEncode()))
		for i, val := range tx.Data {
			contract.state.Update(string(ethutil.NumberToBytes(uint64(i), 32)), val)
		}
		block.UpdateContract(addr, contract)
	}
}

func (block *Block) String() string {
	return fmt.Sprintf("Block(%x):\nPrevHash:%x\nUncleSha:%x\nCoinbase:%v\nRoot:%x\nTxSha:%x\nDiff:%v\nTime:%d\nNonce:%d", block.Hash(), block.PrevHash, block.UncleSha, block.Coinbase, block.state.Root, block.TxSha, block.Difficulty, block.Time, block.Nonce)
}

//////////// UNEXPORTED /////////////////
func (block *Block) header(txSha []byte, uncleSha []byte) []interface{} {
	return []interface{}{
		// Sha of the previous block
		[]byte(block.PrevHash),
		// Sha of uncles
		uncleSha,
		// Coinbase address
		[]byte(block.Coinbase),
		// root state
		[]byte(block.state.Root),
		// Sha of tx
		txSha,
		// Current block Difficulty
		block.Difficulty,
		// Time the block was found?
		uint64(block.Time),
		// Block's Nonce for validation
		block.Nonce,
	}
}

func (block *Block) uncleHeader() []interface{} {
	return []interface{}{
		// Sha of the previous block
		block.PrevHash,
		// Sha of uncles
		block.UncleSha,
		// Coinbase address
		block.Coinbase,
		// root state
		block.state.Root,
		// Sha of tx
		block.TxSha,
		// Current block Difficulty
		block.Difficulty,
		// Time the block was found?
		uint64(block.Time),
		// Block's Nonce for validation
		block.Nonce,
	}
}
