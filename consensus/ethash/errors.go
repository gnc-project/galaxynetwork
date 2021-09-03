package ethash

import "errors"

var (
	ErrUnexpectedDifficulty 	= errors.New("block difficulty is not the expected value")
	ErrUnexpectedChallenge       = errors.New("block challenge is not the expected value")
)

