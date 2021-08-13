package main

import (
	"crypto/rand"
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
		DateCreated: time.Now().Add(-10 * time.Second),
	},
	{
		ID:          3,
		Title:       "Make your login",
		Summary:     "This task has a link for the login page.",
		Description: `I'm glad you made your registration. Here is the link: <a href="/login">login page</a>.`,
		Status:      "Blocked",
		Creator:     "blmayer",
		DateCreated: time.Now().Add(-12 * time.Hour),
	},
	{
		ID:          2,
		Title:       "Create your user",
		Summary:     "The description of this task has a link to the registration page.",
		Description: `Here is the link: <a href="/register">registration page</a>. Welcome!`,
		Status:      "Active",
		Creator:     "blmayer",
		DateCreated: time.Now().Add(-30 * time.Hour),
	},
	{
		ID:          1,
		Title:       "Find this website",
		Summary:     "Congratulations! You found this task manager.",
		Status:      "Done",
		Creator:     "blmayer",
		DateCreated: time.Now().Add(-48 * time.Hour),
	},
}

func index(w http.ResponseWriter, r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) != 1 {
		pages.ExecuteTemplate(w, "index.html", defaultTasks)
		return
	}
	userSession := cookies[0].Value
	if userSession == "" {
		pages.ExecuteTemplate(w, "index.html", defaultTasks)
		return
	}
	if cookies[0].Expires.Unix() < time.Now().Unix() {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}

	user := User{}
	tasks := make([]Task, 0)
	err := usersDB.Get(userSession, &user)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := base.FetchInput{
		Q: base.Query{{"Creator": user.Nick}},
		Dest: &tasks,
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
			ID: len(tasks),
			Title:       r.Form.Get("title"),
			Summary:     r.Form.Get("summary"),
			Description: r.Form.Get("description"),
			Status:      r.Form.Get("status"),
			Creator:     user.Nick,
			DateCreated: time.Now(),
		}
		tasks = append(tasks, t)

		go func(){
			_, err = tasksDB.Put(t)
			if err != nil {
				println("put error:",err)
			}
		}()
	}

	// Sort by time by default
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].DateCreated.Unix() > tasks[j].DateCreated.Unix()
	})

	pages.ExecuteTemplate(w, "index.html", tasks)
}

func task(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	id, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", defaultTasks)
		return
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
	if cookies[0].Expires.Unix() < time.Now().Unix() {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}

	user := User{}
	tasks := make([]Task, 0)
	err = usersDB.Get(userSession, &user)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := base.FetchInput{
		Q: base.Query{{"Creator": user.Nick, "ID": id}},
		Dest: &tasks,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var t Task
	for _, i := range tasks {
		if i.ID == id {
			t = i
			break
		}
	}
	pages.ExecuteTemplate(w, "task.html", t)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	pages.ExecuteTemplate(w, "new.html", time.Now())
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

	dbUsers := []User{}
	query := base.FetchInput{
		Q: base.Query{{"Nick": user.Nick, "Pass": user.Pass}},
		Dest: &dbUsers,
		Limit: 1,
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
	key := dbUsers[0].Key

	t := make([]byte, 128)
	_, err = rand.Read(t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i, n := range t {
		t[i] = chars[int(n)%len(chars)]
	}

	err = usersDB.Update(key, base.Updates{"Token": string(t)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     "token",
			Value:    key,
			Domain:   "tasker.blmayer.dev",
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			Expires:  time.Now().Add(120 * time.Hour),
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

	token := make([]byte, 128)
	_, err = rand.Read(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for i, n := range token {
		token[i] = chars[int(n)%len(chars)]
	}
	newUser.Key = string(token)

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
			Expires:  time.Now().Add(120 * time.Hour),
		},
	)
	http.Redirect(w, r, "/", http.StatusFound)
}
