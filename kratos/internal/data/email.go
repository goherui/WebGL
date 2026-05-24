package data

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	gomail "gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
	from   string
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func NewEmailService(logger log.Logger, config EmailConfig) *EmailService {
	dialer := gomail.NewDialer(config.Host, config.Port, config.Username, config.Password)
	
	log.NewHelper(logger).Info("Email service initialized")
	
	return &EmailService{
		dialer: dialer,
		from:   config.From,
	}
}

func (e *EmailService) SendVerificationCode(to string, code string) error {
	m := gomail.NewMessage()
	m.SetAddressHeader("From", e.from, "未来实验室")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "【未来实验室】邮箱验证码")
	m.SetBody("text/html", fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8">
			<style>
				body { font-family: 'Microsoft YaHei', Arial, sans-serif; background: #f5f5f5; margin: 0; padding: 20px; }
				.container { max-width: 600px; margin: 0 auto; background: #fff; padding: 40px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
				.header { text-align: center; margin-bottom: 30px; }
				.logo { font-size: 24px; font-weight: bold; color: #2dd4bf; }
				.title { font-size: 20px; color: #333; margin: 20px 0; }
				.code-box { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0; }
				.code { font-size: 32px; font-weight: bold; color: #fff; letter-spacing: 8px; }
				.tips { color: #666; font-size: 14px; line-height: 1.6; }
				.warning { color: #ef4444; font-size: 12px; margin-top: 20px; }
				.footer { text-align: center; margin-top: 30px; color: #999; font-size: 12px; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<div class="logo">🔬 未来实验室</div>
				</div>
				<div class="title">邮箱验证码</div>
				<div class="code-box">
					<div class="code">%s</div>
				</div>
				<div class="tips">
					<p>您好！</p>
					<p>您收到这封邮件是因为有人在未来实验室注册账号时使用了您的邮箱地址。</p>
					<p>请使用上方验证码完成注册，该验证码 <strong>60秒内有效</strong>。</p>
				</div>
				<div class="warning">
					⚠️ 如果这不是您本人操作，请忽略此邮件。
				</div>
				<div class="footer">
					<p>未来实验室 - 探索科技的无限可能</p>
				</div>
			</div>
		</body>
		</html>
	`, code))

	if err := e.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}

	return nil
}

func (e *EmailService) GenerateCode() (string, error) {
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = '0' + byte(n.Int64())
	}
	return string(code), nil
}

func (e *EmailService) ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}
	if !strings.Contains(parts[1], ".") {
		return false
	}
	return true
}
