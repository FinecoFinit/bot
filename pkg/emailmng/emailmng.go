package emailmng

import (
	"bot/pkg/dbmng"
	"bytes"
	"fmt"
	"github.com/pquerna/otp/totp"
	"image/png"
	"io"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wneessen/go-mail"
)

type HighWay struct {
	WgServerIP  *string
	WgPublicKey *string
	EmailClient *mail.Client
	EmailUser   *string
	EmailPass   *string
	EmailAddr   *string
}

func (h HighWay) SendEmail(user *dbmng.User) error {
	message := mail.NewMsg()
	if err := message.From(*h.EmailUser); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := message.To(user.UserName); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	message.Subject("Wireguard config")
	message.SetBodyString(mail.TypeTextPlain, "Wireguard config file for "+user.UserName)
	err := message.AttachReader("wireguard.conf", io.Reader(h.GenConf(user)))
	if err != nil {
		return err
	}
	img, err := h.GenKeyImage(user)
	if err != nil {
		return err
	}
	err = message.AttachReader("totp.png", img)
	if err != nil {
		return err
	}

	if err := h.EmailClient.DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}
	return nil
}

func (h HighWay) GenConf(user *dbmng.User) *bytes.Buffer {
	buf := bytes.NewBufferString(
		"[Interface]\r\n" +
			"Address = " + "192.168.88." + strconv.Itoa(user.IP) + "/32\r\n" +
			"PrivateKey = " + user.Peer + "\r\n" +
			"DNS = 192.168.28.15\r\n" +
			"\r\n" +
			"[Peer]\r\nPublicKey = " + *h.WgPublicKey + "\r\n" +
			"PresharedKey = " + user.PeerPre + "\r\n" +
			"AllowedIPs = " + user.AllowedIPs + "\r\n" +
			"Endpoint = " + *h.WgServerIP + "\r\n" +
			"PersistentKeepalive = 15")
	return buf
}

func (h HighWay) GenKeyImage(user *dbmng.User) (*bytes.Buffer, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "test",
		AccountName: user.UserName,
		Secret:      []byte(user.TOTPSecret)})
	if err != nil {
		return nil, err
	}

	img, err := key.Image(256, 256)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, img)
	if err != nil {
		return nil, err
	}

	return buf, err
}
