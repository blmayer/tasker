package main

import (
	"crypto/rsa"
	"time"
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."

	NoPermission = Permissions(1 << iota)
	ReadPermission
	WritePermission
	PublicPermission
)

type Token struct {
	Value   string
	Expires time.Time
}

type Permissions int

type Task struct {
	Key         string `json:"key"`
	ID          int
	List        string
	ListOwner   string
	Title       string
	Status      string
	Summary     string
	Description string
	Creator     string
	Date        time.Time
}

type List struct {
	Name        string
	Permissions Permissions
	TaskNumber  int
	Owner       string
	CreateDate  time.Time
}

type Config struct {
	TaskDisplayLimit int
	DefaultList      string
	DefaultSort      string
	DefaultFilter    string
}

type User struct {
	Key        string `json:"key"`
	Nick       string
	Email      string
	Pass       string
	Token      Token
	CreateDate time.Time
	Configs    Config
	Lists      map[string]List
}

type indexPayload struct {
	User  User
	List  List
	Tasks []Task
}

var key *rsa.PrivateKey

var reservedNames = []string{
	"tasks", "new", "newlist", "profile", "reset", "login", "reset", "logout",
	"register", "newpass", "delete",
}

var defaultUser = User{
	Nick: "blmayer",
	Lists: map[string]List{
		"tasks": {Owner: "blmayer", Permissions: PublicPermission},
	},
}

var tasks = []Task{
	{
		ID:      4,
		List:    "tasks",
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

~~stroked~~ through text using tildes: ` + "`~~`" + `.

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
		Date:    time.Date(2021, 8, 19, 23, 51, 58, 0, time.UTC),
	},
	{
		ID:      3,
		List:    "tasks",
		Title:   "Make your login",
		Summary: "This task has a link for the login page.",
		Description: `<p>I'm glad you made your registration. Here is the
link: <a href="/login">login page</a>.</p>
<p>But if you forgot the password use this link:
<a href="/reset">reset password</a>.</p>`,
		Status:  "Blocked",
		Creator: "blmayer",
		Date:    time.Date(2021, 8, 16, 22, 34, 44, 0, time.UTC),
	},
	{
		ID:          2,
		Title:       "Create your user",
		List:        "tasks",
		Summary:     "The description of this task has a link to the registration page.",
		Description: `<p>Here is the link: <a href="/register">registration page</a>. Welcome!</p>`,
		Status:      "Active",
		Creator:     "blmayer",
		Date:        time.Date(2021, 8, 16, 22, 10, 54, 0, time.UTC),
	},
	{
		ID:      1,
		Title:   "Find this website",
		List:    "tasks",
		Summary: "Congratulations! You found this task manager.",
		Status:  "Done",
		Creator: "blmayer",
		Date:    time.Date(2021, 8, 11, 0, 3, 15, 0, time.UTC),
	},
}

func isReservedName(name string) bool {
	for _, w := range reservedNames {
		if name == w {
			return true
		}
	}
	return false
}
