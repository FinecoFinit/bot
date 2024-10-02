package dbmodule

import (
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
		IPspool []int
		IPs     []string
	)

	// Create TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "test",
		AccountName: user,
	})
	if err != nil {
		panic(err)
	}
	// Generate WG key
	wgcom := exec.Command("wg", "genkey")
	wgkey, err := wgcom.Output()
	if err != nil {
		panic(err)
	}

	// Calcuate IP address
	queueIProw, err := d.Db.Query("SELECT IP from registration_queue")
	if err != nil {
		panic(err)
	}
	defer queueIProw.Close()
	for queueIProw.Next() {
		var IP string
		err = queueIProw.Scan(&IP)
		if err != nil {
			panic(err)
		}
		IPs = append(IPs, IP)
	}

	userIProw, err := d.Db.Query("SELECT IP from users")
	if err != nil {
		panic(err)
	}
	defer userIProw.Close()
	for userIProw.Next() {
		var IP string
		err = userIProw.Scan(&IP)
		if err != nil {
			panic(err)
		}
		IPs = append(IPs, IP)
	}

	for i := 130; i < 255; i++ {
		IPspool = append(IPspool, i)
	}

	IPsloc := slices.DeleteFunc(IPspool, func(n int) bool {
		return slices.Contains(IPs, strconv.Itoa(n))
	})

	_, err = d.Db.Exec(
		"insert into registration_queue(ID, UserName, TOTPSecret, Peer, IP) values($1,$2,$3,$4,$5)",
		id,
		user,
		key.Secret(),
		strings.TrimSuffix(string(wgkey[:]), "\n"),
		IPsloc[0])
	if err != nil {
		return fmt.Errorf("db: insert into registration_queue: %w", err)
	}
	return nil
}

func (d DB) RegisterUser(user *User) error {
	_, err := d.Db.Exec(
		"INSERT INTO users(ID,UserName,Enabled,TOTPSecret,Session,SessionTimeStamp,Peer,AllowedIPs,IP) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
		&user.ID,
		&user.UserName,
		&user.Enabled,
		&user.TOTPSecret,
		&user.Session,
		&user.SessionTimeStamp,
		&user.Peer,
		&user.Allowedips,
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
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (d DB) DisableUser(id *string) error {
	_, err := d.Db.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		0,
		&id)
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
