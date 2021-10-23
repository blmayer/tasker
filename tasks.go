package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deta/deta-go/service/base"
	"github.com/gomarkdown/markdown"
)

func getUserList(user User, listName string) List {
	if listName == "" {
		listName = user.Configs.DefaultList
	}

	return user.Lists[listName]
}

func serveList(w http.ResponseWriter, r *http.Request, user User, owner string) {
	parts := strings.Split(r.URL.Path, "/")
	list := getUserList(user, parts[1])
	if list.Name == "" {
		http.ServeFile(w, r, parts[1])
		return
	}

	var err error
	p := indexPayload{
		User:  defaultUser,
		List:  list,
		Tasks: tasks,
	}

	p.Tasks, err = getTasks(list, owner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

func getTask(id int, listName, owner string) (t Task, err error) {
	tasks := []Task{}
	query := base.FetchInput{
		Q:    base.Query{{"ID": id, "ListOwner": owner, "List": listName}},
		Dest: &tasks,
	}
	_, err = tasksDB.Fetch(&query)
	t = tasks[0]
	return
}

func getTasks(list List, owner string) (t []Task, err error) {
	if list.Permissions&(ReadPermission|PublicPermission) == 0 {
		err = fmt.Errorf("no permission on %s", list.Name)
		return
	}
	query := base.FetchInput{
		Q:    base.Query{{"List": list.Name, "ListOwner": owner}},
		Dest: &t,
	}
	_, err = tasksDB.Fetch(&query)

	return
}

func serveTask(w http.ResponseWriter, r *http.Request, user User, owner string) {
	parts := strings.Split(r.URL.Path, "/")

	list := getUserList(user, parts[1])
	if list.Name == "" {
		http.ServeFile(w, r, parts[1])
		return
	}
	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", err))
		return
	}

	t, err := getTask(taskID, list.Name, owner)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", err))
		return
	}

	md := markdown.ToHTML([]byte(t.Description), nil, nil)
	t.Description = string(md)
	logErr("template", pages.ExecuteTemplate(w, "task.html", t))
}

func serveTaskAction(w http.ResponseWriter, r *http.Request, user User, owner string) {
	parts := strings.Split(r.URL.Path, "/")

	list := getUserList(user, parts[1])
	if list.Name == "" {
		http.ServeFile(w, r, parts[1])
		return
	}

	// Check permissions
	if owner != user.Nick {
		if getUserList(user, parts[1]).Permissions&WritePermission == 0 {
			logErr("template", pages.ExecuteTemplate(w, "index.html", "no permission"))
			return
		}
	}

	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", err))
		return
	}

	t, err := getTask(taskID, list.Name, owner)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "index.html", err))
		return
	}

	switch parts[3] {
	case "new":
		err := r.ParseForm()
		if err != nil {
			pages.ExecuteTemplate(w, "index.html", err)
			return
		}

		t := Task{
			ID:          list.TaskNumber,
			List:        list.Name,
			ListOwner:   t.ListOwner,
			Title:       pol.Sanitize(r.Form.Get("title")),
			Summary:     pol.Sanitize(r.Form.Get("summary")),
			Description: pol.Sanitize(r.Form.Get("description")),
			Status:      pol.Sanitize(r.Form.Get("status")),
			Creator:     user.Nick,
			Date:        time.Now(),
		}

		go func() {
			_, err = tasksDB.Put(t)
			if err != nil {
				println("put error:", err)
			}
		}()

		up := base.Updates{
			"Lists." + parts[1] + ".TaskNumber": usersDB.Util.Increment(1),
		}
		err = usersDB.Update(user.Key, up)
		if err != nil {
			pages.ExecuteTemplate(w, "index.html", err)
			return
		}
	case "edit":
		if r.Method == http.MethodPost {
			err := r.ParseForm()
			if err != nil {
				logErr("template", pages.ExecuteTemplate(w, "index.html", err))
				return
			}

			newDate, err := time.Parse(time.RFC3339, r.Form.Get("date"))
			if err != nil {
				newDate = time.Now()
			}

			t.Title = pol.Sanitize(r.Form.Get("title"))
			t.Summary = pol.Sanitize(r.Form.Get("summary"))
			t.Description = pol.Sanitize(r.Form.Get("description"))
			t.Status = pol.Sanitize(r.Form.Get("status"))
			t.Creator = pol.Sanitize(r.Form.Get("creator"))
			t.Date = newDate

			_, err = tasksDB.Put(t)
			if err != nil {
				logErr("template", pages.ExecuteTemplate(w, "index.html", err))
				return
			}

			break
		}
		logErr("template", pages.ExecuteTemplate(w, "edit.html", t))
	case "delete":
		err = tasksDB.Delete(t.Key)
		if err != nil {
			// TODO: Show error page
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/"+list.Name, http.StatusSeeOther)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	logErr("template", pages.ExecuteTemplate(w, "new.html", Task{List: parts[2], Date: time.Now()}))
}
