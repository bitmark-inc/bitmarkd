package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"

	crypto "github.com/libp2p/go-libp2p-crypto"
)

var zmqPublicKey = "PUBLIC:00fc907720d1ede30f247733d391af01fccd71398285554163edc20904096453"
var zmqPrivateKey = "PRIVATE:0d323c235e1bd5ef22e5ab7916c515717cb0199c1d39e04d31e0d535bb0ce2fd"
var libp2pPrivate = "35fe830273e291ca0f1a91139759e94f09275fb10749525ade7d412e296c76f627cf97d064f3c9cc833b6b7e5820cf6ca1be271632a74ae7f29f0e3f3cf921d0"
var libp2pPublic = "27cf97d064f3c9cc833b6b7e5820cf6ca1be271632a74ae7f29f0e3f3cf921d0"
var Ed25519SecretKey = "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"
var Ed25519PublicKey = "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"
var Ed25519MessageSignature = "e5564300c360ac729086e2cc806e828a 84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b"

func TestKey(t *testing.T) {
	showLibp2pKey()
}

func showZmqKey() {
	prvByteKey, err := ReadPrivateKey(zmqPrivateKey)
	if err != nil {
		panic(err)
	}
	fmt.Println("Private Key Encoded:", hex.EncodeToString(prvByteKey), " Len=", len(prvByteKey))
	fmt.Printf("%v\n", prvByteKey)
	pubByteKey, err := ReadPublicKey(zmqPublicKey)
	if err != nil {
		panic(err)
	}
	fmt.Println("Public Key Encoded:", hex.EncodeToString(pubByteKey), " Len=", len(pubByteKey))
	fmt.Printf("%v\n", pubByteKey)
}

func showLibp2pKey() {
	fmt.Println("--------------- showLibp2pKey  ---------------")
	prvKey, pubKey, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Println("--------------- showLibp2pKey PrvKey---------------")
	prvKeyByte, err := prvKey.Bytes()
	fmt.Println("Private Key Encoded:", hex.EncodeToString(prvKeyByte), " Len=", len(prvKeyByte), " first4byte:", prvKeyByte[:4])
	fmt.Printf("%v\n", prvKeyByte)
	prvKeyRawByte, err := prvKey.Raw()
	if err != nil {
		panic(err)
	}
	fmt.Println("Private  prvKeyRawByte  Encoded:", hex.EncodeToString(prvKeyRawByte), " Len=", len(prvKeyRawByte))
	fmt.Printf("%v\n", prvKeyByte)
	fmt.Println("--------------- showLibp2pKey Pubkey---------------")
	pubKeyByte, err := pubKey.Bytes()
	fmt.Println("Private Key Encoded:", hex.EncodeToString(pubKeyByte), " Len=", len(pubKeyByte))
	fmt.Printf("%v\n", pubKeyByte)
	pubKeyRawByte, err := pubKey.Raw()
	if err != nil {
		panic(err)
	}
	fmt.Println("Private  pubKeyRawByte  Encoded:", hex.EncodeToString(pubKeyRawByte), " Len=", len(pubKeyRawByte))
	fmt.Printf("%v\n", pubKeyRawByte)
}
