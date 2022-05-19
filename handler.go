package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"tasker/internal/permissions"
	"tasker/internal/types"
	"time"
)

func index(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	user := Users[nick]
	if !ok {
		user = Users["public"]
	}

	if user == nil || user.Pass != fmt.Sprintf("%x", sha256.Sum256([]byte(pass))) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Invalid credentials"))
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		owner = user.Nick
	}

	switch len(parts) {
	case 2:
		serveList(w, r, *user, owner)
	case 3:
		serveTask(w, r, *user, owner)
	case 4:
		serveTaskAction(w, r, *user, owner)
	}
}

func newList(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	user := Users[nick]
	if !ok {
		user = Users["public"]
	}

	if user == nil || user.Pass != fmt.Sprintf("%x", sha256.Sum256([]byte(pass))) {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if r.Method == http.MethodPost {
		if user.Permissions&permissions.CreateList == 0 {
			w.WriteHeader(http.StatusBadRequest)
			logErr("template", pages.ExecuteTemplate(w, "error.html", "No permission"))
			return
		}

		err := r.ParseForm()
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
		name := pol.Sanitize(r.Form.Get("name"))
		isPublic := r.Form.Get("public") == "public"

		if isReservedName(name) || strings.Contains(name, "/") {
			logErr("template", pages.ExecuteTemplate(w, "error.html", "Invalid name"))
			return
		}

		if _, has := user.Lists[name]; has {
			logErr("template", pages.ExecuteTemplate(w, "error.html", "Name unavailable"))
			return
		}

		newList := types.List{
			Name:        name,
			Owner:       user.Nick,
			CreateDate:  time.Now(),
			Permissions: permissions.ReadTask | permissions.WriteTask,
		}
		if isPublic {
			newList.Permissions |= permissions.PublicList
		}

		user.Lists[name] = &newList
		writeDB(root)
		logErr("template", pages.ExecuteTemplate(w, "profile.html", user))
		return
	}

	data := map[string]interface{}{
		"date":  time.Now(),
		"words": reservedNames,
	}
	logErr("template", pages.ExecuteTemplate(w, "newlist.html", data))
}

func profile(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	user := Users[nick]
	if !ok {
		user = Users["public"]
	}

	if user == nil || user.Pass != fmt.Sprintf("%x", sha256.Sum256([]byte(pass))) {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		for k := range r.Form {
			switch k {
			case "default_list":
				v := pol.Sanitize(r.Form.Get(k))
				user.Configs.DefaultList = v
			case "task_display_limit":
				lim, err := strconv.Atoi(r.Form.Get(k))
				if err != nil {
					logErr("update task limit", err)
					logErr("template", pages.ExecuteTemplate(w, "error.html", err))
					return
				}
				user.Configs.TaskDisplayLimit = lim
			}
		}
	}
	logErr("template", pages.ExecuteTemplate(w, "profile.html", user))
}

func logout(w http.ResponseWriter, r *http.Request) {
	_, _, ok := r.BasicAuth()
	if ok {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "logout.html", time.Now()))
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func deleteAccount(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "login.html", time.Now()))
		return
	}

	user := Users[nick]
	if user == nil || user.Pass != fmt.Sprintf("%x", sha256.Sum256([]byte(pass))) {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if user.Permissions&permissions.DeleteAccount == 0 {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "No permission"))
		return
	}

	for _, list := range user.Lists {
		list.Tasks = nil
	}
	delete(Users, user.Email)
	delete(Users, nick)
	writeDB(root)

	http.Redirect(w, r, "/", http.StatusUnauthorized)
}

func login(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "logout.html", time.Now()))
		return
	}
	if nick == "" || pass == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	sum := sha256.Sum256([]byte(pass))
	pass = fmt.Sprintf("%x", sum)

	user := Users[nick]
	if user == nil || user.Pass != pass {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User and pass are incorrect"))
		return
	}

	http.Redirect(w, r, "/"+user.Configs.DefaultList, http.StatusFound)
}

func newPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "login.html", time.Now()))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	temp := r.Form.Get("temp")
	req := types.User{
		Nick: r.Form.Get("nick"),
		Pass: r.Form.Get("password"),
	}
	if req.Nick == "" || req.Pass == "" || temp == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	user := Users[req.Nick]
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if temp != user.Pass {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong temp password"))
		return
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)
	writeDB(root)
	http.Redirect(w, r, "/tasks/3", http.StatusFound)
}

func resetPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "reset.html", time.Now()))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	req := types.User{
		Email: r.Form.Get("email"),
		Nick:  r.Form.Get("nick"),
	}
	if req.Email == "" || req.Nick == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty email or nick"))
		return
	}

	user := Users[req.Nick]
	if u := Users[req.Email]; u == nil || u != user {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	pass := make([]byte, 4)
	_, err = rand.Read(pass)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	for i, n := range pass {
		pass[i] = chars[int(n)%len(chars)]
	}

	go sendEmail(user.Email, user.Nick, string(pass))

	user.Pass = string(pass)
	writeDB(root)
	http.Redirect(w, r, "/login", http.StatusFound)
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		e := pages.ExecuteTemplate(w, "register.html", time.Now())
		logErr("template", e)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	newUser := types.User{
		Email:       pol.Sanitize(r.Form.Get("email")),
		Nick:        pol.Sanitize(r.Form.Get("nick")),
		Pass:        r.Form.Get("password"),
		CreateDate:  time.Now(),
		Permissions: permissions.CreateList | permissions.DeleteAccount | permissions.DeleteList,
		Lists: map[string]*types.List{
			"tasks": {
				Name:        "tasks",
				Owner:       pol.Sanitize(r.Form.Get("nick")),
				Permissions: permissions.ReadTask | permissions.WriteTask,
				CreateDate:  time.Now(),
			},
		},
		Configs: types.Config{
			DefaultList:      "tasks",
			TaskDisplayLimit: 20,
		},
	}
	if newUser.Email == "" || newUser.Nick == "" || newUser.Pass == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	if Users[newUser.Nick] != nil || Users[newUser.Email] != nil {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User already exists"))
		return
	}

	sum := sha256.Sum256([]byte(newUser.Pass))
	newUser.Pass = fmt.Sprintf("%x", sum)

	token := make([]byte, 128)
	_, err = rand.Read(token)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	for i, n := range token {
		token[i] = chars[int(n)%len(chars)]
	}

	Users[newUser.Email] = &newUser
	Users[newUser.Nick] = &newUser
	writeDB(root)

	http.Redirect(w, r, "/tasks/3", http.StatusFound)
}
