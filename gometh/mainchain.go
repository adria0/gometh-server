package gometh

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func callLock(value *big.Int) error {
	_, _, err := parentContract.SendTransactionSync(value, "lock")
	return err
}

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

	log.Printf("LockEvent %v %v wei", event.From.Hex(), event.Value)

	mintmsg, err := childContract.Abi.Pack("_mintmultisigned", event.From, event.Value)
	if err != nil {
		return err
	}

	var txhash [32]byte
	copy(txhash[:], eventlog.TxHash.Bytes())

	log.Printf("partialExecuteOff _mint")

	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOn", event.Epoch, txhash, mintmsg,
	)

	return err

}
