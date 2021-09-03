package pocmine

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Owner struct{
	PrivateKey *ecdsa.PrivateKey
	Coinbase common.Address
}

var OwnerCoin *Owner

func SetOwner(pri string) error{
	privateKey, err := crypto.HexToECDSA(pri)
	if err != nil {
		return err
	}
	OwnerCoin = &Owner{
		PrivateKey: privateKey,
	}
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("error casting public key to ECDSA")
	}
	OwnerCoin.Coinbase = crypto.PubkeyToAddress(*publicKeyECDSA)
	return nil
}

func Sign(header *types.Header) (sig []byte, err error) {
	if OwnerCoin == nil {
		return nil,errors.New("owner is nil")
	}
	if OwnerCoin.PrivateKey == nil {
		return nil,errors.New("privateKey is nil")
	}
	return crypto.Sign(ShaInput(header),OwnerCoin.PrivateKey)
}

func Verify(pubkey []byte, header *types.Header, signature []byte) bool  {

	//recoveredPubkey, err := crypto.SigToPub(ShaInput(header), header.Signed)
	//if err != nil || recoveredPubkey == nil {
	//	log.Error("verify pubkey","err",err)
	//	return false
	//}
	//
	//if crypto.PubkeyToAddress(*recoveredPubkey) != header.Coinbase {
	//	log.Error("verify coinbase pub","err","The signature is not from coinbase")
	//	return false
	//}

	return crypto.VerifySignature(pubkey,ShaInput(header),signature)
}

func ShaInput(header *types.Header) []byte {

	sh := sha256.New()
	sh.Write(header.Root[:])
	sh.Write(header.Pid[:])
	sh.Write(header.Proof)
	sh.Write(header.Coinbase[:])

	//sh := sha256.New()
	//sh.Write(header.Root[:])
	//sh.Write(header.Pid[:])
	//sh.Write(header.Proof)
	//sh.Write(header.Coinbase[:])
	//sh.Write(header.ParentHash[:])
	//sh.Write(header.Number.Bytes())
	return sh.Sum(nil)
}
