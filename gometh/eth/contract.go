package eth

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"

	cfg "github.com/adriamb/gometh-server/gometh/config"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Contract is a smartcontract with optional address
type Contract struct {
	Abi      abi.ABI
	Client   *Web3Client
	ByteCode []byte
	Address  *common.Address
}

var (
	ErrAddressHasNoCode = fmt.Errorf("Account has no code")
)

// NewContract initiates a contract ABI & bytecode from json file associated to a web3 client
func NewContract(client *Web3Client, jsonFile string) (*Contract, error) {

	var contract Contract

	content, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, err
	}

	abivalue := fields["abi"]
	bytecodehex := fields["bytecode"].(string)
	if contract.ByteCode, err = hex.DecodeString(bytecodehex[2:]); err != nil {
		return nil, err
	}

	abijson, err := json.Marshal(&abivalue)
	if err != nil {
		return nil, err
	}

	contract.Abi, err = abi.JSON(bytes.NewReader(abijson))
	if err != nil {
		return nil, err
	}

	contract.Client = client

	return &contract, nil
}

// SetAddress sets the contract's address
func (b *Contract) SetAddress(address common.Address) error {
	code, err := b.Client.Client.CodeAt(context.TODO(), address, nil)
	if err != nil {
		return err
	}
	if code == nil || len(code) == 0 {
		log.Println("Account ", address.Hex(), " has no code")
		return ErrAddressHasNoCode
	}
	b.Address = &address
	return nil
}

// SendTransactionSync executes a contract method and wait it finalizes
func (b *Contract) SendTransactionSync(value *big.Int, gasLimit uint64, funcname string, params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	msg, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		if cfg.Verbose > 0 {
			log.Println("Failed packing ", funcname)
			return nil, nil, err
		}
	}
	tx, receipt, err := b.Client.SendTransactionSync(b.Address, value, gasLimit, msg)
	if err != nil && cfg.Verbose > 0 {
		log.Println("Failed calling ", funcname)
	}

	return tx, receipt, err
}

// Deploy the contract
func (b *Contract) Deploy(params ...interface{}) (*types.Transaction, *types.Receipt, error) {

	init, err := b.Abi.Pack("", params...)
	if err != nil {
		return nil, nil, err
	}

	code := append([]byte(nil), b.ByteCode...)
	code = append(code, init...)

	tx, receipt, err := b.Client.SendTransactionSync(nil, big.NewInt(0), 0, code)

	if err == nil {
		b.Address = &receipt.ContractAddress
	}

	return tx, receipt, err
}

// Call an constant method
func (b *Contract) Call(funcname string, params ...interface{}) ([]byte, error) {

	msgdata, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		return nil, err
	}
	return b.Client.Call(b.Address, big.NewInt(0), msgdata)
}

func sign(client *Web3Client, data ...[]byte) ([3][32]byte, error) {
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

func (b *Contract) PartialExecuteOff(eventlog *types.Log, value *big.Int, gasLimit uint64, funcname string, params ...interface{}) error {

	epoch := big.NewInt(0)

	txhash := crypto.Keccak256(eventlog.TxHash.Bytes(), eventlog.Topics[0].Bytes())
	var txid [32]byte
	copy(txid[:], txhash)

	msg, err := b.Abi.Pack(funcname, params...)
	if err != nil {
		return err
	}

	sig, err := sign(b.Client, abi.U256(epoch), txid[:], msg)
	if err != nil {
		return err
	}
	_, _, err = b.SendTransactionSync(
		big.NewInt(0), gasLimit,
		"partialExecuteOff", epoch, txid, msg, sig,
	)

	return err
}
