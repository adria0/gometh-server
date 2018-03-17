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
	err := mainContract.Abi.Unpack(&event, "Log", eventlog.Data)
	if err != nil {
		return err
	}

	log.Printf("contractlog %#v\n", event)

	return nil
}

func serverStart() {

	// -- register event handlers & start processing

	assert(mainClient.RegisterEventHandler(mainContract, "LogLock", handleLockEvent))
	assert(mainClient.RegisterEventHandler(mainContract, "Log", handleLogEvent))

	assert(sideClient.RegisterEventHandler(sideContract, "Log", handleLogEvent))
	assert(sideClient.RegisterEventHandler(sideContract, "LogBurn", handleBurnEvent))
	assert(sideClient.RegisterEventHandler(sideContract, "LogBurnMultisigned", handleBurnMultisignedEvent))
	assert(sideClient.RegisterEventHandler(sideContract, "LogStateChangeMultisigned", handleStateChangeMultisigned))
	assert(sideClient.RegisterEventHandler(sideContract, "LogMintMultisigned", handleMintMultisigned))

	assert(sideClient.RegisterEventHandler(wethContract, "StateChange", handleStateChange))
	assert(sideClient.RegisterEventHandler(wethContract, "Transfer", handleTransferEvent))
	assert(sideClient.RegisterEventHandler(wethContract, "Log", handleLogEvent))

	cterminate := make(chan bool)
	cterminated := make(chan bool)
	pterminate := make(chan bool)
	pterminated := make(chan bool)

	sideClient.HandleEvents(cterminate, cterminated)
	mainClient.HandleEvents(pterminate, pterminated)

	<-time.After(time.Second * 3600)
}
