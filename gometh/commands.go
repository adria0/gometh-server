package gometh

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func callLock(value *big.Int) error {
	_, _, err := mainContract.SendTransactionSync(value, 0, "lock")

	return err
}

func callBurn(value *big.Int) error {

	var txid [32]byte
	terminate := make(chan bool)
	terminated := make(chan bool)

	assert(sideClient.RegisterEventHandler(sideContract, "LogBurnMultisigned", func(eventlog *types.Log) error {
		log.Printf("RECV callBurn_LogBurnMultisigned")

		type LogBurnMultisigned struct {
			Txid  [32]byte
			From  common.Address
			Value *big.Int
		}

		var event LogBurnMultisigned
		err := sideContract.Abi.Unpack(&event, "LogBurnMultisigned", eventlog.Data)
		if err != nil {
			return err
		}

		type GetSignatures struct {
			Epoch *big.Int
			Data  []byte
			Sigs  [][32]byte
		}

		var output GetSignatures
		if sideContract.Call(&output, "getSignatures", txid) != nil {
			return err
		}

		log.Println("GOT VOUCHER")
		log.Println("	---------------------------------------- ")
		log.Println("	TO    : ", event.From.Hex())
		log.Println("	VALUE : ", event.Value)
		log.Println("	---------------------------------------- ")
		log.Println("	EPOCH : ", output.Epoch)
		log.Println("	DATA  : ", hex.EncodeToString(output.Data))
		for _, v := range output.Sigs {
			log.Println("	SIG  : ", hex.EncodeToString(v[:]))
		}

		terminate <- true

		return nil
	}))
	sideClient.HandleEvents(terminate, terminated)

	tx, _, err := sideContract.SendTransactionSync(big.NewInt(0), 0, "burn", value)
	if err != nil {
		return err
	}

	topicID := sideContract.Abi.Events["LogBurn"].Id()

	copy(txid[:], crypto.Keccak256(tx.Hash().Bytes(), topicID.Bytes()))

	fmt.Println("Burn called, receipt id=", hex.EncodeToString(txid[:]))

	<-terminated

	return err
}
