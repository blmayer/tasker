package main

import (
	"crypto/x509"
	"embed"
	"encoding/base64"
	"net/http"
	"os"
	"tasker/internal/types"
	"text/template"

	"github.com/gomarkdown/markdown"

	"github.com/microcosm-cc/bluemonday"
)

var (
	//go:embed www/*.html www/*.css www/*.gohtml www/*.txt www/*.ico
	content embed.FS

	root  string = "db.gob"
	pages *template.Template
	pol   = bluemonday.UGCPolicy()

	Users = map[string]*types.User{}
)

func logErr(prefix string, err error) {
	if err != nil {
		println(prefix, "error:", err.Error())
	}
}

func main() {
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-r", "--root":
			i++
			root = os.Args[i]
		default:
			println("error: wrong argument", os.Args[i], "\n")
			os.Exit(-1)
		}
	}

	err := readDB(root)
	if err != nil {
		panic("read DB: " + err.Error())
	}

	// Parse templates
	pages, err = template.New("").Funcs(
		template.FuncMap{
			"minus": func(a, b int) int {
				return a - b
			},
			"plus": func(a, b int) int {
				return a + b
			},
			"html": func(s string) string {
				return string(markdown.ToHTML([]byte(s), nil, nil))
			},
			"decrypt": func(t types.Task) types.Task {
				if err := t.Decrypt(key); err != nil {
					logErr("decrypt", err)
				}
				return t
			},
		},
	).ParseFS(content, "www/*.html", "www/*.css", "www/*.gohtml", "www/*.txt")
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

	encKey := os.Getenv("KEY")
	encText, err := base64.StdEncoding.DecodeString(encKey)
	logErr("encryption key", err)

	key, err = x509.ParsePKCS1PrivateKey(encText)
	logErr("encryption key parse", err)

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
