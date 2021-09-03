package challenge

import (
	"bytes"
	"crypto/sha256"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

const (
	// New challenge is calculated based on 10 blocks before.
	ChallengeInterval = 10
	MaxReferredBlocks = ChallengeInterval * 2
)

func CalcNextChallenge(parent *types.Header) *common.Hash {

	// Deal with initial <ChallengeInterval> blocks.
	if parent.Number.Uint64() < ChallengeInterval {
		hash := parent.Hash()
		h := sha256.Sum256(hash[:])
		nextHash := common.BytesToHash(h[:])
		return &nextHash
	}

	hash := parent.Hash()
	input := BytesCombines(parent.Pid[:],parent.Proof,hash[:],parent.ParentHash[:],
		parent.Number.Bytes(),new(big.Int).SetUint64(parent.K).Bytes())
	proofHash := sha256.Sum256(input)
	nextHash := common.BytesToHash(proofHash[:])

	return &nextHash
}

func BytesCombine(pBytes ...[]byte) []byte {
	len := len(pBytes)
	s := make([][]byte, len)
	for index := 0; index < len; index++ {
		s[index] = pBytes[index]
	}
	sep := []byte("")
	return bytes.Join(s, sep)
}

func BytesCombines(pBytes ...[]byte) []byte {
	var buffer bytes.Buffer
	len := len(pBytes)
	for index := 0; index < len; index++ {
		buffer.Write(pBytes[index])
	}
	return buffer.Bytes()
}