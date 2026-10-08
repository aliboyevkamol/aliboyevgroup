// Package mail sends transactional email. Without SMTP settings it only logs that a mail was skipped.
package mail

import (
	"log/slog"
	"net/smtp"
	"strings"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type Log struct {
	L        *slog.Logger
	ShowBody bool // development only: prints the body (e.g. reset links)
}

func (m Log) Send(to, subject, body string) error {
	if m.ShowBody {
		m.L.Info("mail not sent (SMTP not configured)", "to", to, "subject", subject, "body", body)
	} else {
		m.L.Info("mail not sent (SMTP not configured)", "to", to, "subject", subject)
	}
	return nil
}

type SMTP struct{ Host, Port, User, Pass, From string }

func clean(s string) string { return strings.NewReplacer("\r", "", "\n", "").Replace(s) }

func (s SMTP) Send(to, subject, body string) error {
	msg := "From: " + clean(s.From) + "\r\nTo: " + clean(to) + "\r\nSubject: " + clean(subject) + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body
	return smtp.SendMail(s.Host+":"+s.Port, smtp.PlainAuth("", s.User, s.Pass, s.Host), s.From, []string{clean(to)}, []byte(msg))
}
