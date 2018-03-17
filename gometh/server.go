package gometh

import (
	"log"
	"time"

	eth "github.com/adriamb/gometh-server/gometh/eth"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func sign(client *eth.Web3Client, data ...[]byte) ([3][32]byte, error) {
	web3SignaturePrefix := []byte("\x19Ethereum Signed Message:\n32")

	hash := crypto.Keccak256(data...)
	prefixedHash := crypto.Keccak256(web3SignaturePrefix, hash)

	var ret [3][32]byte

	// The produced signature is in the [R || S || V] format where V is 0 or 1.
	sig, err := client.Ks.SignHash(client.Account, prefixedHash)
	if err != nil {
		return ret, err
	}

	// We need to convert it to the format []uint256 = {v,r,s} format
	ret[0][31] = sig[64] + 27
	copy(ret[1][:], sig[0:32])
	copy(ret[2][:], sig[32:64])
	return ret, nil
}

func handleLogEvent(eventlog *types.Log) error {

	var event string
	err := parentContract.Abi.Unpack(&event, "Log", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("contractlog %#v\n", event)

	return nil
}

func serverStart() {

	setContractsAddress()

	// -- register event handlers & start processing

	assert(parentClient.RegisterEventHandler(parentContract, "LogLock", handleLockEvent))
	assert(parentClient.RegisterEventHandler(parentContract, "Log", handleLogEvent))

	assert(childClient.RegisterEventHandler(childContract, "Log", handleLogEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogBurn", handleBurnEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogBurnMultisigned", handleBurnMultisignedEvent))
	assert(childClient.RegisterEventHandler(childContract, "LogStateChangeMultisigned", handleStateChangeMultisigned))
	assert(childClient.RegisterEventHandler(childContract, "LogMintMultisigned", handleMintMultisigned))

	assert(childClient.RegisterEventHandler(wethContract, "StateChange", handleStateChange))
	assert(childClient.RegisterEventHandler(wethContract, "Transfer", handleTransferEvent))
	assert(childClient.RegisterEventHandler(wethContract, "Log", handleLogEvent))

	childClient.HandleEvents()
	parentClient.HandleEvents()

	<-time.After(time.Second * 3600)
}
