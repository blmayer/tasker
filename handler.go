package main

import (
	"net/http"
	"sort"
	"time"
)

var tasks = []task{
	task{
		ID:          1,
		Title:       "Find this website",
		Summary:     "Congratulations! You found this task manager.",
		Status:      "Done",
		DateCreated: time.Now().Add(-48 * time.Hour),
	},
	task{
		ID:          4,
		Title:       "Learn to use this",
		Summary:     "Click on the + sign to create a new task.",
		Status:      "Active",
		DateCreated: time.Now().Add(-10 * time.Second),
	},
	task{
		ID:          2,
		Title:       "Create sign up page",
		Summary:     "I must create a sign up page",
		Status:      "Blocked",
		DateCreated: time.Now().Add(-30 * time.Hour),
	},
	task{
		ID:          3,
		Title:       "Make your login",
		Summary:     "This task will have a link for a login page",
		Status:      "Active",
		DateCreated: time.Now().Add(-12 * time.Hour),
	},
}

func index(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			pages.ExecuteTemplate(w, "index.html", err)
			return
		}

		t := task{
			Title:       r.Form.Get("title"),
			Summary:     r.Form.Get("summary"),
			Description: r.Form.Get("description"),
			Status:      r.Form.Get("status"),
			DateCreated: time.Now(),
			Creator:     r.Form.Get("creator"),
		}
		tasks = append(tasks, t)
	}

	// Sort by time by default
	sort.SliceStable(tasks, func(i, j int) bool {
		return tasks[i].DateCreated.Unix() > tasks[j].DateCreated.Unix()
	})

	pages.ExecuteTemplate(w, "index.html", tasks)
}

func newTask(w http.ResponseWriter, r *http.Request) {
	pages.ExecuteTemplate(w, "new.html", time.Now())
}
func login(w http.ResponseWriter, r *http.Request) {
	pages.ExecuteTemplate(w, "login.html", nil)
}
func register(w http.ResponseWriter, r *http.Request) {
	pages.ExecuteTemplate(w, "register.html", nil)
}
