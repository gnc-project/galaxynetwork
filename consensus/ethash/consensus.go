// Copyright 2017 The go-ethereum Authors
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

package ethash

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/gnc-project/galaxynetwork/crypto"
	"github.com/gnc-project/galaxynetwork/log"
	"github.com/gnc-project/galaxynetwork/pocmine"
	"github.com/gnc-project/galaxynetwork/pocmine/challenge"
	"github.com/gnc-project/poc/difficulty"
	"math/big"
	"runtime"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/math"
	"github.com/gnc-project/galaxynetwork/consensus"
	"github.com/gnc-project/galaxynetwork/consensus/misc"
	"github.com/gnc-project/galaxynetwork/core/state"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/params"
	"github.com/gnc-project/galaxynetwork/rewardc"
	"github.com/gnc-project/galaxynetwork/rlp"
	"github.com/gnc-project/galaxynetwork/trie"
	"github.com/gnc-project/poc"
	"golang.org/x/crypto/sha3"
)

// Ethash proof-of-work protocol constants.
var (
	FrontierBlockReward       = rewardc.BlockReward // Block reward in wei for successfully mining a block
	ByzantiumBlockReward      = rewardc.BlockReward // Block reward in wei for successfully mining a block upward from Byzantium
	ConstantinopleBlockReward = rewardc.BlockReward // Block reward in wei for successfully mining a block upward from Constantinople
	maxUncles                     = 2                 // Maximum number of uncles allowed in a single block
	allowedFutureBlockTimeSeconds = int64(15)         // Max seconds from current time allowed for blocks, before they're considered future blocks

	// calcDifficultyEip3554 is the difficulty adjustment algorithm as specified by EIP 3554.
	// It offsets the bomb a total of 9.7M blocks.
	// Specification EIP-3554: https://eips.ethereum.org/EIPS/eip-3554
	calcDifficultyEip3554 = makeDifficultyCalculator(big.NewInt(9700000))

	// calcDifficultyEip2384 is the difficulty adjustment algorithm as specified by EIP 2384.
	// It offsets the bomb 4M blocks from Constantinople, so in total 9M blocks.
	// Specification EIP-2384: https://eips.ethereum.org/EIPS/eip-2384
	calcDifficultyEip2384 = makeDifficultyCalculator(big.NewInt(9000000))

	// calcDifficultyConstantinople is the difficulty adjustment algorithm for Constantinople.
	// It returns the difficulty that a new block should have when created at time given the
	// parent block's time and difficulty. The calculation uses the Byzantium rules, but with
	// bomb offset 5M.
	// Specification EIP-1234: https://eips.ethereum.org/EIPS/eip-1234
	calcDifficultyConstantinople = makeDifficultyCalculator(big.NewInt(5000000))

	// calcDifficultyByzantium is the difficulty adjustment algorithm. It returns
	// the difficulty that a new block should have when created at time given the
	// parent block's time and difficulty. The calculation uses the Byzantium rules.
	// Specification EIP-649: https://eips.ethereum.org/EIPS/eip-649
	calcDifficultyByzantium = makeDifficultyCalculator(big.NewInt(3000000))
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errOlderBlockTime    = errors.New("timestamp older than parent")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return ethash.verifyHeader(chain, header, parent, false, seal, time.Now().Unix())
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs  = make(chan int)
		done    = make(chan int, workers)
		errors  = make([]error, len(headers))
		abort   = make(chan struct{})
		unixNow = time.Now().Unix()
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index, unixNow)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool, index int, unixNow int64) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	return ethash.verifyHeader(chain, headers[index], parent, false, seals[index], unixNow)
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	if len(block.Uncles()) == 0 {
		return nil
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := mapset.NewSet(), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestorHeader := chain.GetHeader(parent, number)
		if ancestorHeader == nil {
			break
		}
		ancestors[parent] = ancestorHeader
		// If the ancestor doesn't have any uncles, we don't have to iterate them
		if ancestorHeader.UncleHash != types.EmptyUncleHash {
			// Need to add those uncles to the banned list too
			ancestor := chain.GetBlock(parent, number)
			if ancestor == nil {
				break
			}
			for _, uncle := range ancestor.Uncles() {
				uncles.Add(uncle.Hash())
			}
		}
		parent, number = ancestorHeader.ParentHash, number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Contains(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true, time.Now().Unix()); err != nil {
			return err
		}
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, seal bool, unixNow int64) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if !uncle {
		if header.Time > uint64(unixNow+int64(rewardc.FutureBlockTime)) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errOlderBlockTime
	}
	// Verify the block's difficulty based on its timestamp and parent's difficulty
	expected := ethash.CalcDifficulty(header, parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify the block's gas usage and (if applicable) verify the base fee.
	if !chain.Config().IsLondon(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, expected 'nil'", header.BaseFee)
		}
		if err := misc.VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
			return err
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block

	//poc
	if err := ethash.verifyPoc(header,parent);err != nil {
		return err
	}
	if err := ethash.verifySig(header);err != nil {
		return err
	}

	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (ethash *Ethash) CalcDifficulty(header, parent *types.Header) *big.Int {
	return CalcDifficulty(header, parent)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(header, parent *types.Header) *big.Int {
	// Ensure diff
	lastTime := time.Unix(int64(parent.Time),0)
	blockTime := time.Unix(int64(header.Time),0)
	return difficulty.CalcNextRequiredDifficulty(lastTime,parent.Difficulty,blockTime)
}



// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	bigMinus99    = big.NewInt(-99)
)

// makeDifficultyCalculator creates a difficultyCalculator with the given bomb-delay.
// the difficulty is calculated with Byzantium rules, which differs from Homestead in
// how uncles affect the calculation
func makeDifficultyCalculator(bombDelay *big.Int) func(time uint64, parent *types.Header) *big.Int {
	// Note, the calculations below looks at the parent number, which is 1 below
	// the block number. Thus we remove one from the delay given
	bombDelayFromParent := new(big.Int).Sub(bombDelay, big1)
	return func(time uint64, parent *types.Header) *big.Int {
		// https://github.com/ethereum/EIPs/issues/100.
		// algorithm:
		// diff = (parent_diff +
		//         (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
		//        ) + 2^(periodCount - 2)

		bigTime := new(big.Int).SetUint64(time)
		bigParentTime := new(big.Int).SetUint64(parent.Time)

		// holds intermediate values to make the algo easier to read & audit
		x := new(big.Int)
		y := new(big.Int)

		// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9
		x.Sub(bigTime, bigParentTime)
		x.Div(x, big9)
		if parent.UncleHash == types.EmptyUncleHash {
			x.Sub(big1, x)
		} else {
			x.Sub(big2, x)
		}
		// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
		if x.Cmp(bigMinus99) < 0 {
			x.Set(bigMinus99)
		}
		// parent_diff + (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
		y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
		x.Mul(y, x)
		x.Add(parent.Difficulty, x)

		// minimum difficulty can ever be (before exponential factor)
		if x.Cmp(params.MinimumDifficulty) < 0 {
			x.Set(params.MinimumDifficulty)
		}
		// calculate a fake block number for the ice-age delay
		// Specification: https://eips.ethereum.org/EIPS/eip-1234
		fakeBlockNumber := new(big.Int)
		if parent.Number.Cmp(bombDelayFromParent) >= 0 {
			fakeBlockNumber = fakeBlockNumber.Sub(parent.Number, bombDelayFromParent)
		}
		// for the exponential factor
		periodCount := fakeBlockNumber
		periodCount.Div(periodCount, expDiffPeriod)

		// the exponential factor, commonly referred to as "the bomb"
		// diff = diff + 2^(periodCount - 2)
		if periodCount.Cmp(big1) > 0 {
			y.Sub(periodCount, big2)
			y.Exp(big2, y, nil)
			x.Add(x, y)
		}
		return x
	}
}

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time uint64, parent *types.Header) *big.Int {
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)

	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).SetUint64(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if x.Cmp(params.MinimumDifficulty) < 0 {
		x.Set(params.MinimumDifficulty)
	}
	// for the exponential factor
	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)

	// the exponential factor, commonly referred to as "the bomb"
	// diff = diff + 2^(periodCount - 2)
	if periodCount.Cmp(big1) > 0 {
		y.Sub(periodCount, big2)
		y.Exp(big2, y, nil)
		x.Add(x, y)
	}
	return x
}

// calcDifficultyFrontier is the difficulty adjustment algorithm. It returns the
// difficulty that a new block should have when created at time given the parent
// block's time and difficulty. The calculation uses the Frontier rules.
func calcDifficultyFrontier(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.SetUint64(parent.Time)

	if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		diff.Set(params.MinimumDifficulty)
	}

	periodCount := new(big.Int).Add(parent.Number, big1)
	periodCount.Div(periodCount, expDiffPeriod)
	if periodCount.Cmp(big1) > 0 {
		// diff = diff + 2^(periodCount - 2)
		expDiff := periodCount.Sub(periodCount, big2)
		expDiff.Exp(big2, expDiff, nil)
		diff.Add(diff, expDiff)
		diff = math.BigMax(diff, params.MinimumDifficulty)
	}
	return diff
}

// Exported for fuzzing
var FrontierDifficultyCalulator = calcDifficultyFrontier
var HomesteadDifficultyCalulator = calcDifficultyHomestead
var DynamicDifficultyCalculator = makeDifficultyCalculator

// verifySeal checks whether a block satisfies the PoW difficulty requirements,
// either using the usual ethash cache for it, or alternatively using a full DAG
// to make remote mining fast.
func (ethash *Ethash) verifySeal(chain consensus.ChainHeaderReader, header *types.Header, fulldag bool) error {

	if err := ethash.verifyPoc(header,chain.CurrentHeader());err != nil {
		return err
	}
	if err := ethash.verifySig(header);err != nil {
		return err
	}

	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = ethash.CalcDifficulty(header, parent)
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state on the header
func (ethash *Ethash) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// Accumulate any block and uncle rewards and commit the final state root
	accumulateRewards(chain, state, header, uncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// FinalizeAndAssemble implements consensus.Engine, accumulating the block and
// uncle rewards, setting the final state and assembling the block.
func (ethash *Ethash) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Finalize block
	ethash.Finalize(chain, header, state, txs, uncles)

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil)), nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (ethash *Ethash) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Extra,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(chain consensus.ChainHeaderReader, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Skip block reward in catalyst mode
	if chain.Config().IsCatalyst(header.Number) {
		return
	}
	reward := rewardc.GetReward(header.Number.Uint64())
	rewardLock, available, lockedRewardVestingSpec := LockedRewardFromReward(new(big.Int).Mul(new(big.Int).Div(reward,big.NewInt(100)),rewardc.MineRewardProportion))

	// unlocked coins
	amountUnlocked := CalculateAmountUnlocked(header.Number, state.GetFunds(header.Coinbase))
	// Accumulate the rewards for the miner and any included uncles
	for _, uncle := range uncles {
		state.AddBalance(uncle.Coinbase, common.Big0)
	}
	state.AddBalance(header.Coinbase, new(big.Int).Add(available, amountUnlocked))

	//calculate funds
	funds := CalculateLockedFunds(header.Number, rewardLock, lockedRewardVestingSpec,state.GetFunds(header.Coinbase))

	for k,v := range funds {
		if v.BlockNumber.Cmp(header.Number) > 0 {
			funds = funds[k:]
			break
		}
	}
	state.SetFunds(header.Coinbase,funds)


	//staking
	stakingList := state.GetAllStakingList(common.AllStakingDB)
	newStakingList := common.StakingList{}
	stakingMap := make(map[string]*common.StakingWeight,0)

	for _,v := range stakingList {
		if v.StartNumber + ( v.FrozenPeriod.Uint64() * rewardc.DayBlock ) > header.Number.Uint64() {
			newStakingList = append(newStakingList,v)
			if sw,ok := stakingMap[v.Account.Hex()]; ok{
				sw.Weight = new(big.Int).Add(sw.Weight,rewardc.CalculateWeight(v.FrozenPeriod,v.Value))
				stakingMap[v.Account.Hex()] = sw
			}else {
				stakingWeight := &common.StakingWeight{Account: v.Account, Weight: rewardc.CalculateWeight(v.FrozenPeriod,v.Value)}
				stakingMap[v.Account.Hex()] = stakingWeight
			}
		}else {
			// free
			state.AddBalance(v.Account,v.Value)
		}
	}

	state.SetStakingList(common.AllStakingDB,newStakingList)

	rewardStaking := make([]*common.StakingWeight, len(stakingMap))
	totalWeight := big.NewInt(0)
	stakingReward := new(big.Int).Mul(new(big.Int).Div(reward,big.NewInt(100)),rewardc.StakingRewardProportion)
	for _,v := range stakingMap {
		rewardStaking = append(rewardStaking,v)
		totalWeight.Add(totalWeight,v.Weight)
	}

	if len(stakingMap) <= rewardc.StakingNum {
		for _, v := range stakingMap {
			accReward := new(big.Int).Mul(new(big.Int).Div(stakingReward,totalWeight),v.Weight)
			state.AddBalance(v.Account,accReward)
		}
		return
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

	rewardStaking = rewardStaking[:rewardc.StakingNum]
	totalWeight = big.NewInt(0)
	for _,v := range rewardStaking {
		totalWeight.Add(totalWeight,v.Weight)
	}
	for _, v := range rewardStaking {
		accReward := new(big.Int).Mul(new(big.Int).Div(stakingReward,totalWeight),v.Weight)
		state.AddBalance(v.Account,accReward)
	}
}

func (ethhash *Ethash) verifyPoc(header, parent *types.Header) error {

	if err := ethhash.verifyHeaderTimestamp(header,parent); err != nil {
		return err
	}

	if err := ethhash.verifyProofOfCapacity(header); err != nil {
		return err
	}

	if err := ethhash.verifyDiff(header,parent); err != nil {
		return err
	}

	if err := ethhash.verifyChallenge(header,parent); err != nil {
		return err
	}

	return nil
}

func (ethhash *Ethash) verifySig(header *types.Header) error {

	pub,err := crypto.Ecrecover(pocmine.ShaInput(header),header.Signed)
	if err != nil {
		return err
	}
	signatureNoRecoverID := header.Signed[:len(header.Signed)-1]
	if !pocmine.Verify(pub,header,signatureNoRecoverID) {
		return errors.New("sig is err")
	}

	return nil
}

func (ethhash *Ethash)verifyHeaderTimestamp(header,parent *types.Header) error {

	if int64(header.Time) > time.Now().Unix() {
		log.Error("block timestamp of unix is too far in the future",
			"allowed",      time.Now().Unix(),
			"timestamp_unix", header.Time,
			"height",        header.Number,
			"block",          header.Hash().Hex(),
		)
		return errors.New("block timestamp is too far in the future")
	}

	before := time.Unix(int64(parent.Time),0).Add(poc.PoCSlot * time.Second)
	if int64(header.Time) < before.Unix() {
		log.Error("block timestamp of unix is too near in the future","allowed")
		return errors.New("block timestamp of unix is too near in the future")
	}

	return nil
}

func (ethhash *Ethash)verifyProofOfCapacity(header *types.Header) error {
	// The Target difficulty must be larger than zero.
	target := header.Difficulty
	if target.Sign() <= 0 {
		log.Error("block Target difficulty is too low", "target", target)
		return errors.New("block difficulty is not the expected value")
	}

	// The Target difficulty must be less than the maximum allowed.
	if target.Cmp(rewardc.MainPocLimit) < 0 {
		log.Error("block Target difficulty is lower than min of pocLimit",
			"target", target, "pocLimit", rewardc.MainPocLimit)
		return errors.New("block difficulty is not the expected value")
	}

	slot := header.Time / poc.PoCSlot
	quality,err := poc.VerifiedQuality(header.Proof,header.Pid,header.Challenge,slot,header.Number.Uint64(),header.K)
	if err != nil {
		return err
	}

	if quality.Cmp(target) <= 0 {
		log.Error("block's proof quality is lower than expected min target",
			"quality", quality, "expected", target, "height", header.Number, "hash", header.Hash().Hex())
		return errors.New("block's proof quality is lower than expected min target")
	}

	return nil
}

func (ethash *Ethash)verifyDiff(header,parent *types.Header) error {

	// Ensure diff
	expectedDiff := ethash.CalcDifficulty(header,parent)
	blockDifficulty := header.Difficulty
	if blockDifficulty.Cmp(expectedDiff) != 0 {
		log.Error("block difficulty is not the expected value",
			"difficulty", blockDifficulty, "expectedTarget", expectedDiff)
		return ErrUnexpectedDifficulty
	}

	return nil
}

func (ethash *Ethash)verifyChallenge(header,parent *types.Header) error  {

	// Ensure the provided challenge in header is right.
	// The calculated challenge based on some rules.
	challenge := CalcNextChallenge(parent)
	if ! ( *challenge == header.Challenge ){
		log.Error("block challenge does not match the expected challenge",
			"block challenge", header.Challenge.Hex(), "blockHeight", header.Number, "expectedChallenge", challenge.Hex())
		return ErrUnexpectedChallenge
	}
	return nil
}

func CalcNextChallenge(parent *types.Header) *common.Hash {

	if parent.Number.Uint64() == rewardc.GenesisNumber {
		hash := parent.Hash()
		h := sha256.Sum256(hash[:])
		nextHash := common.BytesToHash(h[:])
		return &nextHash
	}

	return challenge.CalcNextChallenge(parent)
}



var (
	LockedRewardFactorNum   = big.NewInt(75)
	LockedRewardFactorDenom = big.NewInt(100)
)

type VestingFund struct {
	Epoch  int64
	Amount *big.Int
}

type VestSpec struct {
	MinExpiration int64
	VestPeriod    int64
	StepDuration  int64
}

var RewardVestingSpec = VestSpec{
	MinExpiration: rewardc.MinSectorExpiration,
	VestPeriod:    int64(rewardc.MinSectorExpiration * rewardc.DayBlock),
	StepDuration:  int64(1 * rewardc.DayBlock),
}

// LockedRewardFromReward Calculate and lock 75% of coins
func LockedRewardFromReward(reward *big.Int) (*big.Int, *big.Int, *VestSpec) {
	spec := &RewardVestingSpec
	lockAmount := big.NewInt(0).Div(big.NewInt(0).Mul(reward, LockedRewardFactorNum), LockedRewardFactorDenom)
	return lockAmount, big.NewInt(0).Sub(reward, lockAmount), spec
}


//CalculateLockedFunds Linear release
func CalculateLockedFunds(num *big.Int, vestingSum *big.Int, spec *VestSpec,funds common.MinedBlocks) common.MinedBlocks {
	if vestingSum.Cmp(common.Big0) < 0 {
		return funds
	}

	epochToIndex := make(map[*big.Int]int, len(funds))
	for i, vf := range funds {
		epochToIndex[vf.BlockNumber] = i
	}


	vestEpoch := num
	dayAmount := new(big.Int).Div(vestingSum,big.NewInt(spec.MinExpiration))

	for i:=int64(0); i < spec.MinExpiration; i++{

		vestEpoch = new(big.Int).Add(vestEpoch,big.NewInt(spec.StepDuration))

		if index, ok := epochToIndex[vestEpoch]; ok {
			currentAmt := funds[index].Amount
			funds[index].Amount = big.NewInt(0).Add(currentAmt, dayAmount)
		}else {
			entry := common.MinedBlock{BlockNumber: vestEpoch,Amount: dayAmount}
			funds = append(funds, &entry)
		}

	}

	sort.SliceStable(funds, func(first, second int) bool {
		return funds[first].BlockNumber.Cmp(funds[second].BlockNumber) < 0
	})
	return funds
}

func CalculateAmountUnlocked(num *big.Int,funds common.MinedBlocks) *big.Int  {
	amountUnlocked := big.NewInt(0)
	for _, vf := range funds {
		if vf.BlockNumber.Cmp(num) > 0 {
			continue
		}
		amountUnlocked.Add(amountUnlocked, vf.Amount)
	}

	return amountUnlocked
}
