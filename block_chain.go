package ethchain

import (
	"bytes"
	"github.com/ethereum/ethutil-go"
	"math"
	"math/big"
)

type BlockChain struct {
	// The famous, the fabulous Mister GENESIIIIIIS (block)
	genesisBlock *Block
	// Last known total difficulty
	TD *big.Int

	LastBlockNumber uint64

	CurrentBlock  *Block
	LastBlockHash []byte
}

func NewBlockChain() *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = NewBlockFromData(ethutil.Encode(Genesis))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())

	return bc
}

func (bc *BlockChain) HasBlock(hash []byte) bool {
	data, _ := ethutil.Config.Db.Get(hash)
	return len(data) != 0
}

func (bc *BlockChain) GenesisBlock() *Block {
	return bc.genesisBlock
}

// Get chain return blocks from hash up to max in RLP format
func (bc *BlockChain) GetChainFromHash(hash []byte, max uint64) []interface{} {
	var chain []interface{}
	h := bc.CurrentBlock.Hash()
	parentHash := hash // parent.Hash()
	lastNumber := bc.BlockInfo(bc.CurrentBlock).Number
	parentNumber := bc.BlockInfoByHash(hash).Number
	count := uint64(math.Min(float64(lastNumber-parentNumber), float64(max)))
	startNumber := parentNumber + count

	num := lastNumber
	for ; num > startNumber; h = bc.GetBlock(h).PrevHash {
		num--
	}
	for i := uint64(0); bytes.Compare(h, parentHash) != 0 && num > parentNumber && i < count; i++ {
		block := bc.GetBlock(h)
		h = block.PrevHash

		chain = append(chain, block.RlpData())

		num--
	}

	return chain
}

func (bc *BlockChain) Add(block *Block) {
	bc.writeBlockInfo(block)

	// Prepare the genesis block
	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	ethutil.Config.Db.Put(block.Hash(), block.RlpEncode())
}

func (bc *BlockChain) GetBlock(hash []byte) *Block {
	data, _ := ethutil.Config.Db.Get(hash)

	return NewBlockFromData(data)
}

func (bc *BlockChain) BlockInfoByHash(hash []byte) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(hash, []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (bc *BlockChain) BlockInfo(block *Block) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
func (bc *BlockChain) writeBlockInfo(block *Block) {
	bc.LastBlockNumber++
	bi := BlockInfo{Number: bc.LastBlockNumber, Hash: block.Hash()}

	// For now we use the block hash with the words "info" appended as key
	ethutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}
