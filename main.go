package main

import (
	"embed"
	"net/http"
	"os"
	"text/template"

	"github.com/deta/deta-go/deta"
	"github.com/deta/deta-go/service/base"

	"github.com/microcosm-cc/bluemonday"
)

var (
	//go:embed *.html *.css *.gohtml *.txt *.ico
	content embed.FS

	pages *template.Template
	db    *base.Base

	pol = bluemonday.UGCPolicy()
)

func logErr(prefix string, err error) {
	if err != nil {
		println(prefix, "error:", err.Error())
	}
}

func main() {
	// Parse templates
	var err error
	pages, err = template.New("").Funcs(
		template.FuncMap{
			"minus": func(a, b int) int {
				return a - b
			},
			"plus": func(a, b int) int {
				return a + b
			},
		},
	).ParseFS(content, "*.html", "*.css", "*.gohtml", "*.txt")
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	d, err := deta.New()
	logErr("deta client", err)

	db, err = base.New(d, "tasks")
	logErr("deta base", err)

	http.HandleFunc("/", index)
	http.HandleFunc("/newlist", newList)
	http.HandleFunc("/new/", newTask)
	http.HandleFunc("/profile", profile)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
