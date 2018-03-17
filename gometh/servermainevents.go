package gometh

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func handleLockEvent(eventlog *types.Log) error {

	type LogLockEvent struct {
		Epoch *big.Int
		From  common.Address
		Value *big.Int
	}

	var event LogLockEvent
	err := parentContract.Abi.Unpack(&event, "LogLock", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("RECV LockEvent %v %v wei", event.From.Hex(), event.Value)

	log.Printf("SEND partialExecuteOn _mintmultisigned")

	mintmsg, err := childContract.Abi.Pack("_mintmultisigned", event.From, event.Value)
	if err != nil {
		return err
	}

	txhash := crypto.Keccak256(eventlog.TxHash.Bytes(), eventlog.Topics[0].Bytes())
	var txid [32]byte
	copy(txid[:], txhash)

	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0), 4000000,
		"partialExecuteOn", event.Epoch, txid, mintmsg,
	)

	if err == nil {
		log.Printf("RCPT partialExecuteOn _mintmultisigned")
	}
	return err

}
