package ethchain

import (
	"github.com/ethereum/ethutil-go"
	"github.com/obscuren/secp256k1-go"
	"math/big"
)

type Transaction struct {
	Nonce     uint64
	Recipient []byte
	Value     *big.Int
	Data      []string
	Memory    []int
	v         byte
	r, s      []byte
}

func NewTransaction(to []byte, value *big.Int, data []string) *Transaction {
	tx := Transaction{Recipient: to, Value: value}
	tx.Nonce = 0

	// Serialize the data
	tx.Data = make([]string, len(data))
	for i, val := range data {
		instr, err := ethutil.CompileInstr(val)
		if err != nil {
			//fmt.Printf("compile error:%d %v\n", i+1, err)
		}

		tx.Data[i] = instr
	}

	if ethutil.Config.Debug {
		// TMP
		tx.Sign([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	}

	return &tx
}

func NewTransactionFromData(data []byte) *Transaction {
	tx := &Transaction{}
	tx.RlpDecode(data)

	return tx
}

func NewTransactionFromRlpValue(rlpValue *ethutil.RlpValue) *Transaction {
	tx := &Transaction{}

	tx.Nonce = rlpValue.Get(0).AsUint()
	tx.Recipient = rlpValue.Get(1).AsBytes()
	tx.Value = rlpValue.Get(2).AsBigInt()

	d := rlpValue.Get(3)
	tx.Data = make([]string, d.Length())
	for i := 0; i < d.Length(); i++ {
		tx.Data[i] = d.Get(i).AsString()
	}

	tx.v = byte(rlpValue.Get(4).AsUint())
	tx.r = rlpValue.Get(5).AsBytes()
	tx.s = rlpValue.Get(6).AsBytes()

	return tx
}

func (tx *Transaction) Hash() []byte {
	preEnc := []interface{}{
		tx.Nonce,
		tx.Recipient,
		tx.Value,
		tx.Data,
	}

	return ethutil.Sha3Bin(ethutil.Encode(preEnc))
}

func (tx *Transaction) IsContract() bool {
	return len(tx.Recipient) == 0
}

func (tx *Transaction) Signature(key []byte) []byte {
	h := tx.Hash()
	hash := ethutil.Sha3Bin(h)

	sig, _ := secp256k1.Sign(hash, key)

	return sig
}

func (tx *Transaction) PublicKey() []byte {
	hash := ethutil.Sha3Bin(tx.Hash())

	sig := append(tx.r, tx.s...)
	sig = append(sig, tx.v-27)

	pubkey, _ := secp256k1.RecoverPubkey(hash, sig)

	return pubkey
}

func (tx *Transaction) Sender() []byte {
	pubkey := tx.PublicKey()

	// Validate the returned key.
	// Return nil if public key isn't in full format
	if pubkey[0] != 4 {
		return nil
	}

	return ethutil.Sha3Bin(pubkey)[12:]
}

func (tx *Transaction) Sign(privk []byte) {
	sig := tx.Signature(privk)

	tx.r = sig[:32]
	tx.s = sig[32:64]
	tx.v = sig[64] + 27
}

func (tx *Transaction) RlpData() interface{} {
	// Prepare the transaction for serialization
	return []interface{}{
		tx.Nonce,
		tx.Recipient,
		tx.Value,
		tx.Data,
		tx.v,
		tx.r,
		tx.s,
	}
}

func (tx *Transaction) RlpEncode() []byte {
	return ethutil.Encode(tx.RlpData())
}

func (tx *Transaction) RlpDecode(data []byte) {
	decoder := ethutil.NewRlpValueFromBytes(data)

	tx.Nonce = decoder.Get(0).AsUint()
	tx.Recipient = decoder.Get(1).AsBytes()
	tx.Value = decoder.Get(2).AsBigInt()

	d := decoder.Get(3)
	tx.Data = make([]string, d.Length())
	for i := 0; i < d.Length(); i++ {
		tx.Data[i] = d.Get(i).AsString()
	}

	// TODO something going wrong here
	tx.v = byte(decoder.Get(4).AsUint())
	tx.r = decoder.Get(5).AsBytes()
	tx.s = decoder.Get(6).AsBytes()
}
