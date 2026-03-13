package biz

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/Servora-Kit/servora/api/gen/go/conf/v1"
)

const (
	verifyEmailTmplName   = "verify_email.html"
	resetPasswordTmplName = "reset_password.html"
)

// VerifyEmailData 供 verify_email.html 使用的数据
type VerifyEmailData struct {
	Link        string // 验证链接
	ExpiryHours string // 展示用，如 "24"
}

// ResetPasswordData 供 reset_password.html 使用的数据
type ResetPasswordData struct {
	Link        string // 重置链接
	ExpiryHours string // 展示用，如 "1"
}

var (
	defaultVerifyEmailSubject   = "Verify your email"
	defaultResetPasswordSubject = "Reset your password"
	defaultVerifyEmailHTML      = `<p>Click <a href="{{.Link}}">here</a> to verify your email. This link expires in {{.ExpiryHours}} hours.</p>`
	defaultResetPasswordHTML    = `<p>Click <a href="{{.Link}}">here</a> to reset your password. This link expires in {{.ExpiryHours}} hour(s).</p>`
)

// RenderVerifyEmail 渲染邮箱验证邮件主题与正文。若 conf 中配置了 template_dir 且存在 verify_email.html 则使用该文件，否则使用内嵌默认。
func RenderVerifyEmail(cfg *conf.Mail, link string) (subject string, html []byte, err error) {
	subject = defaultVerifyEmailSubject
	data := VerifyEmailData{Link: link, ExpiryHours: "24"}

	if cfg != nil && cfg.GetTemplateDir() != "" {
		path := filepath.Join(cfg.GetTemplateDir(), verifyEmailTmplName)
		if b, e := renderTemplateFile(path, data); e == nil {
			return subject, b, nil
		}
	}

	t, err := template.New("").Parse(defaultVerifyEmailHTML)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", nil, err
	}
	return subject, buf.Bytes(), nil
}

// RenderResetPassword 渲染密码重置邮件主题与正文。若 conf 中配置了 template_dir 且存在 reset_password.html 则使用该文件，否则使用内嵌默认。
func RenderResetPassword(cfg *conf.Mail, link string) (subject string, html []byte, err error) {
	subject = defaultResetPasswordSubject
	data := ResetPasswordData{Link: link, ExpiryHours: "1"}

	if cfg != nil && cfg.GetTemplateDir() != "" {
		path := filepath.Join(cfg.GetTemplateDir(), resetPasswordTmplName)
		if b, e := renderTemplateFile(path, data); e == nil {
			return subject, b, nil
		}
	}

	t, err := template.New("").Parse(defaultResetPasswordHTML)
	if err != nil {
		return "", nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", nil, err
	}
	return subject, buf.Bytes(), nil
}

func renderTemplateFile(path string, data any) ([]byte, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := template.New(filepath.Base(path)).Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", path, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
