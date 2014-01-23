package ethchain

import (
	"github.com/ethereum/ethutil-go"
)

/*
 * This is the special genesis block.
 */

var GenisisHeader = []interface{}{
	// Previous hash (none)
	"",
	// Sha of uncles
	string(ethutil.Sha256Bin(ethutil.Encode([]interface{}{}))),
	// Coinbase
	"",
	// Root state
	"",
	// Sha of transactions
	string(ethutil.Sha256Bin(ethutil.Encode([]interface{}{}))),
	// Difficulty
	ethutil.BigPow(2, 26),
	// Time
	uint64(1),
	// Nonce
	ethutil.Big("0"),
	// Extra
	"",
}

var Genesis = []interface{}{GenisisHeader, []interface{}{}, []interface{}{}}
