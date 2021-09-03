package pocmine

import (
	"crypto/sha256"
	"github.com/gnc-project/galaxynetwork/common"
	"math/big"
	"sync"
)

type Generator struct {
	sync.Map
}

var gen  *Generator

func init()  {
	gen = &Generator{}
}

func GetGenerator() *Generator {
	return gen
}

func (ge *Generator)AddWorkPoc(wp *WorkPoc)  {
	ge.Store(wp.GeyKey(),wp)
}

func (ge *Generator)GetWorkPoc(pidNumber common.Hash)*WorkPoc  {
	if st,ok := ge.Load(pidNumber);ok{
		return st.(*WorkPoc)
	}else {
		return nil
	}
}

type WorkPoc struct {
	Pid 		common.Hash
	Proof   	[]byte
	K 			uint8
	Difficulty 	*big.Int
	Number 		*big.Int
	Timestamp 	int64
}

func NewWorkPoc(pid common.Hash,Proof []byte, k uint8, diff *big.Int, number *big.Int,timestamp int64)*WorkPoc  {
	return &WorkPoc{
		pid,
		Proof,
		k,
		diff,
		number,
		timestamp,
	}
}

func (s *WorkPoc)GeyKey() common.Hash {
	return sha256.Sum256(append(s.Pid[:],s.Number.Bytes()...))
}

