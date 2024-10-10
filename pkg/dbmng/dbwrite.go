package dbmng

import (
	"bytes"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pquerna/otp/totp"
)

func (d DB) RegisterQueue(id int64, user string) error {
	var (
		IPsPool []int
		IPs     []string
	)

	// Create TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "test",
		AccountName: user,
	})

	if err != nil {
		return fmt.Errorf("func RegisterQueue: failed to get generate totp key: %w", err)
	}

	wgCom := exec.Command("wg", "genkey")
	wgKey, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("func RegisterQueue: failed to get generate peer key: %w", err)
	}

	wgCom = exec.Command("wg", "genpsk")
	wgKeyPre, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("func RegisterQueue: failed to get generate peer key: %w", err)
	}

	wgCom = exec.Command("wg", "pubkey")
	wgCom.Stdin = bytes.NewBuffer(wgKey)
	wgKeyPub, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("func RegisterQueue: failed to get generate pub key: %w", err)
	}

	// Calculate IP address
	qIProw, err := d.Db.Query("SELECT IP from registration_queue")
	if err != nil {
		return fmt.Errorf("RegisterQueue: db: failed to query IPs from registration_queue: %w", err)
	}
	defer qIProw.Close()
	for qIProw.Next() {
		var IP string
		err = qIProw.Scan(&IP)
		if err != nil {
			return fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		IPs = append(IPs, IP)
	}

	uIProw, err := d.Db.Query("SELECT IP from users")
	if err != nil {
		return fmt.Errorf("func RegisterQueue: db: failed to query IPs from users: %w", err)
	}
	defer uIProw.Close()
	for uIProw.Next() {
		var IP string
		err = uIProw.Scan(&IP)
		if err != nil {
			return fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		IPs = append(IPs, IP)
	}

	for i := 130; i < 255; i++ {
		IPsPool = append(IPsPool, i)
	}

	IPsOctet := slices.DeleteFunc(IPsPool, func(n int) bool {
		return slices.Contains(IPs, strconv.Itoa(n))
	})

	_, err = d.Db.Exec(
		"INSERT INTO registration_queue(ID, UserName, TOTPSecret, Peer, PeerPre, PeerPub, IP) VALUES($1,$2,$3,$4,$5,$6,$7)",
		id,
		user,
		key.Secret(),
		strings.TrimSuffix(string(wgKey[:]), "\r\n"),
		strings.TrimSuffix(string(wgKeyPre[:]), "\r\n"),
		strings.TrimSuffix(string(wgKeyPub[:]), "\r\n"),
		IPsOctet[0])
	if err != nil {
		return fmt.Errorf("db: insert into registration_queue: %w", err)
	}
	return nil
}

func (d DB) RegisterUser(user *User) error {
	_, err := d.Db.Exec(
		"INSERT INTO users(ID, UserName, Enabled, TOTPSecret, Session, SessionTimeStamp, Peer, PeerPre, PeerPub, AllowedIPs, IP) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
		&user.ID,
		&user.UserName,
		&user.Enabled,
		&user.TOTPSecret,
		&user.Session,
		&user.SessionTimeStamp,
		&user.Peer,
		&user.PeerPre,
		&user.PeerPub,
		&user.AllowedIPs,
		&user.IP)
	if err != nil {
		return fmt.Errorf("db: insert into users: %w", err)
	}
	_, err = d.Db.Exec(
		"DELETE FROM registration_queue WHERE ID = $1",
		&user.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d DB) EnableUser(id *string) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		1,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to enable user: %w", err)
	}
	return nil
}

func (d DB) DisableUser(id *string) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		0,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to disable user: %w", err)
	}
	return nil
}
