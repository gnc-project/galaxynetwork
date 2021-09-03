package pocmine

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"log"
	"math/big"
	"testing"
)


func TestSign(t *testing.T) {
	header := &types.Header{
		Number: big.NewInt(1),
	}
	privateKey, err := crypto.HexToECDSA("2115999d16eb0fdfef3802b269807ff70422a2658f37ab3475469dcee1dd32e4")
	if err != nil {
		panic(err)
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		panic(errors.New(fmt.Sprintf("cannot assert type: publicKey is not of type *ecdsa.PublicKey err=%s",err.Error())))
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	header.Coinbase = address

	OwnerCoin = &Owner{
		PrivateKey: privateKey,
	}
	sig,err := Sign(header)
	header.Signed = sig
	fmt.Println("sig---->",hex.EncodeToString(sig))
	if err != nil {
		log.Fatalln(err)
	}

	pub,err := crypto.Ecrecover(ShaInput(header),sig)
	fmt.Println(len(pub))
	signatureNoRecoverID := sig[:len(sig)-1]
	if b := Verify(pub,header,signatureNoRecoverID);!b {
		log.Fatalln(fmt.Errorf("err--->b %v",b))
	}else {
		log.Println("yes")
	}
}
