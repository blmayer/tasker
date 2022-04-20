package main

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func public(w http.ResponseWriter, r *http.Request) {
	p := indexPayload{
		Tasks: tasks,
		List:  List{Name: "tasks"},
	}
	parts := strings.Split(r.URL.Path, "/")
	switch parts[1] {
	case "":
		parts[1] = "index.html"
	case "favicon.ico":
		cont, err := content.ReadFile("favicon.ico")
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
		w.Write(cont)
		return
	}

	switch len(parts) {
	case 3:
		id, err := strconv.Atoi(parts[2])
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
		logErr("template", pages.ExecuteTemplate(w, "task.html", tasks[4-id]))
	default:
		logErr("template", pages.ExecuteTemplate(w, "index.html", p))
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	nick, pass, ok := r.BasicAuth()
	if !ok {
		public(w, r)
		return
	}

	sum := sha256.Sum256([]byte(pass))
	user, err := getUser(nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	} else if user.Pass != fmt.Sprintf("%x", sum) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
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
	nick, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "login.html", tasks[2]))
		return
	}

	sum := sha256.Sum256([]byte(pass))
	user, err := getUser(nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	} else if user.Pass != fmt.Sprintf("%x", sum) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
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

		// up := map[string]interface{}{"Lists." + name: newList}
		// err = usersDB.Update(user.Key, up)
		err = updateUser(user)
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
	nick, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "login.html", tasks[2]))
		return
	}

	sum := sha256.Sum256([]byte(pass))
	user, err := getUser(nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	} else if user.Pass != fmt.Sprintf("%x", sum) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
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
			case "task_display_limit":
				lim, err := strconv.Atoi(r.Form.Get(k))
				if err != nil {
					logErr("update task limit", err)
					logErr("template", pages.ExecuteTemplate(w, "error.html", err))
					return
				}
				up["Configs.TaskDisplayLimit"] = lim
				user.Configs.TaskDisplayLimit = lim
			}
		}

		// err = usersDB.Update(user.Key, up)
		err = updateUser(user)
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
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
		logErr("template", pages.ExecuteTemplate(w, "login.html", tasks[2]))
		return
	}

	sum := sha256.Sum256([]byte(pass))
	user, err := getUser(nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	} else if user.Pass != fmt.Sprintf("%x", sum) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
		return
	}

	for _, list := range user.Lists {
		listTasks, err := getTasks(list, user.Nick, user, 0)
		logErr("getTasks", err)
		for _, t := range listTasks {
			logErr("delete task", db.Fetch(db.Del().Key("tasks/"+t.Key), nil))
		}
	}

	err = db.Fetch(db.Del().Key("users/"+user.Key),nil)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

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
	user, err := getUser(nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	} else if user.Pass != fmt.Sprintf("%x", sum) {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Wrong password"))
		return
	}

	http.Redirect(w, r, "/"+user.Configs.DefaultList, http.StatusFound)
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

	dbUser, err := getUser(user.Nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "User not found"))
		return
	}
	if temp != dbUser.Pass {
		w.WriteHeader(http.StatusUnauthorized)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "wrong temp password"))
		return
	}

	sum := sha256.Sum256([]byte(user.Pass))
	user.Pass = fmt.Sprintf("%x", sum)

	// up := base.Updates{"Pass": user.Pass}
	// err = usersDB.Update(dbUsers[0].Key, up)
	err = updateUser(dbUser)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.Redirect(w, r, "/tasks/3", http.StatusFound)
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

	req := User{
		Email: r.Form.Get("email"),
		Nick:  r.Form.Get("nick"),
	}
	if req.Email == "" || req.Nick == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty email or nick"))
		return
	}

	user, err := getUser(req.Nick)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}
	if user.Nick == "" {
		w.Header().Set("WWW-Authenticate", "Basic")
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

	// up := base.Updates{
	// 	"Pass":  string(pass),
	// 	"Token": nil,
	// }
	// err = usersDB.Update(users[0].Key, up)
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
			DefaultList:      "tasks",
			TaskDisplayLimit: 20,
		},
	}
	if newUser.Email == "" || newUser.Nick == "" || newUser.Pass == "" {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", "Empty fields"))
		return
	}

	user, err := getUser(newUser.Nick)
	if user.Email != "" {
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

	err = db.Fetch(db.Put(newUser).Key("users/"+newUser.Nick), nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	http.Redirect(w, r, "/tasks/3", http.StatusFound)
}
