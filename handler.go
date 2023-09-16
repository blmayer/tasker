package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func index(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	switch len(parts) {
	case 2:
		serveList(w, r, parts[1])
	case 3:
		serveTask(w, r, parts[1], parts[2])
	case 4:
		serveTaskAction(w, r, parts[1], parts[2], parts[3])
	}
}

func newList(w http.ResponseWriter, r *http.Request) {
	user := User{}
	err := db.Get("config", &user)
	if err != nil {
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
			CreateDate:  time.Now(),
		}

		up := map[string]interface{}{"Lists." + name: newList}
		err = db.Update("config", up)
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
	var user User
	err := db.Get("config", &user)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
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
				user.DefaultList = v
			case "task_display_limit":
				lim, err := strconv.Atoi(r.Form.Get(k))
				if err != nil {
					logErr("update task limit", err)
					logErr("template", pages.ExecuteTemplate(w, "error.html", err))
					return
				}
				up["Configs.TaskDisplayLimit"] = lim
				user.TaskDisplayLimit = lim
			}
		}

		err = db.Update("config", up)
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
	}
	logErr("template", pages.ExecuteTemplate(w, "profile.html", user))
}

