package ethclient

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	ethereum "github.com/gnc-project/galaxynetwork"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/crypto"
	"github.com/gnc-project/galaxynetwork/pocmine/transfertype"
	"github.com/gnc-project/galaxynetwork/rewardc"
	"math/big"
)


func StakingTransaction(client *Client,privateKeyString string,value *big.Int,periods uint64)(common.Hash,error){

	input := common.Hex2Bytes(transfertype.Staking+hex.EncodeToString([]byte(fmt.Sprintf("%d",periods))))
	perHex := hex.EncodeToString(input[7:])
	if _, ok := rewardc.ParsingStakingBase(perHex); !ok{
		return common.Hash{},transfertype.ErrInvalidPeriods
	}

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
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return common.Hash{},err
	}

	msg:=ethereum.CallMsg{
			From: fromAddress,
			To: &fromAddress,
			Gas: uint64(0),
			Value: value,
			Data: input,
		}

	gasPrice,_:=client.SuggestGasPrice(context.Background())
	gas,err:=client.EstimateGas(context.Background(),msg)
	if err != nil {
		return common.Hash{},err
	}
	tx := types.NewTransaction(nonce, *msg.To,msg.Value,gas,gasPrice, msg.Data)
	chainID,_:=client.ChainID(context.Background())
	//Sign transaction 
		signedTx, err:= types.SignTx(tx, types.NewEIP155Signer(chainID), fromPrivateKey)
		if err != nil {
			return common.Hash{},err
		}
	//send signatureTx 
		err = client.SendTransaction(context.Background(), signedTx)
		if err != nil {
			return common.Hash{},err
		}
		return signedTx.Hash(),nil
}