package main

import (
	"crypto/x509"
	"embed"
	"encoding/base64"
	"net/http"
	"os"
	"text/template"

	"github.com/graphlayer/db/client"

	"github.com/gomarkdown/markdown"

	"github.com/microcosm-cc/bluemonday"
)

var (
	//go:embed *.html *.css *.gohtml *.txt *.ico
	content embed.FS

	pages   *template.Template
	db client.DB

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

	if os.Getenv("DEBUG") != "" {
		println("running in debug mode")
	}

	db, err = client.New("http://localhost:8080", "blmayer", "X916482dXhV")
	if err != nil {
		panic(err)
	}

	encKey := os.Getenv("KEY")
	encText, err := base64.StdEncoding.DecodeString(encKey)
	logErr("encryption key", err)

	key, err = x509.ParsePKCS1PrivateKey(encText)
	logErr("encryption key parse", err)

	for i, t := range tasks {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		tasks[i].Description = string(md)
	}

	http.HandleFunc("/", index)
	http.HandleFunc("/newlist", newList)
	http.HandleFunc("/new/", newTask)

	http.HandleFunc("/register", register)
	http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/profile", profile)
	http.HandleFunc("/reset", resetPass)
	http.HandleFunc("/newpass", newPass)
	http.HandleFunc("/delete", deleteAccount)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
