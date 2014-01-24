package ethchain

import (
	"bytes"
	"container/list"
	"errors"
	"github.com/ethereum/ethutil-go"
	"github.com/ethereum/ethwire-go"
	"log"
	"math/big"
	"sync"
)

const (
	txPoolQueueSize = 50
)

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
func (pool *TxPool) processTransaction(tx *Transaction) error {
	// Get the last block so we can retrieve the sender and receiver from
	// the merkle trie
	block := pool.BlockManager.bc.LastBlock
	// Something has gone horribly wrong if this happens
	if block == nil {
		return errors.New("No last block on the block chain")
	}

	var sender, receiver *Ether

	// Get the sender
	data := block.State().Get(string(tx.Sender()))
	// If it doesn't exist create a new account. Of course trying to send funds
	// from this account will fail since it will hold 0 Wei
	if data == "" {
		sender = NewEther(big.NewInt(0))
	} else {
		sender = NewEtherFromData([]byte(data))
	}
	// Defer the update. Whatever happens it should be persisted
	defer block.State().Update(string(tx.Sender()), string(sender.RlpEncode()))

	// Make sure there's enough in the sender's account. Having insufficient
	// funds won't invalidate this transaction but simple ignores it.
	if sender.Amount.Cmp(tx.Value) < 0 {
		if ethutil.Config.Debug {
			log.Println("Insufficient amount in sender's account. Adding 1 ETH for debug")
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
		receiver = NewEther(big.NewInt(0))
	} else {
		receiver = NewEtherFromData([]byte(data))
	}
	// Defer the update
	defer block.State().Update(tx.Recipient, string(receiver.RlpEncode()))

	// Add the amount to receivers account which should conclude this transaction
	receiver.Amount.Add(receiver.Amount, tx.Value)

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

			// Process the transaction
			err := pool.processTransaction(tx)
			if err != nil {
				log.Println("Error processing Tx", err)
			} else {
				// Call blocking version. At this point it
				// doesn't matter since this is a goroutine
				pool.addTransaction(tx)
			}
		case <-pool.quit:
			break out
		}
	}
}

func (pool *TxPool) QueueTransaction(tx *Transaction) {
	pool.queueChan <- tx
}

func (pool *TxPool) Flush() {
	pool.mutex.Lock()

	defer pool.mutex.Unlock()
}

func (pool *TxPool) Start() {
	go pool.queueHandler()
}

func (pool *TxPool) Stop() {
	log.Println("[TXP] Stopping...")

	close(pool.quit)

	pool.Flush()
}