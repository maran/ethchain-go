package ethchain

import (
	"math/big"
)

var StepFee *big.Int = new(big.Int)
var TxFee *big.Int = new(big.Int)
var ContractFee *big.Int = new(big.Int)
var MemFee *big.Int = new(big.Int)
var DataFee *big.Int = new(big.Int)
var CryptoFee *big.Int = new(big.Int)
var ExtroFee *big.Int = new(big.Int)

var Period1Reward *big.Int = new(big.Int)
var Period2Reward *big.Int = new(big.Int)
var Period3Reward *big.Int = new(big.Int)
var Period4Reward *big.Int = new(big.Int)

func InitFees() {
	// Base for 2**64
	b60 := new(big.Int)
	b60.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
	// Base for 2**80
	b80 := new(big.Int)
	b80.Exp(big.NewInt(2), big.NewInt(80), big.NewInt(0))

	StepFee.Exp(big.NewInt(10), big.NewInt(16), big.NewInt(0))
	//StepFee.Div(b60, big.NewInt(64))
	//fmt.Println("StepFee:", StepFee)

	TxFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
	//fmt.Println("TxFee:", TxFee)

	ContractFee.Exp(big.NewInt(2), big.NewInt(64), big.NewInt(0))
	//fmt.Println("ContractFee:", ContractFee)

	MemFee.Div(b60, big.NewInt(4))
	//fmt.Println("MemFee:", MemFee)

	DataFee.Div(b60, big.NewInt(16))
	//fmt.Println("DataFee:", DataFee)

	CryptoFee.Div(b60, big.NewInt(16))
	//fmt.Println("CrytoFee:", CryptoFee)

	ExtroFee.Div(b60, big.NewInt(16))
	//fmt.Println("ExtroFee:", ExtroFee)

	Period1Reward.Mul(b80, big.NewInt(1024))
	//fmt.Println("Period1Reward:", Period1Reward)

	Period2Reward.Mul(b80, big.NewInt(512))
	//fmt.Println("Period2Reward:", Period2Reward)

	Period3Reward.Mul(b80, big.NewInt(256))
	//fmt.Println("Period3Reward:", Period3Reward)

	Period4Reward.Mul(b80, big.NewInt(128))
	//fmt.Println("Period4Reward:", Period4Reward)
}
