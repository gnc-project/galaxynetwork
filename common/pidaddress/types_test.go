package pidaddress

import (
	"fmt"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/hexutil"
	"testing"
)

func TestPIDAddress(t *testing.T) {
	ads := PIDAddress(common.HexToAddress("0x34669E11808f1879929203092CA3093319221068"),hexutil.MustDecode("0x4317d8d4acf66277e98590a11901ec6ba6851b641e5754d816667aab9ba670e6"))
	fmt.Println(ads.Hex())
}
