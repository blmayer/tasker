package main

import (
	"crypto/x509"
	"embed"
	"encoding/base64"
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
	// domain = "localhost"
)

func logErr(prefix string, err error) {
	if err != nil {
		println(prefix, "error:", err.Error())
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
	logErr("deta client", err)

	usersDB, err = base.New(d, "users")
	logErr("deta base", err)
	tasksDB, err = base.New(d, "tasks")
	logErr("deta base", err)

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
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		panic(err)
	}
}
