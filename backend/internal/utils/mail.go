package utils

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type MailConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
	Security string
}

func SendPasswordResetMail(cfg MailConfig, to, resetURL string) error {
	if strings.TrimSpace(cfg.Host) == "" || strings.TrimSpace(cfg.From) == "" {
		return fmt.Errorf("mail service is not configured")
	}
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(cfg.Host, port)
	from := strings.TrimSpace(cfg.From)
	fromHeader := fromAddressHeader(from, cfg.FromName)
	subject := "PicBed Switcher 密码重置"
	body := fmt.Sprintf("您好，\n\n请点击以下链接重置您的 PicBed Switcher 登录密码：\n%s\n\n如果不是您本人操作，请忽略此邮件。\n", resetURL)
	message := strings.Join([]string{
		"From: " + fromHeader,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if cfg.Username != "" || cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	security, err := resolveSMTPSecurity(cfg.Security, port)
	if err != nil {
		return err
	}
	if security == "ssl" {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Quit()
		return sendSMTP(client, auth, from, to, []byte(message))
	}
	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Quit()
	if err := applyStartTLS(client, cfg.Host, security); err != nil {
		return err
	}
	return sendSMTP(client, auth, from, to, []byte(message))
}

func resolveSMTPSecurity(value, port string) (string, error) {
	security := strings.ToLower(strings.TrimSpace(value))
	if security == "" || security == "auto" {
		if port == "465" {
			return "ssl", nil
		}
		return "auto", nil
	}
	switch security {
	case "ssl", "starttls", "none":
		return security, nil
	default:
		return "", fmt.Errorf("unsupported smtp security mode: %s", value)
	}
}

func applyStartTLS(client *smtp.Client, host, security string) error {
	if security == "none" {
		return nil
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		if security == "starttls" {
			return fmt.Errorf("smtp server does not support STARTTLS")
		}
		return nil
	}
	return client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12})
}

func fromAddressHeader(from, fromName string) string {
	name := strings.TrimSpace(fromName)
	if name == "" {
		return from
	}
	return (&mail.Address{Name: name, Address: from}).String()
}

func sendSMTP(client *smtp.Client, auth smtp.Auth, from, to string, message []byte) error {
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return err
	}
	return writer.Close()
}
