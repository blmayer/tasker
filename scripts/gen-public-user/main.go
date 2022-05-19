package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/gob"
	"io"
	"os"
	"tasker/internal/permissions"
	"tasker/internal/types"
	"time"
)

var defaultUser = types.User{
	Nick: "public",
	Pass: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	Configs: types.Config{
		TaskDisplayLimit: 5,
		DefaultList:      "tasks",
	},
	CreateDate: time.Date(2021, 7, 10, 0, 18, 45, 40, time.UTC),
	Lists: map[string]*types.List{
		"tasks": {
			Owner:       "public",
			Name:        "tasks",
			Permissions: permissions.ReadTask,
			Tasks: []*types.Task{
				{
					ID:      0,
					Title:   "Find this website",
					List:    "tasks",
					Summary: "Congratulations! You found this task manager. Click to learn more.",
					Description: `### About
Tasker is a very simple list web app that everyone can use. I designed it to
be minimalistic, yes this weird looks is on purpose, and easy to use. But there
are some interesting features already:

- No javaScript
- No cookies
- Simple HTML with clean interface
- Free for you
- Tasks are encrypted by default
- Delete your account and data at any time
- No emails
- No ads
- I don't use or sell your data
- Completely open source

Pretty good, huh? I intend to keep working on it on my free time, but I can't
promise you too much. If this software ever get usefull for you please leave your
feedback, or consider making a donation, that will be greatly appreciated.

Tasker is very light on resources, the main page is only 4.02Kb, and will look
good on your cell phone as well with minimum data usage, it is build with simplicity
on mind and will not install anything on your machine nor use the local storage.

To see the source code and the development progress please visit my GitHub page:
[tasker](https://github.com/blmayer/tasker), if you have any doubt, request or
feedback you can send an email to [me](mailto:bleemayer@gmail.com), this
information can also be found on the footer of the main page.

Thank you!`,
					Status:  "Done",
					Creator: "blmayer",
					Date:    time.Date(2021, 8, 11, 0, 3, 15, 0, time.UTC),
				},

				{
					ID:      1,
					Title:   "Create your user",
					List:    "tasks",
					Summary: "The description of this task has a link to the registration page.",
					Description: `Here is the link: [registration page](/register). Welcome!
### Terms and conditions

This is an experimental software, use it at your own risk, we do not waranty that your
data will be available at all times. Although we will not share it with anyone,
except the other users you agree to. We encrypt the summary, title and the content
of your tasks to protect your privacy.

If you like to support this project you can make a donation at [kofi](https://ko-fi.com/blmayer).

Thank you for using tasker. `,
					Status:  "Active",
					Creator: "blmayer",
					Date:    time.Date(2021, 8, 16, 22, 10, 54, 0, time.UTC),
				},
				{
					ID:      2,
					List:    "tasks",
					Title:   "Make your login",
					Summary: "This task has a link for the login page.",
					Description: `I'm glad you made your registration. Here is the
link: [login page](/login).

But if you forgot the password use this link:
[reset password](/reset).`,
					Status:  "Blocked",
					Creator: "blmayer",
					Date:    time.Date(2021, 8, 16, 22, 34, 44, 0, time.UTC),
				},
				{
					ID:      3,
					List:    "tasks",
					Title:   "Learn to use this",
					Summary: "This task has a tutorial.",
					Description: `
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

***

See https://daringfireball.net/projects/markdown/ to learn more.`,
					Status:  "Active",
					Creator: "blmayer",
					Date:    time.Date(2021, 8, 19, 23, 51, 58, 0, time.UTC),
				},
			},
		},
	},
}

func main() {
	encKey := os.Getenv("KEY")
	encText, err := base64.StdEncoding.DecodeString(encKey)
	if err != nil {
		panic(err)
	}

	key, err := x509.ParsePKCS1PrivateKey(encText)
	if err != nil {
		panic(err)
	}

	for _, t := range defaultUser.Lists["tasks"].Tasks {
		enc, err := encrypt(t.Title, key)
		if err != nil {
			println(err.Error())
			continue
		}
		t.Title = enc

		enc, err = encrypt(t.Summary, key)
		if err != nil {
			println(err.Error())
			continue
		}
		t.Summary = enc

		enc, err = encrypt(t.Description, key)
		if err != nil {
			println(err.Error())
			continue
		}
		t.Description = enc
	}

	file, err := os.OpenFile("db.gob", os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(err.Error())
	}
	err = gob.NewEncoder(file).Encode(map[string]*types.User{"public": &defaultUser})
	file.Close()
	if err != nil && err != io.EOF {
		panic(err.Error())
	}
}

func encrypt(text string, key *rsa.PrivateKey) (string, error) {
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
