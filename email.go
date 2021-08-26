package main

import (
	"net/smtp"
	"os"
)

func sendEmail(email, nick, pass string) {
	account := os.Getenv("EMAIL_ACCOUNT")
	from := os.Getenv("EMAIL_FROM")
	fromPass := os.Getenv("EMAIL_PASS")
	auth := smtp.PlainAuth("", account, fromPass, "smtp.gmail.com")

	to := []string{email}
	msg := []byte("To: " + email + "\r\n" +
		"Subject: Tasker password reset for " + nick + " \r\n" +
		"\r\n" +
		"Dear tasker user.\r\n" +
		"We are very glad to see you are using our platform, please find " +
		"your temporary password bellow:\r\n\r\n" +
		pass + "\r\n\r\n" +
		"Login at https://tasker.blmayer.dev/login with this password to " +
		"to the reset page.\r\n\r\n" +
		"Truly yours,\r\n\r\n" +
		"The password reset bot\r\n",
	)

	err := smtp.SendMail("smtp.gmail.com:587", auth, from, to, msg)
	if err != nil {
		println("ERROR:", err.Error())
	}
}
