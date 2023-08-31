package main

import (
	//"crypto/rsa"
	"time"
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuvw1234567890/_."

	NoPermission = Permissions(1 << iota)
	ReadPermission
	WritePermission
	PublicPermission

	TokenExpiredError = Error(iota)
	NoSessionError
	EmptyValueError
)

type Token struct {
	Value   string
	Expires time.Time
}

type Permissions int

type Error int

func (e Error) Error() string {
	switch e {
	case TokenExpiredError:
		return "token expired"
	case NoSessionError:
		return "session not found"
	}
	return "unknown error"
}

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
	Due         *time.Time
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
	Page  int
}

//var key *rsa.PrivateKey

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
		ID:      2,
		Title:   "Create your user",
		List:    "tasks",
		Summary: "The description of this task has a link to the registration page.",
		Description: `<p>Here is the link: <a href="/register">registration page</a>. Welcome!</p>
<h3>Terms and conditions</h3>
<p>
These terms and conditions (“Agreement”) set forth the general terms and conditions of your use of the
tasker.blmayer.dev website (“Website” or “Service”) and any of its related products and
services (collectively, “Services”). This Agreement is legally binding between you (“User”, “you” or “your”)
and this Website operator (“Operator”, “we”, “us” or “our”). If you are entering into this agreement on behalf
of a business or other legal entity, you represent that you have the authority to bind such entity to this agreement
, in which case the terms “User”, “you” or “your” shall refer to such entity. If you do not have such authority, or
if you do not agree with the terms of this agreement, you must not accept this agreement and may not access and use
the Website and Services. By accessing and using the Website and Services, you acknowledge that you have read,
understood, and agree to be bound by the terms of this Agreement. You acknowledge that this Agreement is a contract
between you and the Operator, even though it is electronic and is not physically signed by you, and it governs your
use of the Website and Services. This terms and conditions policy was created with the help of the terms and
conditions generator at https://www.websitepolicies.com/terms-and-conditions-generator
</p>
<h3>Accounts and membership</h3>
<p>
If you create an account on the Website, you are responsible for maintaining the security of your account and you
are fully responsible for all activities that occur under the account and any other actions taken in connection with
it. We may, but have no obligation to, monitor and review new accounts before you may sign in and start using the
Services. You must immediately notify us of any unauthorized uses of your account or any other breaches of security.
We will not be liable for any acts or omissions by you, including any damages of any kind incurred as a result of such acts
or omissions.
</p>
<h3>User content</h3>
<p>
We do not own any data, information or material (collectively, “Content”) that you submit on the Website in the
course of using the Service. You shall have sole responsibility for the accuracy, quality, integrity, legality,
reliability, appropriateness, and intellectual property ownership or right to use of all submitted Content. </p>
<h3>Backups</h3>
<p>
We are not responsible for the Content residing on the Website. In no event shall we be held liable for any loss
of any Content. It is your sole responsibility to maintain appropriate backup of your Content.
</p>
<h3>Links to other resources</h3>
<p>
Although the Website and Services may link to other resources (such as websites, mobile applications, etc.), we are
not, directly or indirectly, implying any approval, association, sponsorship, endorsement, or affiliation with any
linked resource, unless specifically stated herein. We are not responsible for examining or evaluating, and we do
not warrant the offerings of, any businesses or individuals or the content of their resources. We do not assume any
responsibility or liability for the actions, products, services, and content of any other third parties. You should
carefully review the legal statements and other conditions of use of any resource which you access through a link
on the Website. Your linking to any other off-site resources is at your own risk.
</p>
<h3>Changes and amendments</h3>
<p>
We reserve the right to modify this Agreement or its terms related to the Website and Services at any time at our
discretion. When we do, we will post a notification on the main page of the Website. We may also provide notice to
you in other ways at our discretion, such as through the contact information you have provided.
</p>
<p>
An updated version of this Agreement will be effective immediately upon the posting of the revised Agreement
unless otherwise specified. Your continued use of the Website and Services after the effective date of the revised
Agreement (or such other act specified at that time) will constitute your consent to those changes.
</p>
<h3>Acceptance of these terms</h3>
<p>
You acknowledge that you have read this Agreement and agree to all its terms and conditions. By accessing and using
the Website and Services you agree to be bound by this Agreement. If you do not agree to abide by the terms of this
Agreement, you are not authorized to access or use the Website and Services.
</p>
<h3>Contacting us</h3>
<p>
If you have any questions, concerns, or complaints regarding this Agreement, we encourage you to contact us using
the details below:
</p>
<a href="mailto://bleemayer@gmail.com">bleemayer@gmail.com</a>
<p>
This document was last updated on March 18, 2022</p>`,
		Status:  "Active",
		Creator: "blmayer",
		Date:    time.Date(2021, 8, 16, 22, 10, 54, 0, time.UTC),
	},
	{
		ID:      1,
		Title:   "Find this website",
		List:    "tasks",
		Summary: "Congratulations! You found this task manager. Click to learn more.",
		Description: `<h3>About</h3>
		<p>Tasker is a very simple list web app that everyone can use. I designed it to
		be minimalistic, yes this weird looks is on purpose, and easy to use. But there
		are some interesting features already:</p>
		<ul>
			<li>No javaScript</li>
			<li>No cookies</li>
			<li>Simple HTML with clean interface</li>
			<li>Free for you</li>
			<li>Tasks are encrypted by default</li>
			<li>Delete your account and data at any time</li>
			<li>No emails</li>
			<li>No ads</li>
			<li>I don't use or sell your data</li>
			<li>Completely open source</li>
		</ul>
		<p>Pretty good, huh? I intend to keep working on it on my free time, but I can't
		promise you too much. If this software ever get usefull for you please leave your
		feedback, or consider making a donation, that will be greatly appreciated.</p>
		<p>Tasker is very light on resources, the main page is only 4.02Kb, and will look
		good on your cell phone as well with minimum data usage, it is build with simplicity
		on mind and will not install anything on your machine nor use the local storage.</p>
		<p>To see the source code and the development progress please visit my GitHub page:
		<a href="//github.com/blmayer/tasker">Tasker</a>, if you have any doubt, request or
		feedback you can send an email to <a href:"mailto:bleemayer@gmail.com">me</a>, this
		information can also be found on the footer of the main page.</p>
		<p>Thank you!</p>`,
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
