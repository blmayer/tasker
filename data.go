package main

import (
	"time"
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."
)

type Error int

type Task struct {
	Key         string `json:"key"`
	ID          int
	List        string
	Title       string
	Status      string
	Summary     string
	Description string
	Date        time.Time
	Due         *time.Time
}

type List struct {
	Name       string
	TaskNumber int
	CreateDate time.Time
}

type User struct {
	Key              string `json:"key"`
	CreateDate       time.Time
	TaskDisplayLimit int
	DefaultList      string
	DefaultSort      string
	DefaultFilter    string
	Lists            map[string]List
}

type indexPayload struct {
	User  User
	List  List
	Tasks []Task
	Page  int
}

//var key *rsa.PrivateKey

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
