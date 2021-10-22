package transfertype

import (
	"encoding/hex"
	"errors"
	"math/big"
)

var (
	Pledge = hex.EncodeToString([]byte("pledge"))
	Staking = hex.EncodeToString([]byte("staking"))


	Redeem = hex.EncodeToString([]byte("redeem"))
	DelPid = hex.EncodeToString([]byte("delPid"))
	UnlockReward = hex.EncodeToString([]byte("unlockReward"))
)

var (
	ErrDuplicatePledgedPid = errors.New("duplicate pledged pid")
	ErrInsufficientPledge = errors.New("insufficient funds for Pledge")
	ErrInsufficientRedeem1 = errors.New("insufficient funds for Redeem amount")
	ErrInsufficientRedeem2 = errors.New("unlockBlock not now")
	ErrNotPledged  =  errors.New("not pledged")
	ErrInsufficientUnlockRewardValue=errors.New("insufficient funds for UnlockReward")
	ErrInsufficientStakingValue = errors.New("the staking amount is too small,Minimum 1000")
	ErrInsufficientFundsForRedeem = errors.New("insufficient for redeem")
	ErrInvalidPeriods = errors.New("invalid periods")
	ErrInvalidDelPid = errors.New("invalid del pid")
	ErrInvalidPledgedValue = errors.New("invalid pledged value")
)

//CalculatePledgeAmount	file amount
func CalculatePledgeAmount(currentNetCapacity uint64) *big.Int {
	currentNetCapacity = currentNetCapacity / 1024 / 1024
	switch  {
	case currentNetCapacity < 1:
		return new(big.Int).Mul(big.NewInt(56250),big.NewInt(1e+15))
	case currentNetCapacity < 2:
		return new(big.Int).Mul(big.NewInt(49219),big.NewInt(1e+15))
	case currentNetCapacity < 3:
		return new(big.Int).Mul(big.NewInt(42188),big.NewInt(1e+15))
	case currentNetCapacity < 4:
		return new(big.Int).Mul(big.NewInt(31641),big.NewInt(1e+15))
	case currentNetCapacity < 500:
		return new(big.Int).Mul(big.NewInt(25313),big.NewInt(1e+15))
	case currentNetCapacity < 600:
		return new(big.Int).Mul(big.NewInt(21094),big.NewInt(1e+15))
	case currentNetCapacity < 700:
		return new(big.Int).Mul(big.NewInt(18080),big.NewInt(1e+15))
	case currentNetCapacity < 800:
		return new(big.Int).Mul(big.NewInt(15820),big.NewInt(1e+15))
	case currentNetCapacity < 900:
		return new(big.Int).Mul(big.NewInt(15625),big.NewInt(1e+15))
	case currentNetCapacity < 1000:
		return new(big.Int).Mul(big.NewInt(14625),big.NewInt(1e+15))
	case currentNetCapacity < 1100:
		return new(big.Int).Mul(big.NewInt(14318),big.NewInt(1e+15))
	case currentNetCapacity < 1200:
		return new(big.Int).Mul(big.NewInt(14063),big.NewInt(1e+15))
	case currentNetCapacity < 1300:
		return new(big.Int).Mul(big.NewInt(12981),big.NewInt(1e+15))
	case currentNetCapacity < 1400:
		return new(big.Int).Mul(big.NewInt(12054),big.NewInt(1e+15))
	case currentNetCapacity < 1500:
		return new(big.Int).Mul(big.NewInt(11250),big.NewInt(1e+15))
	case currentNetCapacity < 1600:
		return new(big.Int).Mul(big.NewInt(10547),big.NewInt(1e+15))
	case currentNetCapacity < 1700:
		return new(big.Int).Mul(big.NewInt(9926),big.NewInt(1e+15))
	case currentNetCapacity < 1800:
		return new(big.Int).Mul(big.NewInt(9375),big.NewInt(1e+15))
	case currentNetCapacity < 1900:
		return new(big.Int).Mul(big.NewInt(8882),big.NewInt(1e+15))
	case currentNetCapacity < 2000:
		return new(big.Int).Mul(big.NewInt(8438),big.NewInt(1e+15))
	case currentNetCapacity < 3000:
		return new(big.Int).Mul(big.NewInt(7969),big.NewInt(1e+15))
	case currentNetCapacity < 4000:
		return new(big.Int).Mul(big.NewInt(6328),big.NewInt(1e+15))
	case currentNetCapacity < 5000:
		return new(big.Int).Mul(big.NewInt(5063),big.NewInt(1e+15))
	case currentNetCapacity < 6000:
		return new(big.Int).Mul(big.NewInt(4219),big.NewInt(1e+15))
	case currentNetCapacity < 7000:
		return new(big.Int).Mul(big.NewInt(3616),big.NewInt(1e+15))
	case currentNetCapacity < 8000:
		return new(big.Int).Mul(big.NewInt(3164),big.NewInt(1e+15))
	case currentNetCapacity < 9000:
		return new(big.Int).Mul(big.NewInt(2813),big.NewInt(1e+15))
	case currentNetCapacity < 10000:
		return new(big.Int).Mul(big.NewInt(2531),big.NewInt(1e+15))
	case currentNetCapacity < 20000:
		return new(big.Int).Mul(big.NewInt(1266),big.NewInt(1e+15))
	case currentNetCapacity < 30000:
		return new(big.Int).Mul(big.NewInt(844),big.NewInt(1e+15))
	default:
		return new(big.Int).Mul(big.NewInt(844),big.NewInt(1e+15))
	}
}