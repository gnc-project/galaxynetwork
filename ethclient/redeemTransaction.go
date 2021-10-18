package ethclient

import (
	"errors"
	"github.com/gnc-project/galaxynetwork/pocmine/transfertype"
	"math/big"

	"context"
	"crypto/ecdsa"
	ethereum "github.com/gnc-project/galaxynetwork"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/crypto"
)


func RedeemTransaction(client *Client,privateKeyString string)(common.Hash,error){
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
			From:fromAddress,
			To:&fromAddress,
			Gas: uint64(0),
			Value: big.NewInt(0),
			Data: common.Hex2Bytes(transfertype.Redeem),
		}
	gas,err:=client.EstimateGas(context.Background(),msg)
	if err != nil {
		return common.Hash{},err

	}
	gasPrice,_:=client.SuggestGasPrice(context.Background())
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