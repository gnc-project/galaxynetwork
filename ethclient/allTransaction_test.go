package ethclient

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/hexutil"
	"math/big"
	"testing"
	"time"
)

var (
	pri		=	"24ccbb7a360ab4c728e1463db0a0d5b67c7923f91349520e6f89a10fb1e9933e"
	pri2	=	"0054f7c106f606f47dfedca3c3c15366a9a768661cf4dfc6c05b11550414681c"
	from   	= 	"0x461618Dc4480246eBAabb48169BC535e03e9f86E"
)

func TestNewClient(t *testing.T) {

}

func LinkGNC(ip string) *Client{
	for i :=0; i < 3; i++{
		GNCClient, err := Dial(ip)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second*100)
			continue
		}
		return GNCClient
	}
	return nil
}

func TestClient_BalanceAt(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	balance, err := client.BalanceAt(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(balance)
}

func TestGeneralTransaction(t *testing.T)  {
	client := LinkGNC("http://127.0.0.1:8545")
	tx,err := GeneralTransaction(client,pri,"0x55aB559Aff7B42DA26e80c271EfdA798BD799953",big.NewInt(0).Mul(big.NewInt(19000),big.NewInt(1e18)))
	if err != nil{
		t.Fatal(err)
	}
	fmt.Println("tx-->",tx.Hex())
}

func TestPledgeTransaction(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	hash := sha256.Sum256([]byte("4"))
	pidHex := hexutil.Encode(hash[:])

	tx,err := PledgeTransaction(client,pri,pidHex)
	if err!=nil{
		t.Fatal("------------->",err)
		return
	}
	fmt.Println("txHex--->",tx.Hex())
}
func TestDeletePidTransaction(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	hash := sha256.Sum256([]byte("4"))
	pidHex := hexutil.Encode(hash[:])

	tx,err := DeletePidTransaction(client,pri,pidHex)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash---->",tx.Hex())
}

func TestRedeemTransaction(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	tx,err := RedeemTransaction(client,pri)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash---->",tx.Hex())
}

func TestStakingTransaction(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	// periods ---> in keys of rewardc.StakingBase  90  180 360 1080
	tx,err := StakingTransaction(client,pri,new(big.Int).Mul(big.NewInt(10000),big.NewInt(1e18)),180)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("hash---->",tx.Hex())
}

func TestUnlockReward(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	tx,err := UnlockReward(client,pri)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(tx.Hex())
}

func TestClient_VerifyPid(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	hash := sha256.Sum256([]byte("4"))
	pidHex := hexutil.Encode(hash[:])

	b, err := client.VerifyPid(context.Background(),common.HexToAddress(from),pidHex,nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("verifyPId",b,"from",from)
}

func TestClient_GetPledgeAmount(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	hash := sha256.Sum256([]byte("4"))
	pidHex := hexutil.Encode(hash[:])

	amount,err := client.GetPledgeAmount(context.Background(),common.HexToAddress(from),pidHex,nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("pledge amount",amount)
}

func TestClient_GetNeedPledgeAmount(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	amount,err := client.GetNeedPledgeAmount(context.Background(),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("need pledge amount",amount)
}

func TestClient_GetAllPledgeAmount(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	amount, err := client.GetAllPledgeAmount(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("all pledged amount",amount)
}

func TestClient_GetTotalCapacity(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	capacity,err := client.GetTotalCapacity(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("totalCapacity",capacity)
}

func TestClient_GetStakingWeightByAddr(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	stakingWeight,err := client.GetStakingWeightByAddr(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(stakingWeight)
}

func TestClient_GetRewardStakingList(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	stakingList,err := client.GetRewardStakingList(context.Background(),nil)
	if err != nil {
		t.Fatal(err)
	}
	for _,v := range stakingList {
		fmt.Println(v)
	}
}

func TestClient_GetRedeemAmount(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	redeemAmount,err := client.GetRedeemAmount(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(redeemAmount)
}

func TestClient_GetAmountUnlocked(t *testing.T) {
	client := LinkGNC("http://127.0.0.1:8545")
	amountUnlocked,err := client.GetAmountUnlocked(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(amountUnlocked)
}

func TestClient_GetTotalLockedAmount(t *testing.T)  {
	client := LinkGNC("http://127.0.0.1:8545")
	amountUnlocked,err := client.GetTotalLockedAmount(context.Background(),common.HexToAddress(from),nil)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(amountUnlocked)
}


