package rewardc

import (
	"encoding/hex"
	"github.com/shopspring/decimal"
	"math/big"
)
const (
	POCReward              = uint64(600)
	FutureBlockTime        = uint64(18)
	BlockTotal             = 360 * 24 * 60 * 60 / FutureBlockTime
	subsidyHalvingInterval = BlockTotal * 2
	Blocks				   = subsidyHalvingInterval - 5000

	GenesisDifficulty      = uint64(15000000000000000) // Difficulty of the Genesis block.
	MinimumDifficulty      = uint64(15000000000000000) // The minimum that the difficulty may ever be.
	GenesisNumber          = 0
	PledgeNumber           = GenesisNumber + 100
    DayBlock               =  10 // 24 * 60 * 60 / FutureBlockTime
    Day60				   = 1	//Received in 60 days
	GenesisTimestamp       = 1630043179
	MinSectorExpiration    = 180
	ChainID					= 37021
	BaseCapacity			= 102	//GB
	TotalCapacity			= 2 	// default 100PB
	BasePB					= 1024 * 1024

	StakingNum				= 50
)

var (

	StakingBase=map[uint64]float64{
		90:0.1,
		180:0.2,
		360:0.3,
		1080:0.5,
	}

	StakingRewardProportion=big.NewInt(20)
	MineRewardProportion=big.NewInt(80)

	StakingLowerLimit=new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e+18))
)

func ParsingStakingBase(perHex string) (*big.Int, bool) {
	perStr, err := hex.DecodeString(perHex)
	if err != nil {
		return nil, false
	}
	per,ok := new(big.Int).SetString(string(perStr),10)
	if !ok {
		return nil, false
	}
	if _, yes := StakingBase[per.Uint64()]; !yes {
		return nil, false
	}
	return per, true
}

func CalculateWeight(frozenPeriod uint64,amount *big.Int) decimal.Decimal {
	am := decimal.NewFromBigInt(amount,0)
	rate := decimal.NewFromFloat(StakingBase[frozenPeriod])
	return am.Mul(rate)
}

// BigOne bigOne is 1 represented as a big.Int.  It is defined here to avoid
// the overhead of creating it multiple times.
var BigOne = big.NewInt(1)

// MainPocLimit mainPocLimit is the smallest proof of capacity target.
var MainPocLimit = new(big.Int).Sub(new(big.Int).Lsh(BigOne, 20), BigOne)

var Power  = big.NewInt(0).Exp(big.NewInt(2),big.NewInt(64),nil)

var BlockReward = big.NewInt(0).Mul(big.NewInt(0).SetUint64(POCReward), big.NewInt(1e+18))



func GetReward(height uint64) *big.Int {
	height = height + Blocks
	halvings := height / subsidyHalvingInterval
	subsidy := POCReward
	subsidy >>= halvings

	return new(big.Int).Mul(big.NewInt(int64(subsidy)), big.NewInt(1e+18))
}
