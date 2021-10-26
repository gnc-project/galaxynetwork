package rewardc

import (
	"fmt"
	"math/big"
	"testing"
)

func TestGetReward(t *testing.T) {
	for i:=0; i<10000000000;i++ {
		reward := GetReward(uint64(i))
		if new(big.Int).Div(reward,big.NewInt(1e18)).Cmp(big.NewInt(300)) <= 0 {
			fmt.Println("number",i,new(big.Int).Div(reward,big.NewInt(1e18)))
			return
		}
	}
}