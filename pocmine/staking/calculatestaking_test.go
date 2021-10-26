package staking

import (
	"fmt"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/rewardc"
	"github.com/shopspring/decimal"
	"math/big"
	"testing"
)

func TestCalculateStaking(t *testing.T) {

	staking1 := common.Staking{}
	staking1.StartNumber = 1
	staking1.Account = common.HexToAddress("0x461618Dc4480246eBAabb48169BC535e03e9f86E")
	staking1.FrozenPeriod = 90
	staking1.StopNumber = 90
	staking1.Value =  new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e18))
	staking1.Index = big.NewInt(1)


	staking2 := common.Staking{}
	staking2.StartNumber = 1
	staking2.Account = common.HexToAddress("0x6FA7F9Dc05302B7F792A6975A18458574311F432")
	staking2.FrozenPeriod = 180
	staking2.StopNumber = 180
	staking2.Value = new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e18))
	staking2.Index = big.NewInt(2)

	staking3 := common.Staking{}
	staking3.StartNumber = 1
	staking3.Account = common.HexToAddress("0x55aB559Aff7B42DA26e80c271EfdA798BD799953")
	staking3.FrozenPeriod = 360
	staking3.StopNumber = 360
	staking3.Value = new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e18))
	staking3.Index = big.NewInt(3)


	staking4 := common.Staking{}
	staking4.StartNumber = 1
	staking4.Account = common.HexToAddress("0x45590359f91bf968eE3dF3116676A15507C37241")
	staking4.FrozenPeriod = 1080
	staking4.StopNumber = 1080
	staking4.Value = new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e18))
	staking4.Index = big.NewInt(4)


	staking41 := common.Staking{}
	staking41.StartNumber = 1
	staking41.Account = common.HexToAddress("0x45590359f91bf968eE3dF3116676A15507C37241")
	staking41.FrozenPeriod = 1080
	staking41.StopNumber = 1080
	staking41.Value = new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e18))
	staking41.Index = big.NewInt(5)

	list := common.StakingList{}

	list = append(list, &staking2)
	list = append(list, &staking4)
	list = append(list, &staking1)
	list = append(list, &staking3)

	//list = append(list, &staking41)

	re := new(big.Int).Mul(big.NewInt(600),big.NewInt(1e18))
	stakingReward := new(big.Int).Mul(new(big.Int).Div(re,big.NewInt(100)),rewardc.StakingRewardProportion)
	reward := decimal.NewFromBigInt(stakingReward,0)
	accFree, newStakingList, rewardStaking := CalculateStaking(list,50,reward)

	for _,v := range rewardStaking {
		fmt.Printf("address=%s value=%v weight=%v accReward=%s\n",v.Account.Hex(),v.Value,v.Weight,v.Reward.String())
	}
	fmt.Println("----------------------------")
	for _,v := range newStakingList {
		fmt.Println("--newStakingList---sort-->",*v)
	}

	for _,v := range list {
		fmt.Println("--list---sort-->",*v)
	}

	fmt.Println("----------------------------")
	fmt.Printf("free=%v newStakingListLen=%d rewardStakinglen=%d \n",accFree, len(newStakingList), len(rewardStaking))

}
