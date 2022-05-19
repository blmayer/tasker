package main

import (
	"crypto/rsa"
	"encoding/gob"
	"io"
	"os"
	"tasker/internal/types"
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."
)

type indexPayload struct {
	User  types.User
	List  types.List
	Tasks []*types.Task
	Page  int
}

var key *rsa.PrivateKey

var reservedNames = []string{
	"tasks", "new", "newlist", "profile", "reset", "login", "reset", "logout",
	"register", "newpass", "delete",
}

func isReservedName(name string) bool {
	for _, w := range reservedNames {
		if name == w {
			return true
		}
	}
	return false
}

func readDB(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDONLY, os.ModePerm)
	if err != nil {
		return err
	}
	err = gob.NewDecoder(file).Decode(&Users)
	file.Close()
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func writeDB(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	err = gob.NewEncoder(file).Encode(&Users)
	file.Close()
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}
