package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pquerna/otp/totp"
)

func (d DataBase) RegisterQueue(id int64, user string) error {
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
		return fmt.Errorf("RegisterQueue: failed to get generate totp key: %w", err)
	}

	wgCom := exec.Command("wg", "genkey")
	wgKey, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get generate peer key: %w", err)
	}

	wgCom = exec.Command("wg", "genpsk")
	wgKeyPre, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get generate peer preshared key: %w", err)
	}

	wgCom = exec.Command("wg", "pubkey")
	wgCom.Stdin = bytes.NewBuffer(wgKey)
	wgKeyPub, err := wgCom.Output()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get generate pub key: %w", err)
	}

	// Calculate IP address
	qIProw, err := d.DataBase.Query("SELECT IP from registration_queue")
	if err != nil {
		return fmt.Errorf("RegisterQueue: db: failed to query IPs from registration_queue: %w", err)
	}
	defer func(qIProw *sql.Rows) {
		err := qIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(qIProw)
	for qIProw.Next() {
		var IP string
		err = qIProw.Scan(&IP)
		if err != nil {
			return fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		IPs = append(IPs, IP)
	}

	uIProw, err := d.DataBase.Query("SELECT IP from users")
	if err != nil {
		return fmt.Errorf("func RegisterQueue: db: failed to query IPs from users: %w", err)
	}
	defer func(uIProw *sql.Rows) {
		err := uIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(uIProw)
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

	_, err = d.DataBase.Exec(
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

func (d DataBase) UnRegisterQUser(qUser *QueueUser) error {
	_, err := d.DataBase.Exec("DELETE FROM registration_queue WHERE id = $1",
		qUser.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d DataBase) RegisterUser(user *User) error {
	_, err := d.DataBase.Exec(
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
	_, err = d.DataBase.Exec(
		"DELETE FROM registration_queue WHERE ID = $1",
		&user.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d DataBase) UnregisterUser(user *User) error {
	_, err := d.DataBase.Exec("DELETE FROM users WHERE id = $1",
		user.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d DataBase) EnableUser(id *int64) error {
	_, err := d.DataBase.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		1,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to enable user: %w", err)
	}
	return nil
}

func (d DataBase) DisableUser(id *int64) error {
	_, err := d.DataBase.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		0,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to disable user: %w", err)
	}
	return nil
}

func (d DataBase) Edit(user *User, param string, val string) error {
	switch param {
	case "allowedips":
		_, err := d.DataBase.Exec("UPDATE users SET AllowedIPs = $1 WHERE ID = $2", val, user.ID)
		if err != nil {
			return fmt.Errorf("db: failed to edit row: %w", err)
		}
	case "ip":
		_, err := d.DataBase.Exec("UPDATE users SET IP = $1 WHERE ID = $2", val, user.ID)
		if err != nil {
			return fmt.Errorf("db: failed to edit row: %w", err)
		}
	}
	return nil
}