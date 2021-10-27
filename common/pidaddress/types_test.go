package pidaddress

import (
	"fmt"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/hexutil"
	"testing"
)

func TestPIDAddress(t *testing.T) {
	ads := PIDAddress(common.HexToAddress("0xccFD71131015d9dcDDf3f2e97B0aA7bE11E1bF5F"),hexutil.MustDecode("0x74865a286121cc902608ab5646a50d8e77eceacfc45ea101d418cbd48574f7fb"))
	fmt.Println(ads.Hex())
}
