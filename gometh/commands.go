package gometh

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func callLock(value *big.Int) error {
	_, _, err := parentContract.SendTransactionSync(value, 0, "lock")

	return err
}

func callBurn(value *big.Int) error {

	var txid [32]byte

	assert(childClient.RegisterEventHandler(childContract, "LogBurnMultisigned", func(eventlog *types.Log) error {
		log.Printf("RECV callBurn_LogBurnMultisigned")

		data, err := childContract.Call("getSignatures", txid)
		if err != nil {
			return err
		}

		log.Println(data)

		return nil
	}))
	childClient.HandleEvents()

	tx, _, err := childContract.SendTransactionSync(big.NewInt(0), 0, "burn", value)
	if err != nil {
		return err
	}
	topicID := childContract.Abi.Events["LogBurnMultisigned"].Id()

	copy(txid[:], crypto.Keccak256(tx.Hash().Bytes(), topicID.Bytes()))

	fmt.Println("Burn called, receipt id=", hex.EncodeToString(txid[:]))

	<-time.After(time.Second * 3600)

	return err
}
