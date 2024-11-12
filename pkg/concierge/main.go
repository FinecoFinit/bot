package concierge

import (
	"github.com/rs/zerolog"
	tele "gopkg.in/telebot.v4"
)

type DataVars struct {
	AdminLogChat       int64
	AdminLogChatThread int
	WgDNS              string
	WgSubNet           string
}

type Resources struct {
	AdminDBIDs     *[]int64
	UserDBIDs      *[]int64
	QUserDBIDs     *[]int64
	SessionManager map[int64]bool
	MessageManager map[int64]*tele.Message
	Logger         zerolog.Logger
}

type Config struct {
	AdminWgChatID     int64  `yaml:"admin_wg_chat"`
	AdminWgChatThread int    `yaml:"admin_wg_chat_thread"`
	WgPublicIP        string `yaml:"wg_public_ip"`
	WgSubNet          string `yaml:"wg_sub_net"`
	WgPublicKey       string `yaml:"wg_public_key"`
	WgPreKeysDir      string `yaml:"wg_pre_keys_dir"`
	WgAllowedIps      string `yaml:"wg_allowed_ips"`
	WgDNS             string `yaml:"wg_dns"`
	TgToken           string `yaml:"telegram_token"`
	DbPath            string `yaml:"db_path"`
	EmailUser         string `yaml:"email_user"`
	EmailPassword     string `yaml:"email_password"`
	EmailAddress      string `yaml:"email_address"`
	LogFilePath       string `yaml:"log_file_path"`
}
