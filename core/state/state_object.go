// Copyright 2014 The go-ethereum Authors
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

package state

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"time"
	"sort"
	"errors"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

var emptyCodeHash = crypto.Keccak256(nil)

type Code []byte

func (c Code) String() string {
	return string(c) //strings.Join(Disassemble(c), " ")
}

type Storage map[common.Hash]common.Hash

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address  common.Address
	addrHash common.Hash // hash of ethereum address of the account
	data     Account
	db       *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites, reset for every transaction
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution
	fakeStorage    Storage // Fake storage which constructed by caller for debugging purpose.

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}


// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && s.data.Balance.Sign() == 0 && bytes.Equal(s.data.CodeHash, emptyCodeHash)
}

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
	TotalLockedFunds *big.Int //lock coinbase
	Pledge *big.Int //Balance of miners pledged
	CanRedeem common.CanRedeemList
	Funds []struct {
		BlockNumber *big.Int
		Amount      *big.Int
	} //Balance of miners Fund by BlockNumber

	Pid   common.PidList
	Staking common.StakingList
}

// newObject creates a state object.
func newObject(db *StateDB, address common.Address, data Account) *stateObject {
	if data.Balance == nil {
		data.Balance = new(big.Int)
	}

	if data.Pledge == nil {
		data.Pledge = new(big.Int)
	}

	if data.CanRedeem == nil {
		data.CanRedeem =common.CanRedeemList{}
	}

	if data.TotalLockedFunds == nil {
		data.TotalLockedFunds = new(big.Int)
	}
	if data.CodeHash == nil {
		data.CodeHash = emptyCodeHash
	}
	if data.Root == (common.Hash{}) {
		data.Root = emptyRoot
	}

	if data.Pid == nil{
	   data.Pid=common.PidList{}
	}

    if data.Staking==nil{
		data.Staking=common.StakingList{}
	}

	return &stateObject{
		db:             db,
		address:        address,
		addrHash:       crypto.Keccak256Hash(address[:]),
		data:           data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// EncodeRLP implements rlp.Encoder.
func (s *stateObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, s.data)
}

// setError remembers the first non-nil error it is called with.
func (s *stateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *stateObject) markSuicided() {
	s.suicided = true
}

func (s *stateObject) touch() {
	s.db.journal.append(touchChange{
		account: &s.address,
	})
	if s.address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		s.db.journal.dirty(s.address)
	}
}

func (s *stateObject) getTrie(db Database) Trie {
	if s.trie == nil {
		// Try fetching from prefetcher first
		// We don't prefetch empty tries
		if s.data.Root != emptyRoot && s.db.prefetcher != nil {
			// When the miner is creating the pending state, there is no
			// prefetcher
			s.trie = s.db.prefetcher.trie(s.data.Root)
		}
		if s.trie == nil {
			var err error
			s.trie, err = db.OpenStorageTrie(s.addrHash, s.data.Root)
			if err != nil {
				s.trie, _ = db.OpenStorageTrie(s.addrHash, common.Hash{})
				s.setError(fmt.Errorf("can't create storage trie: %v", err))
			}
		}
	}
	return s.trie
}

// GetState retrieves a value from the account storage trie.
func (s *stateObject) GetState(db Database, key common.Hash) common.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a dirty value for this state entry, return it
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value
	}
	// Otherwise return the entry's original value
	return s.GetCommittedState(db, key)
}

// GetCommittedState retrieves a value from the committed account storage trie.
func (s *stateObject) GetCommittedState(db Database, key common.Hash) common.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}
	// If no live objects are available, attempt to use snapshots
	var (
		enc   []byte
		err   error
		meter *time.Duration
	)
	readStart := time.Now()
	if metrics.EnabledExpensive {
		// If the snap is 'under construction', the first lookup may fail. If that
		// happens, we don't want to double-count the time elapsed. Thus this
		// dance with the metering.
		defer func() {
			if meter != nil {
				*meter += time.Since(readStart)
			}
		}()
	}
	if s.db.snap != nil {
		if metrics.EnabledExpensive {
			meter = &s.db.SnapshotStorageReads
		}
		// If the object was destructed in *this* block (and potentially resurrected),
		// the storage has been cleared out, and we should *not* consult the previous
		// snapshot about any storage values. The only possible alternatives are:
		//   1) resurrect happened, and new slot values were set -- those should
		//      have been handles via pendingStorage above.
		//   2) we don't have new values, and can deliver empty response back
		if _, destructed := s.db.snapDestructs[s.addrHash]; destructed {
			return common.Hash{}
		}
		enc, err = s.db.snap.Storage(s.addrHash, crypto.Keccak256Hash(key.Bytes()))
	}
	// If snapshot unavailable or reading from it failed, load from the database
	if s.db.snap == nil || err != nil {
		if meter != nil {
			// If we already spent time checking the snapshot, account for it
			// and reset the readStart
			*meter += time.Since(readStart)
			readStart = time.Now()
		}
		if metrics.EnabledExpensive {
			meter = &s.db.StorageReads
		}
		if enc, err = s.getTrie(db).TryGet(key.Bytes()); err != nil {
			s.setError(err)
			return common.Hash{}
		}
	}
	var value common.Hash
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			s.setError(err)
		}
		value.SetBytes(content)
	}
	s.originStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (s *stateObject) SetState(db Database, key, value common.Hash) {
	// If the fake storage is set, put the temporary state update here.
	if s.fakeStorage != nil {
		s.fakeStorage[key] = value
		return
	}
	// If the new value is the same as old, don't set
	prev := s.GetState(db, key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
	})
	s.setState(key, value)
}

// SetStorage replaces the entire state storage with the given one.
//
// After this function is called, all original state will be ignored and state
// lookup only happens in the fake state storage.
//
// Note this function should only be used for debugging purpose.
func (s *stateObject) SetStorage(storage map[common.Hash]common.Hash) {
	// Allocate fake storage if it's nil.
	if s.fakeStorage == nil {
		s.fakeStorage = make(Storage)
	}
	for key, value := range storage {
		s.fakeStorage[key] = value
	}
	// Don't bother journal since this function should only be used for
	// debugging and the `fake` storage won't be committed to database.
}

func (s *stateObject) setState(key, value common.Hash) {
	s.dirtyStorage[key] = value
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (s *stateObject) finalise(prefetch bool) {
	slotsToPrefetch := make([][]byte, 0, len(s.dirtyStorage))
	for key, value := range s.dirtyStorage {
		s.pendingStorage[key] = value
		if value != s.originStorage[key] {
			slotsToPrefetch = append(slotsToPrefetch, common.CopyBytes(key[:])) // Copy needed for closure
		}
	}
	if s.db.prefetcher != nil && prefetch && len(slotsToPrefetch) > 0 && s.data.Root != emptyRoot {
		s.db.prefetcher.prefetch(s.data.Root, slotsToPrefetch)
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
}

// updateTrie writes cached storage modifications into the object's storage trie.
// It will return nil if the trie has not been loaded and no changes have been made
func (s *stateObject) updateTrie(db Database) Trie {
	// Make sure all dirty slots are finalized into the pending storage area
	s.finalise(false) // Don't prefetch any more, pull directly if need be
	if len(s.pendingStorage) == 0 {
		return s.trie
	}
	// Track the amount of time wasted on updating the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageUpdates += time.Since(start) }(time.Now())
	}
	// The snapshot storage map for the object
	var storage map[common.Hash][]byte
	// Insert all the pending updates into the trie
	tr := s.getTrie(db)
	hasher := s.db.hasher

	usedStorage := make([][]byte, 0, len(s.pendingStorage))
	for key, value := range s.pendingStorage {
		// Skip noop changes, persist actual changes
		if value == s.originStorage[key] {
			continue
		}
		s.originStorage[key] = value

		var v []byte
		if (value == common.Hash{}) {
			s.setError(tr.TryDelete(key[:]))
		} else {
			// Encoding []byte cannot fail, ok to ignore the error.
			v, _ = rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
			s.setError(tr.TryUpdate(key[:], v))
		}
		// If state snapshotting is active, cache the data til commit
		if s.db.snap != nil {
			if storage == nil {
				// Retrieve the old storage map, if available, create a new one otherwise
				if storage = s.db.snapStorage[s.addrHash]; storage == nil {
					storage = make(map[common.Hash][]byte)
					s.db.snapStorage[s.addrHash] = storage
				}
			}
			storage[crypto.HashData(hasher, key[:])] = v // v will be nil if value is 0x00
		}
		usedStorage = append(usedStorage, common.CopyBytes(key[:])) // Copy needed for closure
	}
	if s.db.prefetcher != nil {
		s.db.prefetcher.used(s.data.Root, usedStorage)
	}
	if len(s.pendingStorage) > 0 {
		s.pendingStorage = make(Storage)
	}
	return tr
}

// UpdateRoot sets the trie root to the current root hash of
func (s *stateObject) updateRoot(db Database) {
	// If nothing changed, don't bother with hashing anything
	if s.updateTrie(db) == nil {
		return
	}
	// Track the amount of time wasted on hashing the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageHashes += time.Since(start) }(time.Now())
	}
	s.data.Root = s.trie.Hash()
}

// CommitTrie the storage trie of the object to db.
// This updates the trie root.
func (s *stateObject) CommitTrie(db Database) error {
	// If nothing changed, don't bother with hashing anything
	if s.updateTrie(db) == nil {
		return nil
	}
	if s.dbErr != nil {
		return s.dbErr
	}
	// Track the amount of time wasted on committing the storage trie
	if metrics.EnabledExpensive {
		defer func(start time.Time) { s.db.StorageCommits += time.Since(start) }(time.Now())
	}
	root, err := s.trie.Commit(nil)
	if err == nil {
		s.data.Root = root
	}
	return err
}

// AddBalance adds amount to s's balance.
// It is used to add funds to the destination account of a transfer.
func (s *stateObject) AddBalance(amount *big.Int) {
	// EIP161: We must check emptiness for the objects such that the account
	// clearing (0,0,0 objects) can take effect.
	if amount.Sign() == 0 {
		if s.empty() {
			s.touch()
		}
		return
	}
	s.SetBalance(new(big.Int).Add(s.Balance(), amount))
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.Balance(), amount))
}

func (s *stateObject) SetBalance(amount *big.Int) {
	s.db.journal.append(balanceChange{
		account: &s.address,
		prev:    new(big.Int).Set(s.data.Balance),
	})
	s.setBalance(amount)
}

func (s *stateObject) setBalance(amount *big.Int) {
	s.data.Balance = amount
}

func (s *stateObject) SetFunds(funds []struct {
	BlockNumber *big.Int
	Amount      *big.Int
}) {
	s.db.journal.append(fundsChange{
		account: &s.address,
		prev:    s.data.Funds,
	})
	s.setFunds(funds)
}

func (s *stateObject) setFunds(funds []struct {
	BlockNumber *big.Int
	Amount      *big.Int
}) {
	s.data.Funds = funds
}

// AddPledge adds amount to Pledge's balance.
func (s *stateObject) AddPledge(amount *big.Int) {
	if amount.Sign() == 0 {
		if s.empty() {
			s.touch()
		}
		return
	}
	s.SetPledge(new(big.Int).Add(s.Pledge(), amount))
}

func (s *stateObject) SubPledge(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetPledge(new(big.Int).Sub(s.Pledge(), amount))
}

func (s *stateObject) SetPledge(amount *big.Int) {
	s.db.journal.append(pledgeChange{
		account: &s.address,
		prev:    new(big.Int).Set(s.data.Pledge),
	})
	s.setPledge(amount)
}

func (s *stateObject) setPledge(amount *big.Int) {
	s.data.Pledge = amount
}

func (s *stateObject) AddTotalLockedFunds(amount *big.Int) {
	if amount.Sign() == 0 {
		if s.empty() {
			s.touch()
		}
		return
	}
	s.SetTotalLockedFunds(new(big.Int).Add(s.TotalLockedFunds(), amount))
}

func (s *stateObject) SubTotalLockedFunds(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetTotalLockedFunds(new(big.Int).Sub(s.TotalLockedFunds(), amount))
}

func (s *stateObject) SetTotalLockedFunds(amount *big.Int) {
	s.db.journal.append(totalLockedFundsChange{
		account: &s.address,
		prev:    new(big.Int).Set(s.data.TotalLockedFunds),
	})
	s.setTotalLockedFunds(amount)
}

func (s *stateObject) setTotalLockedFunds(amount *big.Int) {
	s.data.TotalLockedFunds = amount
}

func (s *stateObject) AddPid(pidHex []byte,amount *big.Int) error {
	if len(pidHex)!= PidHashLength {
		return errors.New("invalid PidHashLength")
	}
	
	s.SetPid(pidHex,amount)
	return nil
}

func (s *stateObject) SubPid(pidHex []byte) *big.Int{
	s.db.journal.append(pidChange{
		account: &s.address,
		prev:    s.data.Pid,
	})
	for index,pid:=range s.data.Pid{
       if hex.EncodeToString(pidHex)==pid.PidHex{
		s.data.Pid=append(s.data.Pid[:index], s.data.Pid[index+1:]...)
		return pid.PledgeAmount
	   }
	}
	return common.Big0
}


func (s *stateObject) SetPid(pidHex []byte,amount *big.Int) error {
	s.db.journal.append(pidChange{
		account: &s.address,
		prev:    s.data.Pid,
	})
	pid:=&common.Pid{
		PidHex: hex.EncodeToString(pidHex),
		PledgeAmount:amount,
	}
    s.data.Pid=append(s.data.Pid,pid)

	s.setPid(s.data.Pid)
	return nil
}

func (s *stateObject) setPid(pid common.PidList) {
	s.data.Pid=pid
}
func (s *stateObject) GetRedeemAmount(number uint64)*big.Int{
	redeemBalance:=big.NewInt(0)
	for _,canRedeem:=range s.CanRedeem(){
		if canRedeem.UnlockBlock<number{
			redeemBalance=new(big.Int).Add(redeemBalance,canRedeem.RedeemAmount)
		}
	}
	return redeemBalance
}


func (s *stateObject) AddStakingList(address common.Address,addStaking *common.Staking) error {
	s.db.journal.append(stakingListChange{
		account: &s.address,
		prev:    s.data.Staking,
	})
	stakingMap:=map[common.Address]*common.Staking{}
	for _,lastStaking:=range s.data.Staking{
		stakingMap[*lastStaking.Address]=lastStaking
	}
	if stakingMap[address]==nil{
		stakingMap[address]=addStaking
	}else{
		stakingMap[*addStaking.Address].StakingInfo=append(stakingMap[*addStaking.Address].StakingInfo,addStaking.StakingInfo...)
		stakingMap[*addStaking.Address].TotalValue=new(big.Int).Add(stakingMap[*addStaking.Address].TotalValue,addStaking.TotalValue)
		stakingMap[*addStaking.Address].TotalWeight=new(big.Int).Add(stakingMap[*addStaking.Address].TotalWeight,addStaking.TotalWeight)
	}

	var addresss []string
	for address := range stakingMap {
		addresss = append(addresss, address.Hex())
	}
	sort.Strings(addresss)
	
	var stakingList common.StakingList

	for _, address := range addresss {
		stakingList=append(stakingList, stakingMap[common.HexToAddress(address)])
	}
	sort.Stable(stakingList)
	s.setStakingList(stakingList)
	return nil
}

func (s *stateObject) SubStakingList(address common.Address,nowHeight uint64){
	s.db.journal.append(stakingListChange{
		account: &s.address,
		prev:    s.data.Staking,
	})
	for _,lastStaking:=range s.data.Staking{
		if *lastStaking.Address==address{
			for index,stakingInfo:=range lastStaking.StakingInfo{
				if stakingInfo.StopBlock<nowHeight{
					lastStaking.TotalWeight=new(big.Int).Sub(lastStaking.TotalWeight,stakingInfo.Weight)
					lastStaking.TotalValue=new(big.Int).Sub(lastStaking.TotalValue,stakingInfo.Value)
					lastStaking.StakingInfo[index].Value=big.NewInt(0)
					lastStaking.StakingInfo[index].Weight=big.NewInt(0)
				}
			}
		}
	}
	sort.Stable(s.data.Staking)
	s.setStakingList(s.data.Staking)
}

func (s *stateObject) setStakingList(stakinglist common.StakingList) {
	s.data.Staking=stakinglist
}

func (s *stateObject) AddCanRedeem(number uint64,amount *big.Int) {

	var newCanRedeem=&common.CanRedeem{
		UnlockBlock: number,
		RedeemAmount: amount,
	}

	s.SetCanRedeem(newCanRedeem,-1)
}

// SubBalance removes amount from s's balance.
// It is used to remove funds from the origin account of a transfer.
func (s *stateObject) SubCanRedeem(index int64) {
	s.SetCanRedeem(nil,index)
}

func (s *stateObject) SetCanRedeem(newCanRedeem *common.CanRedeem,index int64) {
	s.db.journal.append(canReDeemChange{
		account: &s.address,
		prev:    s.data.CanRedeem,
	})
	if index==-1{
		s.data.CanRedeem=append(s.data.CanRedeem, newCanRedeem)

	}else{
        s.data.CanRedeem=append(s.data.CanRedeem[:index],s.data.CanRedeem[index+1:]...)
	}
	s.setCanRedeem(s.data.CanRedeem)
}

func (s *stateObject) setCanRedeem(CanRedeem common.CanRedeemList) {
	s.data.CanRedeem =CanRedeem
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	stateObject := newObject(db, s.address, s.data)
	if s.trie != nil {
		stateObject.trie = db.db.CopyTrie(s.trie)
	}
	stateObject.code = s.code
	stateObject.dirtyStorage = s.dirtyStorage.Copy()
	stateObject.originStorage = s.originStorage.Copy()
	stateObject.pendingStorage = s.pendingStorage.Copy()
	stateObject.suicided = s.suicided
	stateObject.dirtyCode = s.dirtyCode
	stateObject.deleted = s.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateObject) Address() common.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code(db Database) []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return nil
	}
	code, err := db.ContractCode(s.addrHash, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.setError(fmt.Errorf("can't load code hash %x: %v", s.CodeHash(), err))
	}
	s.code = code
	return code
}

// CodeSize returns the size of the contract code associated with this object,
// or zero if none. This method is an almost mirror of Code, but uses a cache
// inside the database to avoid loading codes seen recently.
func (s *stateObject) CodeSize(db Database) int {
	if s.code != nil {
		return len(s.code)
	}
	if bytes.Equal(s.CodeHash(), emptyCodeHash) {
		return 0
	}
	size, err := db.ContractCodeSize(s.addrHash, common.BytesToHash(s.CodeHash()))
	if err != nil {
		s.setError(fmt.Errorf("can't load code size %x: %v", s.CodeHash(), err))
	}
	return size
}

func (s *stateObject) SetCode(codeHash common.Hash, code []byte) {
	prevcode := s.Code(s.db.db)
	s.db.journal.append(codeChange{
		account:  &s.address,
		prevhash: s.CodeHash(),
		prevcode: prevcode,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash common.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Balance() *big.Int {
	return s.data.Balance
}

func (s *stateObject) TotalLockedFunds() *big.Int {
	return s.data.TotalLockedFunds
}

func (s *stateObject) Funds() []struct {
	BlockNumber *big.Int
	Amount      *big.Int
} {
	return s.data.Funds
}

func (s *stateObject) Pledge() *big.Int {
	return s.data.Pledge
}


func (s *stateObject) StakingByAddr(addr common.Address)*common.Staking {
	for _,staking:=range s.data.Staking{
		if *staking.Address==addr{
			return staking
		}
	}
	return nil
}

func (s *stateObject) AllStaking()common.StakingList{

	return s.data.Staking
}

func (s *stateObject) Pid() common.PidList{
	return s.data.Pid
}

func (s *stateObject) CanRedeem()common.CanRedeemList{
	return s.data.CanRedeem
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

// Never called, but must be present to allow stateObject to be used
// as a vm.Account interface that also satisfies the vm.ContractRef
// interface. Interfaces are awesome.
func (s *stateObject) Value() *big.Int {
	panic("Value on stateObject should never be called")
}

var  PidHashLength=32