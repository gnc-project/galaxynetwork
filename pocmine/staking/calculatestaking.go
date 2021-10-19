package staking

import (
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/rewardc"
	"github.com/shopspring/decimal"
	"math/big"
	"sort"
)

func CalculateStaking(list common.StakingList,number uint64,reward decimal.Decimal) (map[string]*big.Int,common.StakingList,common.StakingWeightList) {
	newStakingList := common.StakingList{}
	stakingMap := make(map[string]*common.StakingWeight)
	accFree := make(map[string]*big.Int)
	for _,v := range list{
		if v.StopNumber > number {

			newStakingList = append(newStakingList, v)

			if sw,ok := stakingMap[v.Account.Hex()]; ok{
				sw.Weight = sw.Weight.Add(rewardc.CalculateWeight(v.FrozenPeriod,v.Value))
				sw.Value = sw.Value.Add(sw.Value,v.Value)
				stakingMap[v.Account.Hex()] = sw
			}else {
				stakingWeight := &common.StakingWeight{Account: v.Account, Weight: rewardc.CalculateWeight(v.FrozenPeriod,v.Value),Value: v.Value}
				stakingMap[v.Account.Hex()] = stakingWeight
			}

		}else {
			if freeAmount,ok := accFree[v.Account.Hex()]; ok {
				accFree[v.Account.Hex()] = freeAmount.Add(freeAmount,v.Value)
			}else {
				accFree[v.Account.Hex()] = v.Value
			}
		}
	}
	sort.SliceStable(newStakingList, func(i, j int) bool {
		if newStakingList[i].StartNumber < newStakingList[j].StartNumber {
			return true
		}
		if newStakingList[i].Account.Hash().Big().Cmp(newStakingList[j].Account.Hash().Big()) > 0 {
			return true
		}
		return false
	})


	totalWeight := decimal.Zero
	rewardStaking := common.StakingWeightList{}
	for _,v := range stakingMap {
		rewardStaking = append(rewardStaking,v)
		totalWeight = totalWeight.Add(v.Weight)
	}

	sort.SliceStable(rewardStaking, func(first, second int) bool {
		if rewardStaking[first].Weight.Cmp(rewardStaking[second].Weight) > 0 {
			return true
		}
		if rewardStaking[first].Weight.Cmp(rewardStaking[second].Weight) == 0 &&
			rewardStaking[first].Account.Hash().Big().Cmp(rewardStaking[second].Account.Hash().Big()) > 0 {
			return true
		}
		return false
	})

	if len(rewardStaking) > rewardc.StakingNum {
		rewardStaking = rewardStaking[:rewardc.StakingNum]
	}

	for k,v := range rewardStaking {
		rate := v.Weight.Div(totalWeight)
		accReward := reward.Mul(rate)
		rewardStaking[k].Reward = accReward.BigInt()
	}

	return accFree, newStakingList, rewardStaking
}