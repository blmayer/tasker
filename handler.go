package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"

	"github.com/deta/deta-go/service/base"
)

func index(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{
		Tasks: defaultTasks,
	}
	cookies := r.Cookies()
	if len(cookies) != 1 {
		pages.ExecuteTemplate(w, "index.html", p)
		return
	}
	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	p.User = user

	p.Tasks = []Task{}
	query := base.FetchInput{
		Q:    base.Query{{"Creator": p.User.Nick}},
		Dest: &p.Tasks,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			pages.ExecuteTemplate(w, "index.html", err)
			return
		}

		t := Task{
			ID:          len(p.Tasks),
			Title:       pol.Sanitize(r.Form.Get("title")),
			Summary:     pol.Sanitize(r.Form.Get("summary")),
			Description: pol.Sanitize(r.Form.Get("description")),
			Status:      pol.Sanitize(r.Form.Get("status")),
			Creator:     p.User.Nick,
			Date:        time.Now(),
		}
		p.Tasks = append(p.Tasks, t)

		go func() {
			_, err = tasksDB.Put(t)
			if err != nil {
				println("put error:", err)
			}
		}()
	}

	// Sort by time by default
	sort.SliceStable(p.Tasks, func(i, j int) bool {
		return p.Tasks[i].Date.Unix() > p.Tasks[j].Date.Unix()
	})

	for i, t := range p.Tasks {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		p.Tasks[i].Description = string(md)
	}

	pages.ExecuteTemplate(w, "index.html", p)
}

func tasks(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{Tasks: defaultTasks}
	parts := strings.Split(r.URL.Path, "/")

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", p))
		return
	}
	p.Tasks = []Task{}

	cookies := r.Cookies()
	if len(cookies) != 1 {
		logErr("template", pages.ExecuteTemplate(w, "task.html", defaultTasks[4-id]))
		return
	}
	p.User, err = getUserFromCookie(*cookies[0])
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	query := base.FetchInput{
		Q:    base.Query{{"Creator": p.User.Nick, "ID": id}},
		Dest: &p.Tasks,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := p.Tasks[0]
	page := "task.html"
	if len(parts) == 4 {
		switch parts[3] {
		case "edit":
			page = "edit.html"
		case "delete":
			err = tasksDB.Delete(t.Key)
			if err != nil {
				// TODO: Show error page
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	if page == "task.html" {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		t.Description = string(md)
	}
	logErr("template", pages.ExecuteTemplate(w, page, t))
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
	logErr("template", pages.ExecuteTemplate(w, "profile.html", user))
}

func newTask(w http.ResponseWriter, r *http.Request) {
	logErr("template", pages.ExecuteTemplate(w, "new.html", Task{Date: time.Now()}))
}

func editTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", err))
		return
	}

	newDate, err := time.Parse(time.RFC3339, r.Form.Get("date"))
	if err != nil {
		newDate = time.Now()
	}
	id, err := strconv.Atoi(r.Form.Get("id"))
	t := Task{
		ID:          id,
		Key:         pol.Sanitize(r.Form.Get("key")),
		Title:       pol.Sanitize(r.Form.Get("title")),
		Summary:     pol.Sanitize(r.Form.Get("summary")),
		Description: pol.Sanitize(r.Form.Get("description")),
		Status:      pol.Sanitize(r.Form.Get("status")),
		Creator:     pol.Sanitize(r.Form.Get("creator")),
		Date:        newDate,
	}
	go func() {
		_, err = tasksDB.Put(t)
		if err != nil {
			println("put error:", err)
		}
	}()
	http.Redirect(w, r, "/", http.StatusFound)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    string(t),
			Domain:   "tasker.blmayer.dev",
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  token.Expires,
		},
	)

	http.Redirect(w, r, "/", http.StatusFound)
}

func newPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "login.html", Task{Date: time.Now()}))
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
			Domain:   "tasker.blmayer.dev",
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  token.Expires,
		},
	)

	http.Redirect(w, r, "/", http.StatusFound)
}

func resetPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		logErr("template", pages.ExecuteTemplate(w, "reset.html", Task{Date: time.Now()}))
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
		e := pages.ExecuteTemplate(w, "register.html", Task{Date: time.Now()})
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
	http.Redirect(w, r, "/", http.StatusFound)
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
