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
		listName = user.DefaultList
	}

	return user.Lists[listName]
}

func serveList(w http.ResponseWriter, r *http.Request, listName string) {
	var user User
	err := db.Get("config", &user)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	page := 0
	pagination := r.URL.Query().Get("page")
	if pagination != "" {
		page, _ = strconv.Atoi(pagination)
	}

	list := user.Lists[listName]
	p := indexPayload{
		List: list,
		Page: page,
	}

	p.Tasks, err = getTasks(listName, list.TaskNumber, page, user.TaskDisplayLimit)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	// sort by time by default
	sort.SliceStable(p.Tasks, func(i, j int) bool {
		return p.Tasks[i].Date.Unix() > p.Tasks[j].Date.Unix()
	})

	for i, t := range p.Tasks {
		md := markdown.ToHTML([]byte(t.Description), nil, nil)
		p.Tasks[i].Description = string(md)
	}

	logErr("template", pages.ExecuteTemplate(w, "index.html", p))
}

func getTask(id int, listName string) (t Task, err error) {
	tasks := []Task{}
	query := base.FetchInput{
		Q:    base.Query{{"ID": id, "List": listName}},
		Dest: &tasks,
	}
	_, err = db.Fetch(&query)
	if err != nil {
		return
	}

	if len(tasks) == 0 {
		return
	}
	t = tasks[0]

	return
}

func saveTask(t Task) error {
	_, err := db.Put(t)
	return err
}

func getTasks(list string, num, page, limit int) (ts []Task, err error) {
	query := base.FetchInput{
		Q:     base.Query{{"List": list, "ID?gte": num - page*limit}},
		Dest:  &ts,
		Limit: limit,
		Desc:  true,
	}

	// if lastID > 0 {
	// 	query.Q = append(query.Q, map[string]interface{}{"ID?gte": user.Lists[list.Name].TaskNumber - lastID*user.Configs.TaskDisplayLimit})
	// }
	_, err = db.Fetch(&query)
	return ts, err
}

func serveTask(w http.ResponseWriter, r *http.Request, list, task string) {
	taskID, err := strconv.Atoi(task)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	t, err := getTask(taskID, list)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	md := markdown.ToHTML([]byte(t.Description), nil, nil)
	t.Description = string(md)
	logErr("template", pages.ExecuteTemplate(w, "task.html", t))
}

func serveTaskAction(w http.ResponseWriter, r *http.Request, listName, task, action string) {
	var user User
	err := db.Get("config", &user)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	list := getUserList(user, listName)
	if list.Name == "" {
		http.ServeFile(w, r, listName)
		return
	}

	taskID, err := strconv.Atoi(task)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	t, err := getTask(taskID, list.Name)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	switch action {
	case "new":
		err := r.ParseForm()
		if err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		t := Task{
			ID:          list.TaskNumber,
			List:        list.Name,
			Title:       pol.Sanitize(r.Form.Get("title")),
			Summary:     pol.Sanitize(r.Form.Get("summary")),
			Description: pol.Sanitize(r.Form.Get("description")),
			Status:      pol.Sanitize(r.Form.Get("status")),
			Date:        time.Now(),
		}
		if due := r.Form.Get("due"); due != "" {
			dueTime, err := parseTime(due)
			if err != nil {
				logErr("time.Parse", err)
				logErr("template", pages.ExecuteTemplate(w, "error.html", err))
				return
			}
			t.Due = &dueTime
		}

		if err := saveTask(t); err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		go func() {
			up := base.Updates{
				"Lists." + listName + ".TaskNumber": db.Util.Increment(1),
			}
			err = db.Update("config", up)
			if err != nil {
				println("error on user update:", err.Error())
			}
		}()
	case "edit":
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
			t.Date = newDate

			if err := saveTask(t); err != nil {
				logErr("template", pages.ExecuteTemplate(w, "error.html", err))
				return
			}
		}
	case "delete":
		err = db.Delete(t.Key)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}
	}

	http.Redirect(w, r, "/"+list.Name, http.StatusSeeOther)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	logErr("template", pages.ExecuteTemplate(w, "new.html", Task{List: parts[2], Date: time.Now()}))
}

func parseTime(str string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04",
	}

	for _, f := range formats {
		t, err := time.Parse(f, str)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time in any format")
}
