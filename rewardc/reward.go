package rewardc

import "math/big"
const (
	POCReward              = uint64(600)
	FutureBlockTime        = uint64(18)
	BlockTotal             = 365 * 24 * 60 * 60 / FutureBlockTime
	subsidyHalvingInterval = BlockTotal * 2

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
	TotalCapacity			= 100 	//PB
	BasePB					= 1024 * 1024
)

var (
	PledgeBase =map[uint64]*big.Int{
		100:new(big.Int).Mul(big.NewInt(56250),big.NewInt(1e+16)),
		200:new(big.Int).Mul(big.NewInt(49219),big.NewInt(1e+16)),
		300:new(big.Int).Mul(big.NewInt(42188),big.NewInt(1e+16)),
		400:new(big.Int).Mul(big.NewInt(31641),big.NewInt(1e+16)),
		500:new(big.Int).Mul(big.NewInt(25313),big.NewInt(1e+16)),
		600:new(big.Int).Mul(big.NewInt(21094),big.NewInt(1e+16)),
		700:new(big.Int).Mul(big.NewInt(18080),big.NewInt(1e+16)),
		800:new(big.Int).Mul(big.NewInt(15820),big.NewInt(1e+16)),
		900:new(big.Int).Mul(big.NewInt(15625),big.NewInt(1e+16)),
		1000:new(big.Int).Mul(big.NewInt(14625),big.NewInt(1e+16)),
		1100:new(big.Int).Mul(big.NewInt(14318),big.NewInt(1e+16)),
		1200:new(big.Int).Mul(big.NewInt(14063),big.NewInt(1e+16)),
		1300:new(big.Int).Mul(big.NewInt(12981),big.NewInt(1e+16)),
		1400:new(big.Int).Mul(big.NewInt(12054),big.NewInt(1e+16)),
		1500:new(big.Int).Mul(big.NewInt(11250),big.NewInt(1e+16)),
		1600:new(big.Int).Mul(big.NewInt(10547),big.NewInt(1e+16)),
		1700:new(big.Int).Mul(big.NewInt(9926),big.NewInt(1e+16)),
		1800:new(big.Int).Mul(big.NewInt(9375),big.NewInt(1e+16)),
		1900:new(big.Int).Mul(big.NewInt(8882),big.NewInt(1e+16)),
		2000:new(big.Int).Mul(big.NewInt(8438),big.NewInt(1e+16)),
		3000:new(big.Int).Mul(big.NewInt(7969),big.NewInt(1e+16)),
		4000:new(big.Int).Mul(big.NewInt(6328),big.NewInt(1e+16)),
		5000:new(big.Int).Mul(big.NewInt(5063),big.NewInt(1e+16)),
		6000:new(big.Int).Mul(big.NewInt(4219),big.NewInt(1e+16)),
		7000:new(big.Int).Mul(big.NewInt(3616),big.NewInt(1e+16)),
		8000:new(big.Int).Mul(big.NewInt(3164),big.NewInt(1e+16)),
		9000:new(big.Int).Mul(big.NewInt(2813),big.NewInt(1e+16)),
		10000:new(big.Int).Mul(big.NewInt(2531),big.NewInt(1e+16)),
		20000:new(big.Int).Mul(big.NewInt(1266),big.NewInt(1e+16)),
		30000:new(big.Int).Mul(big.NewInt(844),big.NewInt(1e+16)),
		// 40000:new(big.Int).Mul(big.NewInt(633),big.NewInt(1e+16)),
		// 50000:new(big.Int).Mul(big.NewInt(506),big.NewInt(1e+16)),
		// 60000:new(big.Int).Mul(big.NewInt(422),big.NewInt(1e+16)),
		// 70000:new(big.Int).Mul(big.NewInt(362),big.NewInt(1e+16)),
		// 80000:new(big.Int).Mul(big.NewInt(316),big.NewInt(1e+16)),
		// 90000:new(big.Int).Mul(big.NewInt(281),big.NewInt(1e+16)),
		// 100000:new(big.Int).Mul(big.NewInt(253),big.NewInt(1e+16)),
	}
	StakingBase=map[uint64]uint64{
        0:1,
		90:10,
		180:20,
		360:30,
		1080:50,
	}

	
	StakingRewardProportion=big.NewInt(20)
	MineRewardProportion=big.NewInt(80)

	StakingLowerLimit=new(big.Int).Mul(big.NewInt(1000),big.NewInt(1e+18))
)

// bigOne is 1 represented as a big.Int.  It is defined here to avoid
// the overhead of creating it multiple times.
var BigOne = big.NewInt(1)

// mainPocLimit is the smallest proof of capacity target.
var MainPocLimit = new(big.Int).Sub(new(big.Int).Lsh(BigOne, 20), BigOne)

var Power  = big.NewInt(0).Exp(big.NewInt(2),big.NewInt(64),nil)

var BlockReward = big.NewInt(0).Mul(big.NewInt(0).SetUint64(POCReward), big.NewInt(1e+18))



func GetReward(height uint64) *big.Int {
	halvings := height / subsidyHalvingInterval
	subsidy := POCReward
	subsidy >>= halvings

	return new(big.Int).Mul(big.NewInt(int64(subsidy)), big.NewInt(1e+18))
}
