package email

import (
	"bot/pkg/concierge"
	"github.com/wneessen/go-mail"
)

type HighWay struct {
	WgServerIP  *string
	WgPublicKey *string
	EmailClient *mail.Client
	DataVars    concierge.DataVars
	EmailUser   *string
	EmailPass   *string
	EmailAddr   *string
	ConfPrefix  *string
}
