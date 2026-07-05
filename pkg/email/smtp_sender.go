package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPSender 真实 SMTP 邮件发送。
type SMTPSender struct {
	host string
	port int
	user string
	pass string
	from string
}

func (s *SMTPSender) Send(to, subject, body string) error {
	msg := s.buildMessage(to, subject, body)

	var err error
	if s.port == 465 {
		err = s.sendWithSSL(msg, []string{to})
	} else {
		err = s.sendWithSTARTTLS(msg, []string{to})
	}
	return err
}

func (s *SMTPSender) buildMessage(to, subject, body string) []byte {
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", to))
	buf.WriteString(fmt.Sprintf("Subject: =?UTF-8?B?%s?=\r\n", base64.StdEncoding.EncodeToString([]byte(subject))))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	return []byte(buf.String())
}

func (s *SMTPSender) sendWithSSL(msg []byte, to []string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second},
		"tcp",
		addr,
		&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12},
	)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer tlsConn.Close()
	tlsConn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(tlsConn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Quit()
	return s.sendOverConn(client, to, msg)
}

func (s *SMTPSender) sendWithSTARTTLS(msg []byte, to []string) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Quit()

	if err := client.StartTLS(&tls.Config{ServerName: s.host, MinVersion: tls.VersionTLS12}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	return s.sendOverConn(client, to, msg)
}

func (s *SMTPSender) sendOverConn(conn *smtp.Client, to []string, msg []byte) error {
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	if auth != nil {
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := conn.Mail(s.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := conn.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt to: %w", err)
		}
	}
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return w.Close()
}
