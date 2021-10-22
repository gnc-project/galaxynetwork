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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/math"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/params"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type diffTest struct {
	ParentTimestamp    uint64
	ParentDifficulty   *big.Int
	CurrentTimestamp   uint64
	CurrentBlocknumber *big.Int
	CurrentDifficulty  *big.Int
}

func TestLockedRewardFromReward(t *testing.T)  {
	r := big.NewInt(0).Mul(big.NewInt(600),big.NewInt(1e18))
	a,b := LockedRewardFromReward(r)
	fmt.Println(a,b)
}

func TestCalculateLockedFunds(t *testing.T) {

	funds := common.MinedBlocks{}
	reawrd := new(big.Int).Mul(big.NewInt(360),big.NewInt(1e18))
	amount := big.NewInt(0)
	total := big.NewInt(0)
	for i:=1; i<= 21;i++ {

		now := time.Now().UnixNano()/1e6
		if i < 12 {
			funds = CalculateLockedFunds(big.NewInt(int64(i)),reawrd,funds)
		}
		amount,funds = CalculateAmountUnlocked(big.NewInt(int64(i)),funds)
		fmt.Println("ep",time.Now().UnixNano() /1e6 - now)
		fmp := make(map[*big.Int]interface{})
		for _,v := range funds {
			fmp[v.BlockNumber] = v
			//fmt.Println(v)
		}
		if len(funds) != len(fmp) {
			panic("nooooooooooooooooooooooooooooooo")
		}
		total = new(big.Int).Add(total,amount)
		fmt.Println("number",i,"amount",amount,"funds",len(funds),"fmp",len(fmp),"total",total)
		fmt.Println("---------------------------------------------------------------------------------------------------------")

		//fmt.Println("fmp --len--->",len(fmp),"number",i)

		//tim := time.Now().Unix()
		//amount,_ := CalculateAmountUnlocked(big.NewInt(int64(i)),funds)
		////fundsBefore := len(funds)
		//for k,v := range funds {
		//	if v.BlockNumber.Cmp(big.NewInt(int64(i))) > 0 {
		//		funds = funds[k:]
		//		//fmt.Printf("blockNumber=%v  k=%v  fundsAfter=%v fundsBefore=%v \n",v.BlockNumber,k, len(funds),fundsBefore)
		//		break
		//	}
		//}
		//fmt.Println(time.Now().Unix()-tim)
		//
		//fmt.Println("amount ---",amount,"funds len-->", len(funds))
	}




	//
	//funds = CalculateLockedFunds(big.NewInt(3),big.NewInt(600 *1e8),spec,funds)
	//fmt.Println(len(funds))
	//
	//funds = CalculateLockedFunds(big.NewInt(4),big.NewInt(600 *1e8),spec,funds)
	//fmt.Println(len(funds))
}

func TestCalcNextChallenge(t *testing.T) {
	parent := types.Header{}
	err := parent.UnmarshalJSON([]byte("{\"parentHash\":\"0x81eda3140a4e53006fce12213a5e58fa8dae9dc1e17a25fc859cbe7dd9f1aac3\",\"sha3Uncles\":\"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347\",\"miner\":\"0xe1500ea2146dc05cd55b1b33bb5ad277141a5f4d\",\"stateRoot\":\"0x16fef91579a58b0d2954f60981a59efab1ebacfa18494409c014c8ddd5c65a4b\",\"transactionsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"receiptsRoot\":\"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421\",\"logsBloom\":\"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\",\"difficulty\":\"0x4046a9dd0fff43\",\"number\":\"0x201\",\"gasLimit\":\"0x9b176a\",\"gasUsed\":\"0x0\",\"timestamp\":\"0x61288af4\",\"extraData\":\"0xd883010a08846765746888676f312e31362e36856c696e7578\",\"pid\":\"0xc1238aa5a212c5bb139681226b54c99d1dbe26ecf897bb1e4db4ff2b73c35df3\",\"k\":\"0x20\",\"challenge\":\"0x893d4e7e6ba79bbe8f3c10119de926135aa33b22b127dee3ab981520979697d1\",\"proof\":\"0xcef72cd2e076ed46ba98eb63e81738ca8f9c2c583435df2a18719f5f422bb1a25f5292b286a9a7e54473d12f50046ecae8b528b99013037718f29dbd84fc4af494b23b23d9a5ab38327a3c22956b46b228c7660c4e4d3b65a3fdd877426f217d0da85ad72f467e89cfa86c327a49fd33e751df4090d1ff35d87b557ba6a4431b4e255a9174e3c467da4ca0203956e678bc1193ff8eb17441e5b9c3f556347a74fee6815514c84b314a3d8ed1d24f1dc65b45c7a18398a836f8704470db5a0df9b3de1ddc370e088dba6b31e7203325a0bf566d58d7237d06d0c800e7bd3c32124f78e492b73c6d1bd51d8901087600bad3962dcdc66a4aedd25b9cc09f953b7f\",\"signed\":\"0xdaa89ad4ac893feb283ed6f327c1328bfcaa243614ba695c5fe4dfbaadaeb1804837d262334d6f01bfaf74e1719545ceb66227927c20f5ec9575f2ea796dbf5c01\",\"netCapacity\":\"0x7da114\",\"mixHash\":\"0x0000000000000000000000000000000000000000000000000000000000000000\",\"nonce\":\"0x0000000000000000\",\"baseFeePerGas\":\"0x7\",\"hash\":\"0xce46d910e7443ef59e852106685e8ed1ab122902fe28af742c54ee7030dcec7a\"}"))
	if err != nil {
		panic(err)
	}

	fmt.Println(parent.Hash())
	hash := CalcNextChallenge(&parent)
	fmt.Println(hash.Hex())
}

func (d *diffTest) UnmarshalJSON(b []byte) (err error) {
	var ext struct {
		ParentTimestamp    string
		ParentDifficulty   string
		CurrentTimestamp   string
		CurrentBlocknumber string
		CurrentDifficulty  string
	}
	if err := json.Unmarshal(b, &ext); err != nil {
		return err
	}

	d.ParentTimestamp = math.MustParseUint64(ext.ParentTimestamp)
	d.ParentDifficulty = math.MustParseBig256(ext.ParentDifficulty)
	d.CurrentTimestamp = math.MustParseUint64(ext.CurrentTimestamp)
	d.CurrentBlocknumber = math.MustParseBig256(ext.CurrentBlocknumber)
	d.CurrentDifficulty = math.MustParseBig256(ext.CurrentDifficulty)

	return nil
}

func TestCalcDifficulty(t *testing.T) {
	//file, err := os.Open(filepath.Join("..", "..", "tests", "testdata", "BasicTests", "difficulty.json"))
	//if err != nil {
	//	t.Skip(err)
	//}
	//defer file.Close()
	//
	//tests := make(map[string]diffTest)
	//err = json.NewDecoder(file).Decode(&tests)
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	////config := &params.ChainConfig{HomesteadBlock: big.NewInt(1150000)}
	//
	//for name, test := range tests {
	//	number := new(big.Int).Sub(test.CurrentBlocknumber, big.NewInt(1))
	//	diff := CalcDifficulty(nil, &types.Header{
	//		Number:     number,
	//		Time:       test.ParentTimestamp,
	//		Difficulty: test.ParentDifficulty,
	//	})
	//	if diff.Cmp(test.CurrentDifficulty) != 0 {
	//		t.Error(name, "failed. Expected", test.CurrentDifficulty, "and calculated", diff)
	//	}
	//}
}

func randSlice(min, max uint32) []byte {
	var b = make([]byte, 4)
	rand.Read(b)
	a := binary.LittleEndian.Uint32(b)
	size := min + a%(max-min)
	out := make([]byte, size)
	rand.Read(out)
	return out
}

func TestDifficultyCalculators(t *testing.T) {
	rand.Seed(2)
	for i := 0; i < 5000; i++ {
		// 1 to 300 seconds diff
		var timeDelta = uint64(1 + rand.Uint32()%3000)
		diffBig := big.NewInt(0).SetBytes(randSlice(2, 10))
		if diffBig.Cmp(params.MinimumDifficulty) < 0 {
			diffBig.Set(params.MinimumDifficulty)
		}
		//rand.Read(difficulty)
		header := &types.Header{
			Difficulty: diffBig,
			Number:     new(big.Int).SetUint64(rand.Uint64() % 50_000_000),
			Time:       rand.Uint64() - timeDelta,
		}
		if rand.Uint32()&1 == 0 {
			header.UncleHash = types.EmptyUncleHash
		}
		bombDelay := new(big.Int).SetUint64(rand.Uint64() % 50_000_000)
		for i, pair := range []struct {
			bigFn  func(time uint64, parent *types.Header) *big.Int
			u256Fn func(time uint64, parent *types.Header) *big.Int
		}{
			{FrontierDifficultyCalulator, CalcDifficultyFrontierU256},
			{HomesteadDifficultyCalulator, CalcDifficultyHomesteadU256},
			{DynamicDifficultyCalculator(bombDelay), MakeDifficultyCalculatorU256(bombDelay)},
		} {
			time := header.Time + timeDelta
			want := pair.bigFn(time, header)
			have := pair.u256Fn(time, header)
			if want.BitLen() > 256 {
				continue
			}
			if want.Cmp(have) != 0 {
				t.Fatalf("pair %d: want %x have %x\nparent.Number: %x\np.Time: %x\nc.Time: %x\nBombdelay: %v\n", i, want, have,
					header.Number, header.Time, time, bombDelay)
			}
		}
	}
}

func BenchmarkDifficultyCalculator(b *testing.B) {
	x1 := makeDifficultyCalculator(big.NewInt(1000000))
	x2 := MakeDifficultyCalculatorU256(big.NewInt(1000000))
	h := &types.Header{
		ParentHash: common.Hash{},
		UncleHash:  types.EmptyUncleHash,
		Difficulty: big.NewInt(0xffffff),
		Number:     big.NewInt(500000),
		Time:       1000000,
	}
	b.Run("big-frontier", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			calcDifficultyFrontier(1000014, h)
		}
	})
	b.Run("u256-frontier", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			CalcDifficultyFrontierU256(1000014, h)
		}
	})
	b.Run("big-homestead", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			calcDifficultyHomestead(1000014, h)
		}
	})
	b.Run("u256-homestead", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			CalcDifficultyHomesteadU256(1000014, h)
		}
	})
	b.Run("big-generic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x1(1000014, h)
		}
	})
	b.Run("u256-generic", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			x2(1000014, h)
		}
	})
}
