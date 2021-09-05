package main

import (
	"embed"
	"net/http"
	"os"
	"text/template"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"

	"github.com/gomarkdown/markdown"

	"github.com/microcosm-cc/bluemonday"
)

var (
	//go:embed *.html *.css *.gohtml
	content embed.FS

	pages   *template.Template
	usersDB *base.Base
	tasksDB *base.Base

	pol = bluemonday.UGCPolicy()

	domain = "tasker.blmayer.dev"
)

func logErr(err error) {
	if err != nil {
		println("ERROR:", err.Error())
	}
}

func main() {
	// Parse templates
	var err error
	pages, err = template.ParseFS(content, "*.html", "*.css", "*.gohtml")
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if os.Getenv("DEBUG") != "" {
		domain = "localhost"
	}

	detaKey := os.Getenv("DETA_KEY")
	d, err := deta.New(deta.WithProjectKey(detaKey))
	if err != nil {
		println("deta client error:", err.Error())
	}

	usersDB, err = base.New(d, "users")
	if err != nil {
		println("deta base error:", err.Error())
	}
	tasksDB, err = base.New(d, "tasks")
	if err != nil {
		println("deta base error:", err.Error())
	}

	for i, t := range defaultTasks {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		defaultTasks[i].Description = string(md)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/tasks/", tasks)
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
