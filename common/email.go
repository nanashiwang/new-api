package common

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"slices"
	"strings"
	"time"
)

func generateMessageID() (string, error) {
	split := strings.Split(SMTPFrom, "@")
	if len(split) < 2 {
		return "", fmt.Errorf("invalid SMTP account")
	}
	domain := strings.Split(SMTPFrom, "@")[1]
	return fmt.Sprintf("<%d.%s@%s>", time.Now().UnixNano(), GetRandomString(12), domain), nil
}

func shouldUseSMTPLoginAuth() bool {
	if SMTPForceAuthLogin {
		return true
	}
	return isOutlookServer(SMTPAccount) || slices.Contains(EmailLoginAuthServerList, SMTPServer)
}

func getSMTPAuth() smtp.Auth {
	if shouldUseSMTPLoginAuth() {
		return LoginAuth(SMTPAccount, SMTPToken)
	}
	return smtp.PlainAuth("", SMTPAccount, SMTPToken, SMTPServer)
}

type EmailAttachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

func SendEmail(subject string, receiver string, content string) error {
	message, err := buildEmailMessage(subject, receiver, content, nil)
	if err != nil {
		return err
	}
	return sendSMTPMessage(receiver, message)
}

func SendEmailWithAttachments(subject string, receiver string, content string, attachments []EmailAttachment) error {
	message, err := buildEmailMessage(subject, receiver, content, attachments)
	if err != nil {
		return err
	}
	return sendSMTPMessage(receiver, message)
}

func buildEmailMessage(subject string, receiver string, content string, attachments []EmailAttachment) ([]byte, error) {
	if SMTPFrom == "" { // for compatibility
		SMTPFrom = SMTPAccount
	}
	id, err2 := generateMessageID()
	if err2 != nil {
		return nil, err2
	}
	if SMTPServer == "" && SMTPAccount == "" {
		return nil, fmt.Errorf("SMTP 服务器未配置")
	}
	encodedSubject := fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	from := mail.Address{Name: SystemName, Address: SMTPFrom}
	var encodedContent bytes.Buffer
	quotedPrintableWriter := quotedprintable.NewWriter(&encodedContent)
	if _, err := quotedPrintableWriter.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := quotedPrintableWriter.Close(); err != nil {
		return nil, err
	}
	if len(attachments) > 0 {
		boundary := fmt.Sprintf("mixed_%s", GetRandomString(12))
		var message bytes.Buffer
		message.WriteString(fmt.Sprintf("To: %s\r\n"+
			"From: %s\r\n"+
			"Subject: %s\r\n"+
			"Date: %s\r\n"+
			"Message-ID: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: multipart/mixed; boundary=%q\r\n\r\n",
			receiver, from.String(), encodedSubject, time.Now().Format(time.RFC1123Z), id, boundary))
		message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		message.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		message.WriteString(encodedContent.String())
		message.WriteString("\r\n")
		for _, attachment := range attachments {
			filename := strings.TrimSpace(attachment.Filename)
			if filename == "" {
				filename = "attachment"
			}
			contentType := strings.TrimSpace(attachment.ContentType)
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			contentTypeHeader := mime.FormatMediaType(contentType, map[string]string{"name": filename})
			if contentTypeHeader == "" {
				contentTypeHeader = contentType
			}
			disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
			message.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			message.WriteString(fmt.Sprintf("Content-Type: %s\r\n", contentTypeHeader))
			message.WriteString("Content-Transfer-Encoding: base64\r\n")
			message.WriteString(fmt.Sprintf("Content-Disposition: %s\r\n\r\n", disposition))
			message.WriteString(wrapBase64(base64.StdEncoding.EncodeToString(attachment.Data)))
			message.WriteString("\r\n")
		}
		message.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
		return message.Bytes(), nil
	}
	message := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: %s\r\n"+ // 添加 Message-ID 头
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"Content-Transfer-Encoding: quoted-printable\r\n\r\n%s\r\n",
		receiver, from.String(), encodedSubject, time.Now().Format(time.RFC1123Z), id, encodedContent.String()))
	return message, nil
}

func wrapBase64(data string) string {
	if len(data) == 0 {
		return ""
	}
	var builder strings.Builder
	for len(data) > 76 {
		builder.WriteString(data[:76])
		builder.WriteString("\r\n")
		data = data[76:]
	}
	builder.WriteString(data)
	builder.WriteString("\r\n")
	return builder.String()
}

func sendSMTPMessage(receiver string, message []byte) error {
	auth := getSMTPAuth()
	addr := fmt.Sprintf("%s:%d", SMTPServer, SMTPPort)
	to := strings.Split(receiver, ";")
	var err error
	if SMTPPort == 465 || SMTPSSLEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         SMTPServer,
		}
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", SMTPServer, SMTPPort), tlsConfig)
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, SMTPServer)
		if err != nil {
			return err
		}
		defer client.Close()
		if err = client.Auth(auth); err != nil {
			return err
		}
		if err = client.Mail(SMTPFrom); err != nil {
			return err
		}
		receiverEmails := strings.Split(receiver, ";")
		for _, receiver := range receiverEmails {
			if err = client.Rcpt(receiver); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(message)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
	} else {
		err = smtp.SendMail(addr, auth, SMTPFrom, to, message)
	}
	if err != nil {
		SysError(fmt.Sprintf("failed to send email to %s: %v", receiver, err))
	}
	return err
}
