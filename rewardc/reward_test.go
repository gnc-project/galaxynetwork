package rewardc

import (
	"fmt"
	"math/big"
	"testing"
)

func TestGetReward(t *testing.T) {
	for i:=0; i<10000000000;i++ {
		reward := GetReward(uint64(i))
		for n :=uint64(2) ;n < 80; n=n+2{
			if uint64(i) == BlockTotal * n {
				fmt.Println(reward.Div(reward,big.NewInt(1e18)),i,BlockTotal * n)
			}
		}
	}
}