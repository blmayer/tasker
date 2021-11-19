package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	mrand "math/rand"
)

const chars = "-[]#qwertyuiopQWERTYUIOPasdfghjklASDFGHJKLzxcvbnmZXCVBNM \r\n\t"

func encrypt(key *rsa.PrivateKey, text string) (string, error) {
	cypher, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		&key.PublicKey,
		[]byte(text), nil,
	)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(cypher), nil
}

func main() {
	keyBytes, err := ioutil.ReadFile("test.key")
	if err != nil {
		panic("error reading file: " + err.Error())
	}

	encKey, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	key, err := x509.ParsePKCS1PrivateKey(encKey)
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	// Create random text
	for i := 1; ; i++ {
		var text string
		for k := 0; k < i; k++ {
			text += string(chars[mrand.Intn(len(chars))])
		}

		_, err := encrypt(key, text)
		if err != nil {
			println("\nencryption failed", err)
			break
		}

		print("\rlength: ", i)
	}
}
