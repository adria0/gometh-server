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
	mainClient   *eth.Web3Client
	sideClient   *eth.Web3Client
	mainContract *eth.Contract
	sideContract *eth.Contract
	wethContract *eth.Contract
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

	mainClient, err = eth.NewWeb3Client(
		cfg.C.MainChain.RPCURL,
		ks,
		account,
	)
	assert(err)

	sideClient, err = eth.NewWeb3Client(
		cfg.C.SideChain.RPCURL,
		ks,
		account,
	)
	assert(err)

	mainClient.ClientMutex = &sync.Mutex{}
	sideClient.ClientMutex = mainClient.ClientMutex

	parentAccountInfo, err := mainClient.AccountInfo()
	assert(err)
	log.Println("Parent chain account: ", parentAccountInfo)

	childAccountInfo, err := sideClient.AccountInfo()
	assert(err)
	log.Println("Child chain account", childAccountInfo)

	// -- load contracts
	mainContract, err = eth.NewContract(mainClient, cfg.C.Contracts.Path+"/GomethMain.json")
	assert(err)

	sideContract, err = eth.NewContract(sideClient, cfg.C.Contracts.Path+"/GomethSide.json")
	assert(err)

	wethContract, err = eth.NewContract(sideClient, cfg.C.Contracts.Path+"/WETH.json")
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
	_, _, err = mainContract.Deploy(initialSigners)
	assert(err)
	log.Println("GomethMain deployed at ", mainContract.Address.Hex())
	_, _, err = sideContract.Deploy(initialSigners)

	assert(err)
	log.Println("GometSide deployed at ", sideContract.Address.Hex())
	_, _, err = wethContract.Deploy(sideContract.Address)

	assert(err)
	log.Println("WETH deployed at ", wethContract.Address.Hex())

	// -- set weth address
	_, _, err = sideContract.SendTransactionSync(big.NewInt(0), 0, "init", wethContract.Address)
	assert(err)
	log.Println("WETH attached to GometSide")
}

func setContractsAddress() {

	assert(cfg.C.VerifyAddresses())

	mainContract.SetAddress(common.HexToAddress(cfg.C.MainChain.BridgeAddress))
	log.Println("GomethMain address is ", mainContract.Address.Hex())

	sideContract.SetAddress(common.HexToAddress(cfg.C.SideChain.BridgeAddress))
	log.Println("GometSide address is ", sideContract.Address.Hex())

	// -- get weth address
	var wethAddress common.Address
	assert(sideContract.Call(&wethAddress, "weth"))
	wethContract.SetAddress(wethAddress)
	log.Println("WETH address is ", wethContract.Address.Hex())

}
