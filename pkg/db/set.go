package db

import (
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pquerna/otp/totp"
)

func (d DataBase) RegisterQueue(id int64, user string) error {
	// Create TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      d.DataVars.TotpVendor,
		AccountName: user,
	})
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get generate totp key: %w", err)
	}

	wgKey, wgKeyPre, wgKeyPub, err := d.WireGuard.Gen()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to generate wireguard key: %w", err)
	}

	IPsOctet, err := d.CalculateIP()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to calculate IP address: %w", err)
	}

	_, err = d.DataBase.Exec(
		"INSERT INTO registration_queue(ID, UserName, TOTPSecret, Peer, PeerPre, PeerPub, IP) VALUES($1,$2,$3,$4,$5,$6,$7)",
		id,
		user,
		key.Secret(),
		wgKey,
		wgKeyPre,
		wgKeyPub,
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
	default:
		return fmt.Errorf("unknown param: %s", param)

	}
	return nil
}
