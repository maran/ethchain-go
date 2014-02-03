package ethchain

import (
	"github.com/ethereum/ethutil-go"
)

/*
 * This is the special genesis block.
 */

var ZeroHash256 = make([]byte, 32)
var ZeroHash160 = make([]byte, 20)
var EmptyShaList = ethutil.Sha3Bin(ethutil.Encode([]interface{}{}))

var GenisisHeader = []interface{}{
	// Previous hash (none)
	"",
	// Sha of uncles
	EmptyShaList,
	// Coinbase
	"",
	// Root state
	"",
	// Sha of transactions
	EmptyShaList,
	// Difficulty
	ethutil.BigPow(2, 22),
	// Time
	uint64(0),
	// Nonce
	nil,
	// Extra
	"",
}

var Genesis = []interface{}{GenisisHeader, []interface{}{}, []interface{}{}}
