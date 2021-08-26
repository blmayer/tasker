package main

import (
	"embed"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"

	"github.com/gomarkdown/markdown"

	"github.com/microcosm-cc/bluemonday"
)

var (
	//go:embed *.html *.css
	content embed.FS

	pages *template.Template
	usersDB *base.Base
	tasksDB *base.Base

	pol = bluemonday.UGCPolicy()
)

type Token struct {
	Value string
	Expires time.Time
}

type User struct {
	Key string `json:"key"`
	Nick string
	Email string
	Pass string
	Token Token
	CreateDate time.Time
	Configs []interface{}
}

type Task struct {
	Key string `json:"key"`
	ID          int
	Title       string
	Status      string
	Summary     string
	Description string
	Creator     string
	Date time.Time
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

	for i, t := range defaultTasks {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		defaultTasks[i].Description = string(md)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/tasks/", tasks)
	http.HandleFunc("/edit/", tasks)
	http.HandleFunc("/new", newTask)
	http.HandleFunc("/edit", editTask)

	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/profile", profile)
	http.HandleFunc("/reset", resetPass)
	http.HandleFunc("/newpass", newPass)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
