package utils

import (
	"crypto/tls"
	"fmt"
	"html"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
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
	message := buildPasswordResetMessage(fromHeader, to, subject, resetURL)

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

func buildPasswordResetMessage(from, to, subject, resetURL string) string {
	boundary := fmt.Sprintf("picbed-reset-%d", time.Now().UnixNano())
	textBody := fmt.Sprintf("您好，\n\n请点击以下链接重置您的 PicBed Switcher 登录密码：\n%s\n\n该链接将在限定时间后失效。如果不是您本人操作，请忽略此邮件。\n", resetURL)
	htmlBody := passwordResetHTML(resetURL)
	return strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		textBody,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		htmlBody,
		"--" + boundary + "--",
	}, "\r\n")
}

func passwordResetHTML(resetURL string) string {
	escapedURL := html.EscapeString(resetURL)
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>PicBed Switcher 密码重置</title>
</head>
<body style="margin:0;padding:0;background:#f5f7fb;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI','Microsoft YaHei',Arial,sans-serif;">
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#f5f7fb;padding:36px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="max-width:640px;background:#ffffff;border-radius:12px;overflow:hidden;box-shadow:0 18px 40px rgba(15,23,42,0.10);">
          <tr>
            <td align="center" style="background:#12a7bd;padding:34px 24px;color:#ffffff;">
              <div style="font-size:30px;line-height:1.25;font-weight:800;">🔐 密码重置</div>
            </td>
          </tr>
          <tr>
            <td style="padding:42px 42px 34px;">
              <h1 style="margin:0 0 18px;font-size:26px;line-height:1.35;color:#111827;font-weight:800;">重置您的 PicBed Switcher 登录密码</h1>
              <p style="margin:0 0 14px;font-size:16px;line-height:1.8;color:#4b5563;">我们收到了您的密码重置请求。点击下方按钮即可设置新的登录密码。</p>
              <p style="margin:0 0 30px;font-size:15px;line-height:1.8;color:#6b7280;">该链接将在限定时间后失效。如果不是您本人操作，请忽略此邮件，您的账户不会受到影响。</p>
              <div style="text-align:center;margin:34px 0 30px;">
				<a href="%s" target="_blank" rel="noopener noreferrer" style="display:inline-block;background:#12a7bd;color:#ffffff;text-decoration:none;font-size:16px;font-weight:800;padding:14px 28px;border-radius:8px;box-shadow:0 12px 22px rgba(18,167,189,0.22);">重置密码</a>
			  </div>
			  <p style="margin:0 0 8px;font-size:13px;line-height:1.7;color:#9ca3af;">如果按钮无法打开，请复制以下原始链接到浏览器访问：</p>
			  <div style="padding:12px 14px;background:#f8fafc;border:1px solid #e5e7eb;border-radius:8px;color:#475569;font-size:13px;line-height:1.6;word-break:break-all;">%s</div>
			</td>
		  </tr>
        </table>
        <div style="max-width:640px;margin:28px auto 0;padding-top:24px;border-top:1px solid #e5e7eb;text-align:center;color:#94a3b8;font-size:14px;line-height:1.8;">
          这是来自 <strong>PicBed Switcher</strong> 的账户安全邮件。<br>
          如需忽略，请直接删除本邮件。
        </div>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedURL, escapedURL)
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
