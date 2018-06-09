package main

import (
	"net/smtp"
	"config"
	"korok"
)

func SendMail(body string) {
	korok.Info("begin send mail")
	
	from := config.ShannonConf.FromMail
	pass := config.ShannonConf.FromPwd
	to := config.ShannonConf.ToMail

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: [BlockChain] Hourly Status\n\n" +
		body

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		korok.Fatal("smtp error : %s", err)
		return
	}

	korok.Info("send mail success: %s", body)
}