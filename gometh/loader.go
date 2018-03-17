package gometh

import (
	"fmt"
	"log"
	"math/big"
	"sync"

	cfg "github.com/adriamb/gometh-server/gometh/config"
	eth "github.com/adriamb/gometh-server/gometh/eth"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
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

func initClient() {
	// -- open keystore

	var err error
	var account accounts.Account

	ks := keystore.NewKeyStore(cfg.C.Keystore.Path, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(ks.Accounts()) != 1 {
		panic(fmt.Sprintf("Not exact one account in keystore, was %v", len(ks.Accounts())))
	}
	account = ks.Accounts()[0]
	assert(ks.Unlock(account, cfg.C.Keystore.Passwd))

	// -- create clients

	parentClient, err = eth.NewWeb3Client(
		cfg.C.MainChain.RPCURL,
		ks,
		account,
	)
	assert(err)

	childClient, err = eth.NewWeb3Client(
		cfg.C.SideChain.RPCURL,
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
	parentContract, err = eth.NewContract(parentClient, cfg.C.Contracts.Path+"/GometParent.json")
	assert(err)

	childContract, err = eth.NewContract(childClient, cfg.C.Contracts.Path+"/GometChild.json")
	assert(err)

	wethContract, err = eth.NewContract(childClient, cfg.C.Contracts.Path+"/WETH.json")
	assert(err)

}

func deployContracts() {

	var err error

	if len(cfg.C.Contracts.DeploySigners) == 0 {
		assert(fmt.Errorf("Initial signers list is empty"))
	}

	assert(cfg.C.VerifyDeploySigners())

	initialSigners := make([]common.Address, len(cfg.C.Contracts.DeploySigners))
	for i, signer := range cfg.C.Contracts.DeploySigners {
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
	_, _, err = childContract.SendTransactionSync(big.NewInt(0), 0, "init", wethContract.Address)
	assert(err)
	log.Println("WETH attached to GometChild")
}

func setContractsAddress() {

	assert(cfg.C.VerifyAddresses())

	parentContract.SetAddress(common.HexToAddress(cfg.C.MainChain.BridgeAddress))
	log.Println("GometParent address is ", parentContract.Address.Hex())

	childContract.SetAddress(common.HexToAddress(cfg.C.SideChain.BridgeAddress))
	log.Println("GometChild address is ", childContract.Address.Hex())

	// -- get weth address
	wethcallresult, err := childContract.Call("weth")
	assert(err)
	wethContract.SetAddress(common.BytesToAddress(wethcallresult[12:]))
	log.Println("WETH address is ", wethContract.Address.Hex())

}
