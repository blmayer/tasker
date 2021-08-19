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

	"github.com/deta/deta-go/service/base"
)

const chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."

var defaultTasks = []Task{
	{
		ID:          4,
		Title:       "Learn to use this",
		Summary:     "Click on the + sign to create a new task.",
		Status:      "Active",
		Creator:     "blmayer",
		Date: time.Now().Add(-10 * time.Second),
	},
	{
		ID:          3,
		Title:       "Make your login",
		Summary:     "This task has a link for the login page.",
		Description: `I'm glad you made your registration. Here is the link: <a href="/login">login page</a>.`,
		Status:      "Blocked",
		Creator:     "blmayer",
		Date: time.Now().Add(-12 * time.Hour),
	},
	{
		ID:          2,
		Title:       "Create your user",
		Summary:     "The description of this task has a link to the registration page.",
		Description: `Here is the link: <a href="/register">registration page</a>. Welcome!`,
		Status:      "Active",
		Creator:     "blmayer",
		Date: time.Now().Add(-30 * time.Hour),
	},
	{
		ID:          1,
		Title:       "Find this website",
		Summary:     "Congratulations! You found this task manager.",
		Status:      "Done",
		Creator:     "blmayer",
		Date: time.Now().Add(-48 * time.Hour),
	},
}

type indexPayload struct {
	Tasks []Task
	User User
}

func index(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{
		Tasks: defaultTasks,
	}
	cookies := r.Cookies()
	if len(cookies) != 1 {
		pages.ExecuteTemplate(w, "index.html", p)
		return
	}
	userSession := cookies[0].Value
	if userSession == "" {
		pages.ExecuteTemplate(w, "index.html", p)
		return
	}

	users := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Token.Value": userSession}},
		Dest: &users,
	}
	_, err := usersDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(users) != 1 {
		// TODO: Show error page
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}
	p.User = users[0]
	if p.User.Token.Expires.Unix() < time.Now().Unix() {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}

	p.Tasks = []Task{}
	query = base.FetchInput{
		Q: base.Query{{"Creator": p.User.Nick}},
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
			ID: len(p.Tasks),
			Title:       r.Form.Get("title"),
			Summary:     r.Form.Get("summary"),
			Description: r.Form.Get("description"),
			Status:      r.Form.Get("status"),
			Creator:     p.User.Nick,
			Date: time.Now(),
		}
		p.Tasks = append(p.Tasks, t)

		go func(){
			_, err = tasksDB.Put(t)
			if err != nil {
				println("put error:",err)
			}
		}()
	}

	// Sort by time by default
	sort.SliceStable(p.Tasks, func(i, j int) bool {
		return p.Tasks[i].Date.Unix() > p.Tasks[j].Date.Unix()
	})

	pages.ExecuteTemplate(w, "index.html", p)
}

func tasks(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{Tasks: defaultTasks}
	parts := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", p)
		return
	}
	p.Tasks = []Task{}

	page := "task.html"
	if parts[1] == "edit" {
		page = "edit.html"
	}

	cookies := r.Cookies()
	if len(cookies) != 1 {
		pages.ExecuteTemplate(w, "task.html", defaultTasks[4-id])
		return
	}
	userSession := cookies[0].Value
	if userSession == "" {
		pages.ExecuteTemplate(w, "task.html", defaultTasks[4-id])
		return
	}

	users := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Token.Value": userSession}},
		Dest: &users,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(users) != 1 {
		// TODO: Show error page
		http.Error(w, "user conflict", http.StatusInternalServerError)
		return
	}
	p.User = users[0]
	if p.User.Token.Expires.Unix() < time.Now().Unix() {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}

	query = base.FetchInput{
		Q: base.Query{{"Creator": p.User.Nick, "ID": id}},
		Dest: &p.Tasks,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var t Task
	for _, i := range p.Tasks {
		if i.ID == id {
			t = i
			break
		}
	}
	pages.ExecuteTemplate(w, page, t)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	pages.ExecuteTemplate(w, "new.html", time.Now())
}

func editTask(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", err)
		return
	}

	newDate, err := time.Parse(time.RFC822Z, r.Form.Get("date"))
	if err != nil {
		newDate = time.Now()
	}
	id, err := strconv.Atoi(r.Form.Get("id"))
	t := Task{
		ID: id,
		Title:       r.Form.Get("title"),
		Summary:     r.Form.Get("summary"),
		Description: r.Form.Get("description"),
		Status:      r.Form.Get("status"),
		Creator:     r.Form.Get("creator"),
		Date: newDate,
	}
	go func(){
		_, err = tasksDB.Put(t)
		if err != nil {
			println("put error:",err)
		}
	}()
}

func logout(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	userSession := cookies[0].Value
	users := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Token.Value": userSession}},
		Dest: &users,
	}
	_, err := usersDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := Token{
		Expires: time.Now(),
	}

	err = usersDB.Update(users[0].Key, base.Updates{"Token": token})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cookies[0].Domain = "tasker.blmayer.dev"
	cookies[0].Expires = time.Now()
	cookies[0].MaxAge = 0
	cookies[0].Path = "/"
	http.SetCookie(w, cookies[0])
	http.Redirect(w, r, "/", http.StatusFound)
}

func login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		pages.ExecuteTemplate(w, "login.html", time.Now())
		return
	}

	err := r.ParseForm()
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", err)
		return
	}

	user := User{
		Nick: r.Form.Get("username"),
		Pass:  r.Form.Get("password"),
	}
	if user.Nick == "" || user.Pass == "" {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	dbUsers := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Nick": user.Nick, "Pass": user.Pass}},
		Dest: &dbUsers,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dbUsers) == 0 {
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	t := make([]byte, 128)
	_, err = rand.Read(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i, n := range t {
		t[i] = chars[int(n)%len(chars)]
	}
	token := Token{
		Value: string(t),
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

func register(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		pages.ExecuteTemplate(w, "register.html", time.Now())
		return
	}

	err := r.ParseForm()
	if err != nil {
		// TODO: Print error to an html page
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	newUser := User{
		Email: r.Form.Get("email"),
		Nick:  r.Form.Get("username"),
		Pass:  r.Form.Get("password"),
	}
	if newUser.Email == "" || newUser.Nick == "" || newUser.Pass == "" {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}


	dbUsers := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Nick": newUser.Nick}, {"Email": newUser.Email}},
		Dest: &dbUsers,
		Limit: 1,
	}
	_, err = usersDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dbUsers) > 0 {
		http.Error(w, "user already exists", http.StatusUnauthorized)
		return
	}

	sum := sha256.Sum256([]byte(newUser.Pass))
	newUser.Pass = fmt.Sprintf("%x", sum)

	token := make([]byte, 128)
	_, err = rand.Read(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i, n := range token {
		token[i] = chars[int(n)%len(chars)]
	}
	newUser.Token = Token{
		Value: string(token),
		Expires: time.Now().Add(120 * time.Hour),
	}

	_, err = usersDB.Insert(newUser)
	if err != nil {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    string(token),
			Domain:   "tasker.blmayer.dev",
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  newUser.Token.Expires,
		},
	)
	http.Redirect(w, r, "/", http.StatusFound)
}
