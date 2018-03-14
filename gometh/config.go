package gometh

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

// Config is the server configurtion

type Config struct {
	Keystore struct {
		Path   string
		Passwd string
	}

	Contracts struct {
		Path          string
		DeploySigners []string
	}

	MainChain struct {
		RPCURL        string
		BridgeAddress string
	}

	SideChain struct {
		RPCURL        string
		BridgeAddress string
	}
}

func (c *Config) VerifyDeploySigners() error {

	for _, signer := range C.Contracts.DeploySigners {
		if !common.IsHexAddress(signer) {
			return fmt.Errorf("Bad initial deploy address %v", signer)
		}
	}

	return nil
}

func (c *Config) VerifyAddresses() error {

	if !common.IsHexAddress(c.MainChain.BridgeAddress) {
		return fmt.Errorf("Bad ParentAddress %v", c.MainChain.BridgeAddress)
	}

	if !common.IsHexAddress(c.SideChain.BridgeAddress) {
		return fmt.Errorf("Bad ChildAddress %v", c.SideChain.BridgeAddress)
	}

	return nil

}
