package gometh

import (
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func handleBurnEvent(eventlog *types.Log) error {

	type BurnEvent struct {
		Epoch *big.Int
		From  common.Address
		Value *big.Int
	}

	var event BurnEvent
	err := sideContract.Abi.Unpack(&event, "LogBurn", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("RECV LogBurn")
	log.Printf("SEND partialExecuteOff _burnmultisigned")

	_, err = sideContract.PartialExecuteOff(
		eventlog, big.NewInt(0), 4000000,
		"_burnmultisigned", event.From, event.Value,
	)

	return err
}

func handleBurnMultisignedEvent(eventlog *types.Log) error {

	log.Printf("RECV LogBurnMultisigned")

	return nil
}

func handleStateChange(eventlog *types.Log) error {

	log.Printf("RECV StateChangeEvent")

	type StateChangeEvent struct {
		BlockNo   *big.Int
		RootState [32]byte
	}

	var event StateChangeEvent
	err := wethContract.Abi.Unpack(&event, "StateChange", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("SEND partialExecuteOff _statechangemultisigned")

	_, err = sideContract.PartialExecuteOff(
		eventlog, big.NewInt(0), 4000000,
		"_statechangemultisigned", event.BlockNo, event.RootState,
	)

	return err
}

func handleStateChangeMultisigned(eventlog *types.Log) error {

	type StateChangeMultisignedEvent struct {
		TxID      [32]byte
		BlockNo   *big.Int
		RootState [32]byte
	}

	log.Printf("RECV StateChangeMultisigned")

	return nil
}

func handleMintMultisigned(eventlog *types.Log) error {

	type MintMultisignedEvent struct {
		TxID  [32]byte
		To    common.Address
		Value *big.Int
	}

	var event MintMultisignedEvent
	err := sideContract.Abi.Unpack(&event, "LogMintMultisigned", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("RECV MintMultisigned %v %v wei\n", event.To.Hex(), event.Value)

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

	log.Printf("RECV Transfer %v %v->%v\n", event.Value, from.Hex(), to.Hex())

	return nil
}
