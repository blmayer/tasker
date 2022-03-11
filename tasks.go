package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/deta/deta-go/service/base"
	"github.com/gomarkdown/markdown"
)

func encrypt(text string, wg *sync.WaitGroup) (string, error) {
	defer wg.Done()
	cypher, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		&key.PublicKey,
		[]byte(text), nil,
	)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(cypher), nil
}

func decrypt(cypher string, wg *sync.WaitGroup) (string, error) {
	defer wg.Done()
	bytes, err := base64.StdEncoding.DecodeString(cypher)
	if err != nil {
		return "", err
	}

	text, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		key,
		bytes, nil,
	)

	return string(text), err
}

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

	page := 0
	pagination := r.URL.Query().Get("page")
	if pagination != "" {
		page, _ = strconv.Atoi(pagination)
	}

	var err error
	p := indexPayload{
		User:  user,
		List:  list,
		Tasks: tasks,
		Page:  page,
	}

	p.Tasks, err = getTasks(list, owner, user.Configs.TaskDisplayLimit, page)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
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

	logErr("template", pages.ExecuteTemplate(w, "index.html", p))
}

func getTask(id int, listName, owner string) (t Task, err error) {
	tasks := []Task{}
	query := base.FetchInput{
		Q:    base.Query{{"ID": id, "ListOwner": owner, "List": listName}},
		Dest: &tasks,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		return
	}

	if len(tasks) == 0 {
		return
	}
	t = tasks[0]

	// Decrypt
	wg := sync.WaitGroup{}
	wg.Add(3)
	var anyErr error
	go func() {
		t.Title, err = decrypt(t.Title, &wg)
		if err != nil {
			anyErr = err
		}
	}()
	go func() {
		t.Summary, err = decrypt(t.Summary, &wg)
		if err != nil {
			anyErr = err
		}
	}()
	go func() {
		t.Description, err = decrypt(t.Description, &wg)
		if err != nil {
			anyErr = err
		}
	}()
	wg.Wait()

	return t, anyErr
}

func saveTask(t Task) error {
	var err, anyErr error
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		t.Title, err = encrypt(t.Title, &wg)
		if err != nil {
			anyErr = err
		}
	}()
	go func() {
		t.Summary, err = encrypt(t.Summary, &wg)
		if err != nil {
			anyErr = err
		}
	}()
	go func() {
		t.Description, err = encrypt(t.Description, &wg)
		if err != nil {
			anyErr = err
		}
	}()

	wg.Wait()

	if anyErr != nil {
		return anyErr
	}
	_, err = tasksDB.Put(t)
	return err
}

func getTasks(list List, owner string, limit, lastID int) (ts []Task, err error) {
	if list.Permissions&(ReadPermission|PublicPermission) == 0 {
		err = fmt.Errorf("no permission on %s", list.Name)
		return
	}
	query := base.FetchInput{
		Q:     base.Query{{"List": list.Name, "ListOwner": owner, "ID?gte": lastID * limit}},
		Dest:  &ts,
		Limit: limit,
	}
	_, err = tasksDB.Fetch(&query)
	if err != nil {
		return
	}

	// Decode in parallel
	wg := sync.WaitGroup{}
	wg.Add(2 * len(ts))
	var anyErr error
	for i := range ts {
		go func(i int) {
			ts[i].Title, err = decrypt(ts[i].Title, &wg)
			if err != nil {
				anyErr = err
			}
		}(i)
		go func(i int) {
			ts[i].Summary, err = decrypt(ts[i].Summary, &wg)
			if err != nil {
				anyErr = err
			}
			ts[i].Description = ""
		}(i)
	}
	wg.Wait()

	return ts, anyErr
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
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	t, err := getTask(taskID, list.Name, owner)
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
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
			logErr("template", pages.ExecuteTemplate(w, "error.html", "no permission"))
			return
		}
	}

	taskID, err := strconv.Atoi(parts[2])
	if err != nil {
		logErr("template", pages.ExecuteTemplate(w, "error.html", err))
		return
	}

	t, err := getTask(taskID, list.Name, owner)
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

		t := Task{
			ID:          list.TaskNumber,
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

		if err := saveTask(t); err != nil {
			logErr("template", pages.ExecuteTemplate(w, "error.html", err))
			return
		}

		go func() {
			up := base.Updates{
				"Lists." + parts[1] + ".TaskNumber": usersDB.Util.Increment(1),
			}
			err = usersDB.Update(user.Key, up)
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
			t.Creator = pol.Sanitize(r.Form.Get("creator"))
			t.Date = newDate

			if err := saveTask(t); err != nil {
				logErr("template", pages.ExecuteTemplate(w, "error.html", err))
				return
			}
		}
	case "delete":
		err = tasksDB.Delete(t.Key)
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
