package email

import (
	"bot/concierge"
	"bytes"
	"fmt"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/pquerna/otp/totp"
	"image/png"
	"io"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wneessen/go-mail"
)

type Email struct {
	Config      concierge.Config
	EmailClient *mail.Client
}

func (e Email) SendEmail(user *concierge.User) error {
	message := mail.NewMsg()
	if err := message.From(e.Config.EmailUser); err != nil {
		return fmt.Errorf("failed to set From address: %s", err)
	}
	if err := message.To(user.UserName); err != nil {
		return fmt.Errorf("failed to set To address: %s", err)
	}
	message.Subject("Wireguard config")
	message.SetBodyString(mail.TypeTextPlain, "Wireguard config file for "+user.UserName)
	err := message.AttachReader("wireguard_"+strings.Split(user.UserName, "@")[0]+"_"+e.Config.ConfPrefix+".conf", io.Reader(e.GenConf(user)))
	if err != nil {
		return err
	}
	confImg, err := e.GenConfImg(e.GenConf(user))
	if err != nil {
		return err
	}
	err = message.AttachReader("wireguard_"+strings.Split(user.UserName, "@")[0]+"_"+e.Config.ConfPrefix+".png", confImg)
	if err != nil {
		return err
	}
	img, err := e.GenKeyImage(user)
	if err != nil {
		return err
	}
	err = message.AttachReader("totp.png", img)
	if err != nil {
		return err
	}

	if err := e.EmailClient.DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send mail: %s", err)
	}
	return nil
}

func (e Email) GenConf(user *concierge.User) *bytes.Buffer {
	buf := bytes.NewBufferString(
		"[Interface]\r\n" +
			"Address = " + e.Config.WgSubNet + strconv.Itoa(user.IP) + "/32\r\n" +
			"PrivateKey = " + user.Peer + "\r\n" +
			"DNS = " + e.Config.WgDNS + "\r\n" +
			"\r\n" +
			"[Peer]\r\nPublicKey = " + e.Config.WgPublicKey + "\r\n" +
			"PresharedKey = " + user.PeerPre + "\r\n" +
			"AllowedIPs = " + user.AllowedIPs + "\r\n" +
			"Endpoint = " + e.Config.WgPublicIP + "\r\n" +
			"PersistentKeepalive = 15")
	return buf
}

func (e Email) GenConfImg(buf *bytes.Buffer) (*bytes.Buffer, error) {
	qrCode, err := qr.Encode(buf.String(), qr.M, qr.Auto)
	if err != nil {
		return nil, err
	}
	qrCode, _ = barcode.Scale(qrCode, 256, 256)
	imgBuf := new(bytes.Buffer)
	err = png.Encode(imgBuf, qrCode)
	if err != nil {
		return nil, err
	}
	return imgBuf, nil
}

func (e Email) GenKeyImage(user *concierge.User) (*bytes.Buffer, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      e.Config.TotpVendor,
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
