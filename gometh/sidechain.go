package gometh

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func callBurn(value *big.Int) error {
	_, _, err := childContract.SendTransactionSync(big.NewInt(0), "burn", value)
	return err
}

func handleBurnEvent(eventlog *types.Log) error {

	type BurnEvent struct {
		Epoch *big.Int
		From  common.Address
		Value *big.Int
	}

	var event BurnEvent
	err := childContract.Abi.Unpack(&event, "LogBurn", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("LogBurn")

	burnmsg, err := childContract.Abi.Pack("_burnmultisigned", event.From, event.Value)
	if err != nil {
		return err
	}

	var txhash [32]byte
	copy(txhash[:], eventlog.TxHash.Bytes())

	log.Printf("partialExecuteOff _burnmultisigned")

	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOn", event.Epoch, txhash, burnmsg,
	)

	return err

}

func handleBurnMultisignedEvent(eventlog *types.Log) error {

	log.Printf("LogBurnMultisigned")

	return nil

}

func handleStateChange(eventlog *types.Log) error {

	type StateChangeEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	epoch := big.NewInt(0)
	txid := common.BytesToHash(eventlog.TxHash.Bytes())

	var event StateChangeEvent
	err := wethContract.Abi.Unpack(&event, "StateChange", eventlog.Data)
	if err != nil {
		return err
	}

	msg, err := childContract.Abi.Pack("_statechangemultisigned", event.BlockNo, event.RootState)
	if err != nil {
		return err
	}
	sig, err := sign(childClient, abi.U256(epoch), txid[:], msg)

	if err != nil {
		return err
	}

	log.Printf("partialExecuteOff _statechangemultisigned")
	_, _, err = childContract.SendTransactionSync(
		big.NewInt(0),
		"partialExecuteOff", epoch, txid, msg, sig,
	)

	return err
}

func handleStateChangeMultisigned(eventlog *types.Log) error {

	type StateChangeMultisignedEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	log.Printf("StateChangeMultisigned")

	return nil
}

func handleMintMultisigned(eventlog *types.Log) error {

	type MintMultisignedEvent struct {
		To    common.Address
		Value *big.Int
	}

	var event MintMultisignedEvent
	err := childContract.Abi.Unpack(&event, "LogMintMultisigned", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("MintMultisigned %v %v wei\n", event.To.Hex(), event.Value)

	return nil
}

func handleTransferEvent(eventlog *types.Log) error {

	type TransferEvent struct {
		Value *big.Int
	}

	var event TransferEvent
	err := wethContract.Abi.Unpack(&event, "Transfer", eventlog.Data)
	if err != nil {
		return err
	}

	from := common.BytesToAddress(eventlog.Topics[1][:])
	to := common.BytesToAddress(eventlog.Topics[2][:])

	log.Printf("WTransfer %v %v->%v\n", event.Value, from.Hex(), to.Hex())

	return nil
}
