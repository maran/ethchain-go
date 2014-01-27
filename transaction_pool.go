package ethchain

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"math/big"
	"sync"
)

const (
	txPoolQueueSize = 50
)

type TxPoolHook chan *Transaction

func FindTx(pool *list.List, finder func(*Transaction, *list.Element) bool) *Transaction {
	for e := pool.Front(); e != nil; e = e.Next() {
		if tx, ok := e.Value.(*Transaction); ok {
			if finder(tx, e) {
				return tx
			}
		}
	}

	return nil
}

type PublicSpeaker interface {
	Broadcast(msgType ethwire.MsgType, data interface{})
}

// The tx pool a thread safe transaction pool handler. In order to
// guarantee a non blocking pool we use a queue channel which can be
// independently read without needing access to the actual pool. If the
// pool is being drained or synced for whatever reason the transactions
// will simple queue up and handled when the mutex is freed.
type TxPool struct {
	//server *Server
	Speaker PublicSpeaker
	// The mutex for accessing the Tx pool.
	mutex sync.Mutex
	// Queueing channel for reading and writing incoming
	// transactions to
	queueChan chan *Transaction
	// Quiting channel
	quit chan bool
	// The actual pool
	pool *list.List

	BlockManager *BlockManager

	Hook TxPoolHook
}

func NewTxPool() *TxPool {
	return &TxPool{
		//server:    s,
		mutex:     sync.Mutex{},
		pool:      list.New(),
		queueChan: make(chan *Transaction, txPoolQueueSize),
		quit:      make(chan bool),
	}
}

// Blocking function. Don't use directly. Use QueueTransaction instead
func (pool *TxPool) addTransaction(tx *Transaction) {
	pool.mutex.Lock()
	pool.pool.PushBack(tx)
	pool.mutex.Unlock()

	// Broadcast the transaction to the rest of the peers
	pool.Speaker.Broadcast(ethwire.MsgTxTy, tx.RlpData())
}

// Process transaction validates the Tx and processes funds from the
// sender to the recipient.
func (pool *TxPool) ProcessTransaction(tx *Transaction, block *Block) error {
	log.Printf("[TXPL] Processing Tx %x\n", tx.Hash())

	var sender, receiver *Address

	// Get the sender
	data := block.State().Get(string(tx.Sender()))
	// If it doesn't exist create a new account. Of course trying to send funds
	// from this account will fail since it will hold 0 Wei
	if data == "" {
		sender = NewAddress(big.NewInt(0))
		// Create a new account for this sender
		block.State().Update(string(tx.Sender()), string(sender.RlpEncode()))
	} else {
		sender = NewAddressFromData([]byte(data))
	}

	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Amount.Cmp(tx.Value) < 0 {
		if ethutil.Config.Debug {
			log.Printf("Insufficient amount (ETH: %v) in sender's (%x) account. Adding 1 ETH for debug\n", sender.Amount, tx.Sender())
			sender.Amount = ethutil.BigPow(10, 18)
		} else {
			return errors.New("Insufficient amount in sender's account")
		}
	}

	// Subtract the amount from the senders account
	sender.Amount.Sub(sender.Amount, tx.Value)
	// Increment the nonce making each tx valid only once to prevent replay
	// attacks
	sender.Nonce += 1

	// Get the receiver
	data = block.State().Get(tx.Recipient)
	// If the receiver doesn't exist yet, create a new account to which the
	// funds will be send.
	if data == "" {
		receiver = NewAddress(big.NewInt(0))
		// Create a new account for the recipient
		block.State().Update(tx.Recipient, string(receiver.RlpEncode()))
	} else {
		receiver = NewAddressFromData([]byte(data))
	}
	// Add the amount to receivers account which should conclude this transaction
	receiver.Amount.Add(receiver.Amount, tx.Value)

	block.State().Update(string(tx.Sender()), string(sender.RlpEncode()))
	block.State().Update(tx.Recipient, string(receiver.RlpEncode()))

	return nil
}

func (pool *TxPool) ValidateTransaction(tx *Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	block := pool.BlockManager.CurrentBlock
	// Something has gone horribly wrong if this happens
	if block == nil {
		return errors.New("No last block on the block chain")
	}

	var sender *Address

	// Get the sender
	data := block.State().Get(string(tx.Sender()))
	// If it doesn't exist create a new account. Of course trying to send funds
	// from this account will fail since it will hold 0 Wei
	if data == "" {
		sender = NewAddress(big.NewInt(0))
	} else {
		sender = NewAddressFromData([]byte(data))
	}

	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Amount.Cmp(tx.Value) < 0 {
		if ethutil.Config.Debug {
			log.Printf("Insufficient amount (ETH: %v) in sender's (%x) account. Adding 1 ETH for debug\n", sender.Amount, tx.Sender())
		} else {
			return errors.New("Insufficient amount in sender's account")
		}
	}

	if sender.Nonce != tx.Nonce {
		if ethutil.Config.Debug {
			return fmt.Errorf("Invalid nonce %d(%d) continueing anyway", tx.Nonce, sender.Nonce)
		} else {
			return fmt.Errorf("Invalid nonce %d(%d)", tx.Nonce, sender.Nonce)
		}
	}

	return nil
}

func (pool *TxPool) queueHandler() {
out:
	for {
		select {
		case tx := <-pool.queueChan:
			hash := tx.Hash()
			foundTx := FindTx(pool.pool, func(tx *Transaction, e *list.Element) bool {
				return bytes.Compare(tx.Hash(), hash) == 0
			})

			if foundTx != nil {
				break
			}

			// Validate the transaction
			err := pool.ValidateTransaction(tx)
			if err != nil {
				if ethutil.Config.Debug {
					log.Println("Validating Tx failed", err)
				}
			} else {
				// Call blocking version. At this point it
				// doesn't matter since this is a goroutine
				pool.addTransaction(tx)

				if pool.Hook != nil {
					pool.Hook <- tx
				}
			}
		case <-pool.quit:
			break out
		}
	}
}

func (pool *TxPool) QueueTransaction(tx *Transaction) {
	pool.queueChan <- tx
}

func (pool *TxPool) Flush() []*Transaction {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	txList := make([]*Transaction, pool.pool.Len())
	i := 0
	for e := pool.pool.Front(); e != nil; e = e.Next() {
		if tx, ok := e.Value.(*Transaction); ok {
			txList[i] = tx
		}

		i++
	}

	// Recreate a new list all together
	// XXX Is this the fastest way?
	pool.pool = list.New()

	return txList
}

func (pool *TxPool) Start() {
	go pool.queueHandler()
}

func (pool *TxPool) Stop() {
	log.Println("[TXP] Stopping...")

	close(pool.quit)

	pool.Flush()
}
