package main

import (
	"embed"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"
)

var (
	//go:embed *.html *.css
	content embed.FS

	pages *template.Template
	usersDB *base.Base
	tasksDB *base.Base
)

type User struct {
	Nick string
	Email string
	Pass string
	Key string `json:"key"`
	Configs []interface{}
}

type Task struct {
	ID          int
	Title       string
	Status      string
	Summary     string
	Description string
	Creator     string
	DateCreated time.Time
}

func main() {
	// Parse templates
	var err error
	pages, err = template.ParseFS(content, "*.html", "*.css")
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	detaKey := os.Getenv("DETA_KEY")
	d, err := deta.New(deta.WithProjectKey(detaKey))
	if err != nil {
		println("deta client error:", err)
	}

	usersDB, err = base.New(d, "users")
	if err != nil {
		println("deta base error:", err)
	}
	tasksDB, err = base.New(d, "tasks")
	if err != nil {
		println("deta base error:", err)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/tasks/", task)
	http.HandleFunc("/new", newTask)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
