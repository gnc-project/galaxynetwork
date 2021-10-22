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
	"github.com/gnc-project/galaxynetwork/consensus/ethash"
	"math/big"

	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/consensus"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/core/vm"
	"github.com/gnc-project/galaxynetwork/rewardc"
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
		beneficiary common.Address
		baseFee     *big.Int
	)

//	If we don't have an explicit author (i.e. not mining), extract from the header
	if author == nil {
		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	} else {
		beneficiary = *author
	}

	if header.BaseFee != nil {
		baseFee = new(big.Int).Set(header.BaseFee)
	}

	return vm.BlockContext{
		CanTransfer:          CanTransfer,
		Transfer:             Transfer,
		CanRedeem:            CanRedeem,
		PledgeTransfer:       PledgeTransfer,
		RedeemTransfer:       RedeemTransfer,
		DelectPidTransfer:    DeletePidTransfer,
		UnlockRewardTransfer: UnlockRewardTransfer,
		StakingTransfer:      StakingTransfer,
		GetHash:              GetHashFn(header, chain),
		Coinbase:             beneficiary,
		BlockNumber:          new(big.Int).Set(header.Number),
		//Time:                 new(big.Int).SetUint64(header.Time),
		//Difficulty:           new(big.Int).Set(header.Difficulty),
		BaseFee:              baseFee,
		GasLimit:             header.GasLimit,
		NetCapacity:          header.NetCapacity,
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

func CanRedeem(db vm.StateDB, addr common.Address,number *big.Int) bool {
	redeemAmount:=db.GetRedeemAmount(addr,number.Uint64())
	return redeemAmount.Cmp(big.NewInt(0)) > 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// PledgeTransfer Subtracts pledgeAmount from sender’s balance and adds pledge to recipient’s pledge
func PledgeTransfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.PledgeBinding(recipient,sender)
	db.SetPledgeAmount(recipient,amount)
	db.AddTotalPledgeAmount(sender,amount)
	db.AddTotalCapacity(sender,big.NewInt(rewardc.BaseCapacity))
}

// DeletePidTransfer 1
func DeletePidTransfer(db vm.StateDB, sender, recipient common.Address,number *big.Int) {

	redeemAmount := db.GetPledgeAmount(recipient,sender)
	if redeemAmount.Cmp(common.Big0) <= 0 {
		return
	}

	db.DeleteBinding(recipient)
	db.SetPledgeAmount(recipient,big.NewInt(0))
	db.SubTotalPledgeAmount(sender,redeemAmount)
	db.SubTotalCapacity(sender,big.NewInt(rewardc.BaseCapacity))

	if new(big.Int).Div(db.GetTotalCapacity(sender),big.NewInt(rewardc.BasePB)).Uint64() < rewardc.TotalCapacity{
		db.SetFunds(sender,common.MinedBlocks{})
	}

	trueRedeemAmount := new(big.Int).Div(new(big.Int).Mul(redeemAmount,big.NewInt(75)),big.NewInt(100))
	db.AddCanRedeem(sender,number.Uint64() + (rewardc.Day60 * rewardc.DayBlock),trueRedeemAmount)
}

// RedeemTransfer 2
func RedeemTransfer(db vm.StateDB, sender common.Address,number *big.Int){

	CanRedeemList := db.GetCanRedeem(sender)

	amount := big.NewInt(0)
	newRedeemList := common.CanRedeemList{}
	for _,canRedeem := range CanRedeemList{
		if canRedeem.UnlockBlock <= number.Uint64() {
			amount.Add(amount,canRedeem.RedeemAmount)
		}else {
			newRedeemList = append(newRedeemList, canRedeem)
		}
	}
	db.SubCanRedeem(sender,newRedeemList)
	db.AddBalance(sender,amount)
}

// UnlockRewardTransfer Linear release
func UnlockRewardTransfer(db vm.StateDB, sender common.Address, number *big.Int) {
	funds := db.GetFunds(sender)
	amountUnlocked,newFunds := ethash.CalculateAmountUnlocked(number,funds)
	db.AddBalance(sender, amountUnlocked)
	db.SetFunds(sender,newFunds)
}

func StakingTransfer(db vm.StateDB, sender common.Address, amount *big.Int,frozenPeriod *big.Int,number *big.Int){
	staking := &common.Staking{Account: sender, FrozenPeriod: frozenPeriod.Uint64(), Value: amount,
		StartNumber: number.Uint64(),StopNumber: number.Uint64() + frozenPeriod.Uint64() * rewardc.DayBlock}
	db.SubBalance(sender, amount)
	db.AddStakingList(common.AllStakingDB,staking)
}
