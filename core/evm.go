// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"
	"strconv"
	"encoding/hex"


	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rewardc"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"


)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hash.
	GetHeader(common.Hash, uint64) *types.Header
}

// NewEVMBlockContext creates a new context for use in the EVM.
func NewEVMBlockContext(header *types.Header, chain ChainContext, author *common.Address) vm.BlockContext {
	var (
		// beneficiary common.Address
		baseFee     *big.Int
	)

	// If we don't have an explicit author (i.e. not mining), extract from the header
	// if author == nil {
	// 	beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	// } else {
	// 	beneficiary = *author
	// }
	if header.BaseFee != nil {
		baseFee = new(big.Int).Set(header.BaseFee)
	}
	return vm.BlockContext{
		CanTransfer:          CanTransfer,
		CanRedeem:            CanRedeem,
		Transfer:             Transfer,
		PledgeTransfer:       PledgeTransfer,
		RedeemTransfer:       RedeemTransfer,
		DelectPidTransfer:    DelectPidTransfer,
		UnlockRewardTransfer: UnlockRewardTransfer,
		StakingTransfer:      StakingTransfer,
		UnlockStakingTransfer: UnlockStakingTransfer,
		GetHash:              GetHashFn(header, chain),
		//Coinbase:             beneficiary,
		BlockNumber:          new(big.Int).Set(header.Number),
		//Time:                 new(big.Int).SetUint64(header.Time),
		//Difficulty:           new(big.Int).Set(header.Difficulty),
		BaseFee:              baseFee,
		GasLimit:             header.GasLimit,
	}
}

// NewEVMTxContext creates a new transaction context for a single transaction.
func NewEVMTxContext(msg Message) vm.TxContext {
	return vm.TxContext{
		Origin:   msg.From(),
		GasPrice: new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn(ref *types.Header, chain ChainContext) func(n uint64) common.Hash {
	// Cache will initially contain [refHash.parent],
	// Then fill up with [refHash.p, refHash.pp, refHash.ppp, ...]
	var cache []common.Hash

	return func(n uint64) common.Hash {
		// If there's no hash cache yet, make one
		if len(cache) == 0 {
			cache = append(cache, ref.ParentHash)
		}
		if idx := ref.Number.Uint64() - n - 1; idx < uint64(len(cache)) {
			return cache[idx]
		}
		// No luck in the cache, but we can start iterating from the last element we already know
		lastKnownHash := cache[len(cache)-1]
		lastKnownNumber := ref.Number.Uint64() - uint64(len(cache))

		for {
			header := chain.GetHeader(lastKnownHash, lastKnownNumber)
			if header == nil {
				break
			}
			cache = append(cache, header.ParentHash)
			lastKnownHash = header.ParentHash
			lastKnownNumber = header.Number.Uint64() - 1
			if n == lastKnownNumber {
				return lastKnownHash
			}
		}
		return common.Hash{}
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

func CanRedeem(db vm.StateDB, addr common.Address, amount *big.Int,number *big.Int) bool {

	redeemAmount:=db.GetRedeemAmount(addr,number.Uint64())
	return redeemAmount.Cmp(amount) >= 0

}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

//Subtracts pledgeamount from sender’s balance and adds pledge to recipient’s pledge
func PledgeTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int, data []byte) {
	db.SubBalance(sender, amount)
	db.AddPledge(recipient, amount)
	pidData := hexutil.SlitData(data)
	for i := 0; i < len(pidData); i++ {
		amount:=new(big.Int).Div(amount,big.NewInt(int64(len(pidData))))
		db.AddPid(recipient, pidData[i],amount)
	}
}

func RedeemTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int,number *big.Int){
	CanRedeemList:=db.GetCanRedeem(sender)
	for index,canRedeem:=range CanRedeemList{
		if canRedeem.UnlockBlock<number.Uint64(){
			db.SubCanRedeem(sender,int64(index))
			db.AddBalance(recipient,canRedeem.RedeemAmount)
		}
	}
}

func DelectPidTransfer(db vm.StateDB, sender, recipient common.Address,data []byte,number *big.Int){
	pidData := hexutil.SlitData(data)
	redeemAmount:=big.NewInt(0)
	for _,pid:=range pidData{
		redeemAmount=new(big.Int).Add(redeemAmount,db.SubPid(sender,pid))
	}
	if redeemAmount.Cmp(big.NewInt(0))>0{
		if len(db.GetPid(sender))*102/1048576<=100{
			lockedFunds := db.GetTotalLockedFunds(sender)
			db.SubTotalLockedFunds(sender,lockedFunds)
			funds:=[]struct{BlockNumber *big.Int; Amount *big.Int}{}
			db.SetFunds(sender,funds)
		}
		db.SubPledge(sender,redeemAmount)
		trueRedeemAmout:=new(big.Int).Div(new(big.Int).Mul(redeemAmount,big.NewInt(75)),big.NewInt(100))
		db.AddCanRedeem(recipient,number.Uint64()+(60*rewardc.DayBlock),trueRedeemAmout)
	}
}

func UnlockRewardTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int, number *big.Int) {
	rewardToLock, available, lockedRewardVestingSpec := ethash.LockedRewardFromReward(big.NewInt(0))

	amountUnlocked := ethash.SetLockedFunds(rewardToLock, lockedRewardVestingSpec, db.(*state.StateDB), recipient, number)

	db.AddBalance(recipient, new(big.Int).Add(available, amountUnlocked))
}

func StakingTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int,data []byte,number *big.Int){


	frozenPeriod,_:=strconv.ParseUint(hex.EncodeToString(data[7:]), 16, 64)

	
	stakingInfo:=struct{Value *big.Int; Weight *big.Int; StopBlock uint64; StartBlock uint64}{
		Value:amount,
		StopBlock:number.Uint64()+uint64((frozenPeriod*rewardc.DayBlock)),
		StartBlock: number.Uint64(),
	}

	switch{
	case frozenPeriod<90:
		frozenPeriod=0
	case  90<=frozenPeriod&&frozenPeriod<180:
		frozenPeriod=90
	case  180<=frozenPeriod&&frozenPeriod<360:
		frozenPeriod=180
	case  360<=frozenPeriod&&frozenPeriod<1080:
		frozenPeriod=360
	case  1080<=frozenPeriod:
		frozenPeriod=1080
	}
	
	stakingInfo.Weight=new(big.Int).Mul(new(big.Int).SetUint64(rewardc.StakingBase[frozenPeriod]),stakingInfo.Value)
	staking:=&common.Staking{
		Address:&sender,
		StakingInfo:append([]struct{Value *big.Int; Weight *big.Int; StopBlock uint64; StartBlock uint64}{},stakingInfo),
		TotalValue:stakingInfo.Value,
		TotalWeight:stakingInfo.Weight,
	  }
	db.SubBalance(sender, amount)
	db.AddStakingList(sender,staking)
}



func UnlockStakingTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int,number *big.Int) {
    db.SubStakingList(sender,number.Uint64())
	db.AddBalance(recipient, amount)
}
