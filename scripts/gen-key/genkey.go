package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

func main() {
	priv, err := rsa.GenerateKey(rand.Reader, 512*64)
	if err != nil {
		fmt.Println(err)
		return
	}

	privText := x509.MarshalPKCS1PrivateKey(priv)

	encText := base64.StdEncoding.EncodeToString(privText)
	ioutil.WriteFile("test.key", []byte(encText), 0444)
}
