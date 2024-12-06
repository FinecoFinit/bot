package service

import (
	"bot/concierge"
	"os"

	"github.com/rs/zerolog"
	"github.com/wneessen/go-mail"
	"gopkg.in/yaml.v3"
)

func ReadConfig(p string) (concierge.Config, error) {
	var c concierge.Config
	yamlData, err := os.ReadFile(p)
	if err != nil {
		return c, err
	}
	err = yaml.Unmarshal(yamlData, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

func InitLogger(p string) (zerolog.Logger, error) {
	logFile, err := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return zerolog.Logger{}, err
	}
	logger := zerolog.New(zerolog.MultiLevelWriter(os.Stdout, logFile)).With().Timestamp().Logger()
	return logger, nil
}

func InitEmail(c concierge.Config) (*mail.Client, error) {
	var e *mail.Client
	e, err := mail.NewClient(
		c.EmailAddress,
		mail.WithPort(587),
		mail.WithSSL(),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(c.EmailUser),
		mail.WithPassword(c.EmailPassword))
	if err != nil {
		return e, err
	}
	return e, nil
}
