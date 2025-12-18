package utils

import (
	"GH-Server/pkg/zaplog"
	"bytes"
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"os"
	"text/template"

	_ "github.com/joho/godotenv/autoload"
	"github.com/jordan-wright/email"
)

var (
	smtpHost = os.Getenv("SMTP_HOST")
	smtpPort = os.Getenv("SMTP_PORT")
	smtpUser = os.Getenv("SMTP_USER")
	smtpAuth = os.Getenv("SMTP_AUTH")
)

type MailData struct {
	UserName string
	Code     string
}

func VerifyMailSend(to string, userName string, code string) error {
	tpl, err := template.ParseFiles("template/mail.html")
	if err != nil {
		zaplog.Zap.Error(fmt.Sprintf("parse template failed: %v", err))
		return err
	}
	data := MailData{
		UserName: userName,
		Code:     code,
	}
	var buf bytes.Buffer
	if err = tpl.Execute(&buf, data); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("execute template failed: %v", err))
		return err
	}
	e := email.NewEmail()
	e.From = fmt.Sprintf("%s<%s>", mime.QEncoding.Encode("UTF-8", "GH-Server"), smtpUser)
	e.To = []string{to}
	e.Subject = "GH-Server 验证码"
	e.HTML = buf.Bytes()
	auth := smtp.PlainAuth("", smtpUser, smtpAuth, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)
	if err = e.SendWithTLS(addr, auth, &tls.Config{
		ServerName: smtpHost,
	}); err != nil {
		if err.Error() == "short response: \u0000\u0000\u0000\u001a\u0000\u0000\u0000" {
			return nil
		}
		zaplog.Zap.Error(fmt.Sprintf("send email failed:%v", err))
		return err
	}
	return nil
}
