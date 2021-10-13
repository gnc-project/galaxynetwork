package ethclient

import (
	"errors"
	"fmt"
	ethereum "github.com/gnc-project/galaxynetwork"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/common/pidaddress"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/crypto"
	"github.com/gnc-project/galaxynetwork/pocmine/transfertype"
	"strings"
	"time"

	"context"
	"crypto/ecdsa"
)

func PledgeTransaction(client *Client,privateKeyString string,pidHex string)(common.Hash,error){
	fromPrivateKey, err := crypto.HexToECDSA(privateKeyString)
	if err != nil {
		return common.Hash{},err
	}
	publicKey := fromPrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Hash{},errors.New("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	pid := common.HexToHash(pidHex)
	toAddress := pidaddress.PIDAddress(fromAddress,pid[:])

	pledgeValue,err := client.GetNeedPledgeAmount(context.Background())
	if err != nil {
		return common.Hash{},err
	}

	msg:=ethereum.CallMsg{
			From:fromAddress,
			To:&toAddress,
			Gas: uint64(0),
			Value: pledgeValue,
			Data: common.Hex2Bytes(transfertype.Pledge),
		}

	gas,err:=client.EstimateGas(context.Background(),msg)
	if err != nil {
		return common.Hash{},fmt.Errorf("EstimateGas err=%v",err)
	}
	gasPrice,err:=client.SuggestGasPrice(context.Background())
	if err != nil {
		return common.Hash{},err
	}

	chainID,err:=client.ChainID(context.Background())
	if err != nil {
		return common.Hash{},err
	}

	for  {
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			return common.Hash{},err
		}
		fmt.Printf("address=%s nonce=%d\n",fromAddress.Hex(),nonce)
		tx := types.NewTransaction(nonce, *msg.To,msg.Value,gas,gasPrice, msg.Data)

		//Sign transaction
		signedTx, err:= types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivateKey)
		if err != nil {
			return common.Hash{},err
		}

		//send signatureTx
		if err = client.SendTransaction(context.Background(), signedTx);err != nil {
			if strings.Contains(err.Error(),"nonce") {
				time.Sleep(2 *time.Second)
				continue
			}else {
				return common.Hash{},err
			}
		}

		return signedTx.Hash(),nil
	}

}