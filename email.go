package main

import (
	"net/smtp"
)

func sendEmail(email, nick, pass string) {
	auth := smtp.PlainAuth("", "user@example.com", "password", "mail.google.com")

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

	err := smtp.SendMail("mail.example.com:25", auth, "sender@example.org", to, msg)
	if err != nil {
		println("ERROR:", err.Error())
	}
}
