package main

import (
	"embed"
	"net/http"
	"os"
	"text/template"
	"time"
)

var (
	//go:embed *.html *.css
	content embed.FS

	pages *template.Template
)

type task struct {
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

	http.HandleFunc("/", index)
	http.HandleFunc("/new", newTask)
	http.HandleFunc("/login", login)
	http.HandleFunc("/register", register)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
