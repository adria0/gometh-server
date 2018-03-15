package gometh

import (
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	eth "github.com/adriamb/gometh-server/gometh/eth"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func assert(err error) {
	if err != nil {
		panic("Failed: " + err.Error())
	}
}

var (
	parentClient   *eth.Web3Client
	childClient    *eth.Web3Client
	parentContract *eth.Contract
	childContract  *eth.Contract
	wethContract   *eth.Contract
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

func dotest() {
	//callLock(big.NewInt(1000))
	//callBurn(big.NewInt(10))
}

func serverInit() {
	// -- open keystore

	var err error
	var account accounts.Account

	ks := keystore.NewKeyStore(C.Keystore.Path, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) != 1 {
		panic(fmt.Sprintf("Not exact one account in keystore, was %v", len(ks.Accounts())))
	}
	account = ks.Accounts()[0]
	assert(ks.Unlock(account, C.Keystore.Passwd))

	// -- create clients

	parentClient, err = eth.NewWeb3Client(
		C.MainChain.RPCURL,
		ks,
		account,
	)
	assert(err)

	childClient, err = eth.NewWeb3Client(
		C.SideChain.RPCURL,
		ks,
		account,
	)
	assert(err)

	parentClient.ClientMutex = &sync.Mutex{}
	childClient.ClientMutex = parentClient.ClientMutex

	parentAccountInfo, err := parentClient.AccountInfo()
	assert(err)
	log.Println("Parent chain account: ", parentAccountInfo)

	childAccountInfo, err := childClient.AccountInfo()
	assert(err)
	log.Println("Child chain account", childAccountInfo)

	// -- load contracts
	parentContract, err = eth.NewContract(parentClient, C.Contracts.Path+"/GometParent.json")
	assert(err)

	childContract, err = eth.NewContract(childClient, C.Contracts.Path+"/GometChild.json")
	assert(err)

	wethContract, err = eth.NewContract(childClient, C.Contracts.Path+"/WETH.json")
	assert(err)

}

func serverDeploy() {

	var err error

	if len(C.Contracts.DeploySigners) == 0 {
		assert(fmt.Errorf("Initial signers list is empty"))
	}

	assert(C.VerifyDeploySigners())

	initialSigners := make([]common.Address, len(C.Contracts.DeploySigners))
	for i, signer := range C.Contracts.DeploySigners {
		initialSigners[i] = common.HexToAddress(signer)
	}

	// -- deploy contracts
	_, _, err = parentContract.Deploy(initialSigners)
	assert(err)
	log.Println("GometParent deployed at ", parentContract.Address.Hex())
	_, _, err = childContract.Deploy(initialSigners)

	assert(err)
	log.Println("GometChild deployed at ", childContract.Address.Hex())
	_, _, err = wethContract.Deploy(childContract.Address)

	assert(err)
	log.Println("WETH deployed at ", wethContract.Address.Hex())

	// -- set weth address
	_, _, err = childContract.SendTransactionSync(big.NewInt(0), "init", wethContract.Address)
	assert(err)
	log.Println("WETH attached to GometChild")
}

func serverStart() {

	assert(C.VerifyAddresses())

	parentContract.SetAddress(common.HexToAddress(C.MainChain.BridgeAddress))
	log.Println("GometParent address is ", parentContract.Address.Hex())

	childContract.SetAddress(common.HexToAddress(C.SideChain.BridgeAddress))
	log.Println("GometChild address is ", childContract.Address.Hex())

	// -- get weth address
	wethcallresult, err := childContract.Call(big.NewInt(0), "weth")
	assert(err)
	wethContract.SetAddress(common.BytesToAddress(wethcallresult[12:]))
	log.Println("WETH address is ", wethContract.Address.Hex())

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

	dotest()

	<-time.After(time.Second * 3600)

}
