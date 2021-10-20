package transfertype

import (
	"encoding/hex"
	"errors"
	"github.com/gnc-project/galaxynetwork/rewardc"
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

	switch{
	case currentNetCapacity < 100:
		currentNetCapacity=1
	case 100<=currentNetCapacity&&currentNetCapacity<2000:
		currentNetCapacity=currentNetCapacity/100
	case 2000<=currentNetCapacity&&currentNetCapacity<10000:
		currentNetCapacity=currentNetCapacity/1000*10
	case 10000<=currentNetCapacity&&currentNetCapacity<30000:
		currentNetCapacity=currentNetCapacity/10000*100
	default :
		currentNetCapacity=300
	}

	return new(big.Int).Div(rewardc.PledgeBase[currentNetCapacity*100],big.NewInt(10))
}