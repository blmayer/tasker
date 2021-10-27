package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deta/deta-go/service/base"
)

func public(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{
		Tasks: tasks,
		List:  List{Name: "tasks"},
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		logErr("template", pages.ExecuteTemplate(w, "index.html", p))
		return
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	logErr("template", pages.ExecuteTemplate(w, "task.html", tasks[4-id]))
}

func index(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		public(w, r)
		return
	}

	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		owner = user.Nick
	}

	switch len(parts) {
	case 2:
		serveList(w, r, user, owner)
	case 3:
		serveTask(w, r, user, owner)
	case 4:
		serveTaskAction(w, r, user, owner)
	}
}

func newList(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		public(w, r)
		return
	}

	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	if r.Method == http.MethodPost {
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

		newList := List{
			Name:        name,
			Owner:       user.Nick,
			CreateDate:  time.Now(),
			Permissions: ReadPermission | WritePermission,
		}
		if isPublic {
			newList.Permissions |= PublicPermission
		}

		up := map[string]interface{}{"Lists." + name: newList}
		err = usersDB.Update(user.Key, up)
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

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
	cookies := r.Cookies()
	if len(cookies) != 1 {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		up := map[string]interface{}{}
		for k := range r.Form {
			switch k {
			case "default_list":
				v := pol.Sanitize(r.Form.Get(k))
				up["Configs.DefaultList"] = v
				user.Configs.DefaultList = v
			}
		}

		err = usersDB.Update(user.Key, up)
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
	}
	logErr("template", pages.ExecuteTemplate(w, "profile.html", user))
}

func logout(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	token := Token{
		Expires: time.Now(),
	}

	err = usersDB.Update(user.Key, base.Updates{"Token": token})
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	cookies[0].Domain = domain
	cookies[0].Expires = time.Now()
	cookies[0].MaxAge = 0
	cookies[0].Path = "/"
	http.SetCookie(w, cookies[0])
	http.Redirect(w, r, "/", http.StatusFound)
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "login.html", tasks[2]))
		return
	}

	err := r.ParseForm()
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	user := User{
		Nick:       r.Form.Get("nick"),
		Pass:       r.Form.Get("password"),
		CreateDate: time.Now(),
	}
	if user.Nick == "" || user.Pass == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:    base.Query{{"Nick": user.Nick}},
		Dest: &dbUsers,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if len(dbUsers) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if len(user.Pass) == 4 && user.Pass == dbUsers[0].Pass {
		logErr("template", pages.ExecuteTemplate(w, "newpass.html", user))
		return
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	if user.Pass != dbUsers[0].Pass {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
		return
	}

	t := make([]byte, 128)
	_, err = rand.Read(t)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	for i, n := range t {
		t[i] = chars[int(n)%len(chars)]
	}
	token := Token{
		Value:   string(t),
		Expires: time.Now().Add(120 * time.Hour),
	}

	err = usersDB.Update(dbUsers[0].Key, base.Updates{"Token": token})
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    string(t),
			Domain:   domain,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  token.Expires,
		},
	)

	http.Redirect(w, r, "/"+dbUsers[0].Configs.DefaultList, http.StatusFound)
}

func newPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "login.html", tasks[3]))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	temp := r.Form.Get("temp")
	user := User{
		Nick: r.Form.Get("nick"),
		Pass: r.Form.Get("password"),
	}
	if user.Nick == "" || user.Pass == "" || temp == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:    base.Query{{"Nick": user.Nick}},
		Dest: &dbUsers,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if len(dbUsers) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}

	if temp != dbUsers[0].Pass {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "wrong temp password"))
		return
	}

	t := make([]byte, 128)
	_, err = rand.Read(t)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	for i, n := range t {
		t[i] = chars[int(n)%len(chars)]
	}
	token := Token{
		Value:   string(t),
		Expires: time.Now().Add(120 * time.Hour),
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	up := base.Updates{"Pass": user.Pass, "Token": token}
	err = usersDB.Update(dbUsers[0].Key, up)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    string(t),
			Domain:   domain,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  token.Expires,
		},
	)

	http.Redirect(w, r, "/", http.StatusFound)
}

func resetPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "reset.html", tasks[3]))
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	user := User{
		Email: r.Form.Get("email"),
		Nick:  r.Form.Get("nick"),
	}
	if user.Email == "" || user.Nick == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty email or nick"))
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:     base.Query{{"Nick": user.Nick}, {"Email": user.Email}},
		Dest:  &dbUsers,
		Limit: 1,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if len(dbUsers) == 0 {
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

	up := base.Updates{
		"Pass":  string(pass),
		"Token": nil,
	}
	err = usersDB.Update(dbUsers[0].Key, up)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
}

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		e := pages.ExecuteTemplate(w, "register.html", tasks[3])
		logErr("template", e)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	newUser := User{
		Email:      pol.Sanitize(r.Form.Get("email")),
		Nick:       pol.Sanitize(r.Form.Get("nick")),
		Pass:       r.Form.Get("password"),
		CreateDate: time.Now(),
		Lists: map[string]List{
			"tasks": {
				Name:        "tasks",
				Owner:       pol.Sanitize(r.Form.Get("nick")),
				Permissions: ReadPermission | WritePermission,
				CreateDate:  time.Now(),
			},
		},
		Configs: Config{
			DefaultList: "tasks",
		},
	}
	if newUser.Email == "" || newUser.Nick == "" || newUser.Pass == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:     base.Query{{"Nick": newUser.Nick}, {"Email": newUser.Email}},
		Dest:  &dbUsers,
		Limit: 1,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if len(dbUsers) > 0 {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "user already exists"))
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
	newUser.Token = Token{
		Value:   string(token),
		Expires: time.Now().Add(120 * time.Hour),
	}

	_, err = usersDB.Insert(newUser)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    string(token),
			Domain:   domain,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  newUser.Token.Expires,
		},
	)
	http.Redirect(w, r, "/tasks", http.StatusFound)
}

func getUserFromCookie(c http.Cookie) (User, error) {
	userSession := c.Value
	if userSession == "" {
		return User{}, fmt.Errorf("empty value")
	}

	users := []User{}
	query := base.FetchInput{
		Q:    base.Query{{"Token.Value": userSession}},
		Dest: &users,
	}
	_, err := usersDB.Fetch(&query)
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, fmt.Errorf("user not found")
	}

	if users[0].Token.Expires.Unix() < time.Now().Unix() {
		return User{}, fmt.Errorf("token expired")
	}
	return users[0], nil
}
