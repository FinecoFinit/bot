package concierge

import (
	tele "gopkg.in/telebot.v3"
)

type Managers struct {
	AdminDBIDs     *[]int64
	UserDBIDs      *[]int64
	QUserDBIDs     *[]int64
	SessionManager map[int64]bool
	MessageManager map[int64]*tele.Message
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
	ConfPrefix        string `yaml:"conf_prefix"`
	TotpVendor        string `yaml:"totp_vendor"`
}

type User struct {
	ID               int64
	UserName         string
	Enabled          int
	TOTPSecret       string
	Session          int
	SessionTimeStamp string
	Peer             string
	PeerPre          string
	PeerPub          string
	AllowedIPs       string
	IP               int
}

type Admin struct {
	ID       int64
	UserName string
}

type QueueUser struct {
	ID         int64
	UserName   string
	TOTPSecret string
	Peer       string
	PeerPub    string
	PeerPre    string
	IP         int
}
