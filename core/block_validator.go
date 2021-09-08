// Copyright 2015 The go-ethereum Authors
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
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/gnc-project/galaxynetwork/common/hexutil"
	"github.com/gnc-project/galaxynetwork/consensus"
	"github.com/gnc-project/galaxynetwork/core/state"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/params"
	"github.com/gnc-project/galaxynetwork/rewardc"
	"github.com/gnc-project/galaxynetwork/trie"
)

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	if hash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil)); hash != header.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, header.TxHash)
	}
	if !v.bc.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		if !v.bc.HasBlock(block.ParentHash(), block.NumberU64()-1) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	transactions := block.Transactions()
	if len(transactions) > 0 {
		for i := 0; i < len(transactions); i++ {
			msg, err := transactions[i].AsMessage(types.MakeSigner(v.config, header.Number), header.BaseFee)
			if err != nil {
				return fmt.Errorf("getFromErr :%v", err)
			}
			var snapdata []byte
			if msg.Data() == nil {
				snapdata = []byte{}
			} else {
				snapdata = msg.Data()
			}
			if len(snapdata) > 6 && strings.EqualFold(hex.EncodeToString(snapdata[:6]), hex.EncodeToString([]byte("pledge"))) {

				pidData := hexutil.SlitData(snapdata)
				
				currentNetCapacity:=v.bc.GetBlockByHash(block.ParentHash()).NetCapacity()/1048576
				switch{
				case currentNetCapacity<100:
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

				pledgeValue := new(big.Int).Mul(new(big.Int).SetInt64(int64(len(pidData))), new(big.Int).Div(rewardc.PledgeBase[currentNetCapacity*100],big.NewInt(10)))
				if msg.Value().Cmp(pledgeValue) < 0 {
					return fmt.Errorf("invalid pledge tx value (remote: %v local: %v)", msg.Value(), pledgeValue)
				}
			}
			if len(snapdata) > 7&&strings.EqualFold(hex.EncodeToString(snapdata[:7]), hex.EncodeToString([]byte("staking"))){
				if msg.Value().Cmp(rewardc.StakingLowerLimit)<0{
					return ErrInsufficientStakingValue
				}
			}
		}
	}
	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	header := block.Header()
	//poc
	if header.Number.Uint64() >= rewardc.PledgeNumber {
		if !statedb.VerifyPid(header.Coinbase, header.Pid[:]) {
			return fmt.Errorf("invalid pid=%v is not pledged address=%v", hex.EncodeToString(header.Pid[:]), header.Coinbase.Hex())
		}
	}

	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, Rn]]))
	receiptSha := types.DeriveSha(receipts, trie.NewStackTrie(nil))
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}

// CalcGasLimit computes the gas limit of the next block after parent. It aims
// to keep the baseline gas close to the provided target, and increase it towards
// the target if the baseline gas is lower.
func CalcGasLimit(parentGasLimit, desiredLimit uint64) uint64 {
	delta := parentGasLimit/params.GasLimitBoundDivisor - 1
	limit := parentGasLimit
	if desiredLimit < params.MinGasLimit {
		desiredLimit = params.MinGasLimit
	}
	// If we're outside our allowed gas range, we try to hone towards them
	if limit < desiredLimit {
		limit = parentGasLimit + delta
		if limit > desiredLimit {
			limit = desiredLimit
		}
		return limit
	}
	if limit > desiredLimit {
		limit = parentGasLimit - delta
		if limit < desiredLimit {
			limit = desiredLimit
		}
	}
	return limit
}
