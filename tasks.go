package main

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"tasker/internal/permissions"
	"tasker/internal/types"
	"time"
)

func serveList(w http.ResponseWriter, r *http.Request, user types.User, owner string) {
	parts := strings.Split(r.URL.Path, "/")
	if parts[1] == "" {
		parts[1] = user.Configs.DefaultList
	}
	list := user.Lists[parts[1]]
	if list == nil {
		http.ServeFile(w, r, parts[1])
		return
	}

	page := 0
	pagination := r.URL.Query().Get("page")
	if pagination != "" {
		page, _ = strconv.Atoi(pagination)
	}

	tasks := make([]*types.Task, 0, len(list.Tasks))
	for _, t := range list.Tasks {
		if t.Status != "deleted" {
			tasks = append(tasks, t)
		}
	}
	p := indexPayload{
		User:  user,
		List:  *list,
		Tasks: tasks,
		Page:  page,
	}

	// Sort by time by default
	sort.SliceStable(p.Tasks, func(i, j int) bool {
		return p.Tasks[i].Date.Unix() > p.Tasks[j].Date.Unix()
	})

	logErr("template", pages.ExecuteTemplate(w, "index.html", p))
}

func serveTask(w http.ResponseWriter, r *http.Request, user types.User, owner string) {
	parts := strings.Split(r.URL.Path, "/")

	list := user.Lists[parts[1]]
	if list == nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", "List not found"))
		return
	}
	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	t := list.Tasks[taskID]
	if t == nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	logErr("template", pages.ExecuteTemplate(w, "task.html", t))
}

func serveTaskAction(w http.ResponseWriter, r *http.Request, user types.User, owner string) {
	parts := strings.Split(r.URL.Path, "/")

	list := user.Lists[parts[1]]
	if list.Name == "" {
		http.ServeFile(w, r, parts[1])
		return
	}

	// Check permissions
	if list.Permissions&permissions.WriteTask == 0 {
		logErr("template", pages.ExecuteTemplate(w, "error.html", "no permission"))
		return
	}

	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	switch parts[3] {
	case "new":
		err := r.ParseForm()
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		t := types.Task{
			ID:          len(list.Tasks),
			List:        list.Name,
			ListOwner:   list.Owner,
			Title:       pol.Sanitize(r.Form.Get("title")),
			Summary:     pol.Sanitize(r.Form.Get("summary")),
			Description: pol.Sanitize(r.Form.Get("description")),
			Status:      pol.Sanitize(r.Form.Get("status")),
			Creator:     user.Nick,
			Date:        time.Now(),
		}
		if due := r.Form.Get("due"); due != "" {
			dueTime, err := time.Parse(time.RFC3339, due)
			if err != nil {
				logErr("time.Parse", err)
				logErr("template", pages.ExecuteTemplate(w, "error.html", err))
				return
			}
			t.Due = &dueTime
		}

		if err = t.Encrypt(*key); err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err.Error()))
			return
		}

		user.Lists[list.Name].Tasks = append(user.Lists[list.Name].Tasks, &t)

	case "edit":
		t := user.Lists[list.Name].Tasks[taskID]
		if t == nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", "Task not found"))
			return
		}

		switch r.Method {
		case http.MethodGet:
			logErr("template", pages.ExecuteTemplate(w, "edit.html", t))
			return
		case http.MethodPost:
			err := r.ParseForm()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				logErr("template", pages.ExecuteTemplate(w, "error.html", err))
				return
			}

			newDate, err := time.Parse(time.RFC3339, r.Form.Get("date"))
			if err != nil {
				newDate = time.Now()
			}
			newDue := r.Form.Get("due")
			if newDue != "" {
				dueTime, err := time.Parse(time.RFC3339, newDue)
				if err != nil {
					logErr("time.Parse", err)
					logErr("template", pages.ExecuteTemplate(w, "error.html", err))
					return
				}
				t.Due = &dueTime
			} else {
				t.Due = nil
			}

			t.Title = pol.Sanitize(r.Form.Get("title"))
			t.Summary = pol.Sanitize(r.Form.Get("summary"))
			t.Description = pol.Sanitize(r.Form.Get("description"))
			t.Status = pol.Sanitize(r.Form.Get("status"))
			t.Creator = pol.Sanitize(r.Form.Get("creator"))
			t.Date = newDate
			if err = t.Encrypt(*key); err != nil {
				logErr("template", pages.ExecuteTemplate(w, "error.html", err.Error()))
				return
			}
		}
	case "delete":
		user.Lists[list.Name].Tasks[taskID].Status = "deleted"
	}

	writeDB(root)
	http.Redirect(w, r, "/"+list.Name, http.StatusSeeOther)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	logErr("template", pages.ExecuteTemplate(w, "new.html", types.Task{List: parts[2], Date: time.Now()}))
}
