package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"os"
	"time"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"
)

type Task struct {
	Key         string `json:"key"`
	ID          int
	List        string
	ListOwner   string
	Title       string
	Status      string
	Summary     string
	Description string
	Creator     string
	Date        time.Time
}

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

func decrypt(key *rsa.PrivateKey, cypher string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return "", err
	}

	text, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		key,
		bytes, nil,
	)

	return string(text), err
}

func main() {
	oldKeyBytes, err := ioutil.ReadFile("old.key")
	if err != nil {
		panic("error reading file: " + err.Error())
	}

	oldEncKey, err := base64.StdEncoding.DecodeString(string(oldKeyBytes))
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	oldKey, err := x509.ParsePKCS1PrivateKey(oldEncKey)
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	newKeyBytes, err := ioutil.ReadFile("new.key")
	if err != nil {
		panic("error reading file: " + err.Error())
	}

	newEncKey, err := base64.StdEncoding.DecodeString(string(newKeyBytes))
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	newKey, err := x509.ParsePKCS1PrivateKey(newEncKey)
	if err != nil {
		panic("error parsing key: " + err.Error())
	}

	// Get tasks
	detaKey := os.Getenv("DETA_KEY")
	d, err := deta.New(deta.WithProjectKey(detaKey))
	if err != nil {
		panic("deta client " + err.Error())
	}

	tasksDB, err := base.New(d, "tasks")
	if err != nil {
		panic("deta base " + err.Error())
	}

	tasks := []Task{}
	input := base.FetchInput{
		Q:    nil,
		Dest: &tasks,
	}
	_, err = tasksDB.Fetch(&input)
	if err != nil {
		panic("fetch:" + err.Error())
	}

	for _, t := range tasks {
		println("updating", t.ListOwner, t.ID)
		oldDesc, err := decrypt(oldKey, t.Description)
		if err != nil {
			println("error decrypting:", err)
			continue
		}
		newDesc, err := encrypt(newKey, oldDesc)
		if err != nil {
			println("error encrypting:", err)
			continue
		}
		oldSumm, err := decrypt(oldKey, t.Summary)
		if err != nil {
			println("error decrypting:", err)
			continue
		}
		newSumm, err := encrypt(newKey, oldSumm)
		if err != nil {
			println("error encrypting:", err)
			continue
		}
		oldTitle, err := decrypt(oldKey, t.Summary)
		if err != nil {
			println("error decrypting:", err)
			continue
		}
		newTitle, err := encrypt(newKey, oldTitle)
		if err != nil {
			println("error encrypting:", err)
			continue
		}
		up := base.Updates{
			"Summary":     newSumm,
			"Title":       newTitle,
			"Description": newDesc,
		}
		err = tasksDB.Update(t.Key, up)
		if err != nil {
			println("error updating:", err)
		}
	}
}
