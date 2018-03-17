package eth

import (
	"context"
	"encoding/hex"
	"log"
	"math/big"
	"sync"
	"time"

	cfg "github.com/adriamb/gometh-server/gometh/config"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"fmt"
)

var (
	// ErrReceiptStatusFailed when recieving a failed transaction
	ErrReceiptStatusFailed = fmt.Errorf("ReceiptStatusFailed")
	// ErrReceiptNotRecieved when unable to retrieve a transaction
	ErrReceiptNotRecieved = fmt.Errorf("ErrReceiptNotRecieved")
)

type EventHandlerFunc func(*types.Log) error

// EventHandler associates a function to an event
type EventHandler struct {
	Address        common.Address
	EventSignature string
	Topic          string
	Handler        EventHandlerFunc
}

// Web3Client defines a connection to a client via websockets
type Web3Client struct {
	ClientMutex    *sync.Mutex
	Client         *ethclient.Client
	Account        accounts.Account
	Ks             *keystore.KeyStore
	ReceiptTimeout time.Duration
	EventHandlers  []EventHandler
}

// NewWeb3Client creates a client, using a keystore and an account for transactions
func NewWeb3Client(rpcURL string, ks *keystore.KeyStore, account accounts.Account) (*Web3Client, error) {

	var err error

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &Web3Client{
		Client:         client,
		Ks:             ks,
		Account:        account,
		ReceiptTimeout: 120 * time.Second,
		EventHandlers:  []EventHandler{},
	}, nil
}

// AccountInfo retieves information about the default account
func (b *Web3Client) AccountInfo() (string, error) {

	address := b.Account.Address.Hex()
	ctx := context.TODO()
	balance, err := b.Client.BalanceAt(ctx, b.Account.Address, nil)
	if err != nil {

		return "", err
	}
	return address + "=" + balance.String() + " wei", nil
}

// SendTransactionSync executes a contract method and wait it finalizes
func (b *Web3Client) SendTransactionSync(to *common.Address, value *big.Int, gasLimit uint64, calldata []byte) (*types.Transaction, *types.Receipt, error) {

	b.ClientMutex.Lock()
	defer b.ClientMutex.Unlock()

	var err error
	var tx *types.Transaction
	var receipt *types.Receipt

	ctx := context.TODO()

	network, err := b.Client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err
	}

	gasPrice, err := b.Client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, nil, err
	}

	callmsg := ethereum.CallMsg{
		From:  b.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	}

	if gasLimit == 0 {
		gasLimit, err = b.Client.EstimateGas(ctx, callmsg)
		if err != nil {
			if cfg.Verbose > 0 {
				log.Printf("Failed EstimateGas from=%v to=%v value=%v data=%v",
					callmsg.From.Hex(), callmsg.To.Hex(),
					callmsg.Value, hex.EncodeToString(callmsg.Data),
				)
			}
			return nil, nil, err
		}
	}

	nonce, err := b.Client.NonceAt(ctx, b.Account.Address, nil)
	if err != nil {
		return nil, nil, err
	}

	if to == nil {
		tx = types.NewContractCreation(
			nonce,    // nonce int64
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	} else {
		tx = types.NewTransaction(
			nonce,    // nonce int64
			*to,      // to common.Address
			value,    // amount *big.Int
			gasLimit, // gasLimit *big.Int
			gasPrice, // gasPrice *big.Int
			calldata, // data []byte
		)
	}

	if tx, err = b.Ks.SignTx(b.Account, tx, network); err != nil {
		return nil, nil, err
	}

	if cfg.Verbose > 0 {
		log.Println(tx.String())
	}

	if err = b.Client.SendTransaction(ctx, tx); err != nil {
		return nil, nil, err
	}

	start := time.Now()
	for receipt == nil && time.Now().Sub(start) < b.ReceiptTimeout {
		receipt, err = b.Client.TransactionReceipt(ctx, tx.Hash())
		if receipt == nil {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if receipt != nil && receipt.Status == types.ReceiptStatusFailed {
		log.Println("FAILED RECEIPT TX", receipt.String())
		return tx, receipt, ErrReceiptStatusFailed
	}

	if receipt == nil {
		log.Println("FAILED TX", tx.String())
		return tx, receipt, ErrReceiptNotRecieved
	}

	return tx, receipt, err
}

// Call an constant method
func (b *Web3Client) Call(to *common.Address, value *big.Int, calldata []byte) ([]byte, error) {

	ctx := context.TODO()

	msg := ethereum.CallMsg{
		From:  b.Account.Address,
		To:    to,
		Value: value,
		Data:  calldata,
	}

	return b.Client.CallContract(ctx, msg, nil)
}

// RegisterEventHandler registers a function to be called on event emission
func (b *Web3Client) RegisterEventHandler(contract *Contract, event string, handler EventHandlerFunc) error {

	abievent, ok := contract.Abi.Events[event]
	if !ok {
		return fmt.Errorf("Event %v not found", event)
	}
	topicID := abievent.Id()

	eventHandler := EventHandler{
		Address:        *contract.Address,
		EventSignature: abievent.String(),
		Topic:          "0x" + hex.EncodeToString(topicID[:]),
		Handler:        handler,
	}

	b.EventHandlers = append(b.EventHandlers, eventHandler)
	return nil
}

func debugLog(eventlog *types.Log) {
	log.Println("Log from address", eventlog.Address.Hex())
	for c, t := range eventlog.Topics {
		log.Printf("  Topic[%v]: %v", c, t.Hex())
	}
	log.Println("  Data:", hex.EncodeToString(eventlog.Data))
}

// HandleEvents starts processing event handling
func (b *Web3Client) HandleEvents(terminatech, terminatedch chan bool) error {

	ctx := context.TODO()
	ch := make(chan types.Log)

	addrs := []common.Address{}

	for _, v := range b.EventHandlers {
		found := false
		for _, addr := range addrs {
			if addr == v.Address {
				found = true
				break
			}
		}
		if !found {
			addrs = append(addrs, v.Address)
		}
	}

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(0),
		ToBlock:   big.NewInt(10000000),
		Addresses: addrs,
		Topics:    [][]common.Hash{{}},
	}

	_, err := b.Client.SubscribeFilterLogs(ctx, query, ch)
	if err != nil {
		return err
	}

	processEvent := func(logevent *types.Log) {
		if logevent.Removed {
			return
		}
		for _, v := range b.EventHandlers {
			if logevent.Address == v.Address && logevent.Topics[0].Hex() == v.Topic {
				if v.Handler != nil {
					err := v.Handler(logevent)
					if err != nil {
						log.Println("[EventProcessingFailed]", v.EventSignature, err)
					}
				} else {
					log.Println("[Event] ", v.EventSignature)
				}
				return
			}
		}
	}

	go func() {
		for true {
			select {
			case logevent := <-ch:
				go processEvent(&logevent)
			case <-terminatech:
				terminatedch <- true
				return
			}
		}
	}()

	return nil
}
