package ethclient


import (
	"errors"

	ethereum "github.com/gnc-project/galaxynetwork"
	"github.com/gnc-project/galaxynetwork/common"
	"github.com/gnc-project/galaxynetwork/core/types"
	"github.com/gnc-project/galaxynetwork/crypto"

	"context"
	"crypto/ecdsa"
	"math/big"
)


func GeneralTransaction(client *Client,privateKeyString string,toString string,value *big.Int)(common.Hash,error){
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
	toAddress:=common.HexToAddress(toString)

	msg:=ethereum.CallMsg{
			From:fromAddress,
			To:&toAddress,
			Gas: uint64(0),
			Value: value,
			Data:[]byte(""),
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