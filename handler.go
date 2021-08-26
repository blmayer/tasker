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

const chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."

var defaultTasks = []Task{
	{
		ID:      4,
		Title:   "Learn to use this",
		Summary: "This task has a tutorial.",
		Description: `# Welcome!
Thank you for taking your time using this, I've done it for
my own needs but decided to open it as a web service. So
feel free to email me or use GitHub for droping a message.

## First steps
Follow the tasks on the main page, they will guide you to
create a user and log in whenever you need.

## Creating tasks
After logging in you can create your own tasks, to do so
click on the + sign to create a new task. Then fill in
the fields, **only the description is optional**.

### Markdown support
Yes, you can use markdown on the **description** field,
we support a nice set of extensions, just some:

Tables like

Name    | Age
--------|------
Bob     | 27
Alice   | 23

Can be entered by typing:
` + "```" + `
Name    | Age
--------|------
Bob     | 27
Alice   | 23

` + "```" + `

~~striked~~ through text using tildes: ` + "`~~`" + `.

### Updating tasks
There is a small link, edit task, below the date when you are seeing a task.

### Notes
This site doesn't use JavaScript, I try to make it as simple as possible,
so authentication uses cookies, but with a strict security, to create a
session.

***

See https://daringfireball.net/projects/markdown/ to learn more.
`,
		Status:  "Active",
		Creator: "blmayer",
		Date:    time.Now().Add(-10 * time.Second),
	},
	{
		ID:      3,
		Title:   "Make your login",
		Summary: "This task has a link for the login page.",
		Description: `<p>I'm glad you made your registration. Here is the
link: <a href="/login">login page</a>.</p>
<p>But if you forgot the password use this link:
<a href="/reset">reset password</a>.</p>`,
		Status:  "Blocked",
		Creator: "blmayer",
		Date:    time.Now().Add(-12 * time.Hour),
	},
	{
		ID:          2,
		Title:       "Create your user",
		Summary:     "The description of this task has a link to the registration page.",
		Description: `<p>Here is the link: <a href="/register">registration page</a>. Welcome!</p>`,
		Status:      "Active",
		Creator:     "blmayer",
		Date:        time.Now().Add(-30 * time.Hour),
	},
	{
		ID:      1,
		Title:   "Find this website",
		Summary: "Congratulations! You found this task manager.",
		Status:  "Done",
		Creator: "blmayer",
		Date:    time.Now().Add(-48 * time.Hour),
	},
}

type indexPayload struct {
	Tasks []Task
	User  User
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
	user, err := getUserFromCookie(*cookies[0])
	if err != nil {
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusUnauthorized)
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
	page := "task.html"
	if parts[1] == "edit" {
		page = "edit.html"
	}

	id, err := strconv.Atoi(parts[2])
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", p)
		return
	}
	p.Tasks = []Task{}

	cookies := r.Cookies()
	if len(cookies) != 1 {
		pages.ExecuteTemplate(w, "task.html", defaultTasks[4-id])
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
	if page == "task.html" {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		t.Description = string(md)
	}
	pages.ExecuteTemplate(w, page, t)
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
	pages.ExecuteTemplate(w, "profile.html", user)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := Token{
		Expires: time.Now(),
	}

	err = usersDB.Update(user.Key, base.Updates{"Token": token})
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
		Nick: r.Form.Get("nick"),
		Pass: r.Form.Get("password"),
	}
	if user.Nick == "" || user.Pass == "" {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:    base.Query{{"Nick": user.Nick}},
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

	if len(user.Pass) == 4 && user.Pass == dbUsers[0].Pass {
		pages.ExecuteTemplate(w, "newpass.html", user)
		return
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	if user.Pass != dbUsers[0].Pass {
		http.Error(w, "wrong password", http.StatusUnauthorized)
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
		pages.ExecuteTemplate(w, "login.html", time.Now())
		return
	}

	err := r.ParseForm()
	if err != nil {
		pages.ExecuteTemplate(w, "index.html", err)
		return
	}

	temp := r.Form.Get("temp")
	user := User{
		Nick: r.Form.Get("nick"),
		Pass: r.Form.Get("password"),
	}
	if user.Nick == "" || user.Pass == "" || temp == "" {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
		return
	}

	dbUsers := []User{}
	query := base.FetchInput{
		Q:    base.Query{{"Nick": user.Nick}},
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

	if temp != dbUsers[0].Pass {
		http.Error(w, "wrong temp password", http.StatusUnauthorized)
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
		Value:   string(t),
		Expires: time.Now().Add(120 * time.Hour),
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	up := base.Updates{"Pass": user.Pass, "Token": token}
	err = usersDB.Update(dbUsers[0].Key, up)
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

func resetPass(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		pages.ExecuteTemplate(w, "reset.html", time.Now())
		return
	}

	err := r.ParseForm()
	if err != nil {
		// TODO: Print error to an html page
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := User{
		Email: r.Form.Get("email"),
		Nick:  r.Form.Get("nick"),
	}
	if user.Email == "" || user.Nick == "" {
		// TODO: Same error page
		http.Error(w, "empty email or nick", http.StatusBadRequest)
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
		// TODO: Show error page
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(dbUsers) == 0 {
		// TODO: Show error page
		http.Error(w, "user not found", http.StatusUnauthorized)
		return
	}

	pass := make([]byte, 4)
	_, err = rand.Read(pass)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		// TODO: Same error page
		http.Error(w, "update "+err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/login", http.StatusFound)
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
		Email:      pol.Sanitize(r.Form.Get("email")),
		Nick:       pol.Sanitize(r.Form.Get("nick")),
		Pass:       r.Form.Get("password"),
		CreateDate: time.Now(),
	}
	if newUser.Email == "" || newUser.Nick == "" || newUser.Pass == "" {
		// TODO: Same error page
		http.Error(w, "empty fields", http.StatusBadRequest)
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
		Value:   string(token),
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
