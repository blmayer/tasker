package types

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

type Permissions int

type Error int

type Task struct {
	ID          int
	List        string
	ListOwner   string
	Title       string
	Status      string
	Summary     string
	Description string
	Creator     string
	Date        time.Time
	Due         *time.Time
}

type List struct {
	Name        string
	Permissions Permissions
	Owner       string
	CreateDate  time.Time
	Tasks       []*Task
}

type Config struct {
	TaskDisplayLimit int
	DefaultList      string
	DefaultSort      string
	DefaultFilter    string
}

type User struct {
	Nick        string
	Email       string
	Pass        string
	CreateDate  time.Time
	Configs     Config
	Lists       map[string]*List
	Permissions Permissions
}

func (t *Task) Encrypt(key rsa.PrivateKey) error {
	wg := sync.WaitGroup{}
	var err error
	wg.Add(3)

	go func() {
		var cypher []byte
		cypher, err = rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			&key.PublicKey,
			[]byte(t.Title), nil,
		)
		t.Title = base64.StdEncoding.EncodeToString(cypher)
		wg.Done()
	}()
	go func() {
		var cypher []byte
		cypher, err = rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			&key.PublicKey,
			[]byte(t.Description), nil,
		)
		t.Description = base64.StdEncoding.EncodeToString(cypher)
		wg.Done()
	}()
	go func() {
		var cypher []byte
		cypher, err = rsa.EncryptOAEP(
			sha256.New(),
			rand.Reader,
			&key.PublicKey,
			[]byte(t.Summary), nil,
		)
		t.Summary = base64.StdEncoding.EncodeToString(cypher)
		wg.Done()
	}()
	wg.Wait()

	return err
}

func (t *Task) Decrypt(key *rsa.PrivateKey) error {
	wg := sync.WaitGroup{}
	var err error
	wg.Add(3)

	go func() {
		bytes, _ := base64.StdEncoding.DecodeString(t.Title)
		var text []byte
		text, err = rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			key,
			bytes, nil,
		)
		t.Title = string(text)
		wg.Done()
	}()
	go func() {
		bytes, _ := base64.StdEncoding.DecodeString(t.Description)
		var text []byte
		text, err = rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			key,
			bytes, nil,
		)
		t.Description = string(text)
		wg.Done()
	}()
	go func() {
		bytes, _ := base64.StdEncoding.DecodeString(t.Summary)
		var text []byte
		text, err = rsa.DecryptOAEP(
			sha256.New(),
			rand.Reader,
			key,
			bytes, nil,
		)
		t.Summary = string(text)
		wg.Done()
	}()
	wg.Wait()
	return err
}
