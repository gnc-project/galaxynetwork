package pidaddress

import (
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/crypto"
)

func PIDAddress(ads common.Address,pid []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(append(ads[:], pid...)))
}
