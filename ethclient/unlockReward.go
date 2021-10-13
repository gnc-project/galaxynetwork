package ethclient

import (
	"context"
	"crypto/ecdsa"
	"errors"
	ethereum "github.com/gnc-project/galaxynetwork"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/crypto"
	"github.com/gnc-project/galaxynetwork/pocmine/transfertype"
)

func UnlockReward(client *Client,privateKeyString string)(common.Hash,error){
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

	value,err:=client.GetAmountUnlocked(context.Background(),fromAddress,nil)
	if err != nil {
		return common.Hash{},err
	}
	msg:=ethereum.CallMsg{
			From:fromAddress,
			To:&fromAddress,
			Gas: uint64(0),
			Value:value,
			Data: common.Hex2Bytes(transfertype.UnlockReward),
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