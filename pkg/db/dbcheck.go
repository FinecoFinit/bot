package dbmodule

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID               int64
	UserName         string
	Enabled          int
	TOTPSecret       string
	Session          int
	SessionTimeStamp string
	Peer             string
	Allowedips       string
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
	IP         int
}

type DB struct {
	Db *sql.DB
}

func (d DB) GetUsers() ([]User, error) {
	rows, err := d.Db.Query("SELECT * FROM users")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var (
			ID               int64
			UserName         string
			Enabled          int
			TOTPSecret       string
			Session          int
			SessionTimeStamp string
			Peer             string
			Allowedips       string
			IP               int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&Enabled,
			&TOTPSecret,
			&Session,
			&SessionTimeStamp,
			&Peer,
			&Allowedips,
			&IP)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		users = append(users, User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			Peer:             Peer,
			Allowedips:       Allowedips,
			IP:               IP})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return users, nil
}

func (d DB) GetUser(id *int64) (User, error) {
	var user User
	rows, err := d.Db.Query(
		"SELECT * FROM users WHERE id = $1",
		&id)
	if err != nil {
		return user, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			ID               int64
			UserName         string
			Enabled          int
			TOTPSecret       string
			Session          int
			SessionTimeStamp string
			Peer             string
			Allowedips       string
			IP               int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&Enabled,
			&TOTPSecret,
			&Session,
			&SessionTimeStamp,
			&Peer,
			&Allowedips,
			&IP)
		if err != nil {
			return user, fmt.Errorf("scan row: %w", err)
		}
		user = User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			Peer:             Peer,
			Allowedips:       Allowedips,
			IP:               IP}
	}
	if err = rows.Err(); err != nil {
		return user, fmt.Errorf("rows: %w", err)
	}
	return user, nil
}

func (d DB) GetQueueUsers() ([]QueueUser, error) {
	rows, err := d.Db.Query("SELECT * FROM registration_queue")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var queueusers []QueueUser

	for rows.Next() {
		var (
			ID         int64
			UserName   string
			TOTPSecret string
			Peer       string
			IP         int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&TOTPSecret,
			&Peer,
			&IP)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		queueusers = append(queueusers, QueueUser{
			ID:       ID,
			UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return queueusers, nil
}

func (d DB) GetQueueUser(id *int64) (QueueUser, error) {
	var queueuser QueueUser
	rows, err := d.Db.Query(
		"SELECT * FROM registration_queue WHERE id = $1",
		&id)
	if err != nil {
		return queueuser, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			ID         int64
			UserName   string
			TOTPSecret string
			Peer       string
			IP         int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&TOTPSecret,
			&Peer,
			&IP)
		if err != nil {
			return queueuser, fmt.Errorf("scan row: %w", err)
		}
		queueuser = QueueUser{
			ID:         ID,
			UserName:   UserName,
			TOTPSecret: TOTPSecret,
			Peer:       Peer,
			IP:         IP}
	}
	if err = rows.Err(); err != nil {
		return queueuser, fmt.Errorf("rows: %w", err)
	}

	return queueuser, nil
}

func (d DB) GetAdmins() ([]Admin, error) {
	rows, err := d.Db.Query("SELECT * FROM admins")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var admins []Admin

	for rows.Next() {
		var (
			ID       int64
			UserName string
		)
		err = rows.Scan(&ID, &UserName)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		admins = append(admins, Admin{ID: ID, UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	return admins, nil
}
