package dbmng

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thoas/go-funk"
)

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

type DB struct {
	Db *sql.DB
}

func (d DB) GetUsers() ([]User, error) {
	rows, err := d.Db.Query("SELECT * FROM users")
	if err != nil {
		return nil, fmt.Errorf("db: query failed: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

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
			PeerPre          string
			PeerPub          string
			AllowedIPs       string
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
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return nil, fmt.Errorf("db: failed to scan row: %w", err)
		}
		users = append(users, User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			Peer:             Peer,
			PeerPub:          PeerPub,
			PeerPre:          PeerPre,
			AllowedIPs:       AllowedIPs,
			IP:               IP})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return users, nil
}

func (d DB) GetUser(id *int64) (User, error) {
	var user User
	rows, err := d.Db.Query(
		"SELECT * FROM users WHERE id = $1",
		&id)
	if err != nil {
		return user, fmt.Errorf("db: query failed: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	for rows.Next() {
		var (
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
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&Enabled,
			&TOTPSecret,
			&Session,
			&SessionTimeStamp,
			&Peer,
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return user, fmt.Errorf("db: scan row: %w", err)
		}
		user = User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			Peer:             Peer,
			PeerPre:          PeerPre,
			PeerPub:          PeerPub,
			AllowedIPs:       AllowedIPs,
			IP:               IP}
	}
	if err = rows.Err(); err != nil {
		return user, fmt.Errorf("db: rows: %w", err)
	}
	return user, nil
}

func (d DB) GetUserName(u *string) (User, error) {
	var user User
	rows, err := d.Db.Query(
		"SELECT * FROM users WHERE UserName like $1",
		"%"+*u+"%")
	if err != nil {
		return user, fmt.Errorf("db: query failed: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	for rows.Next() {
		var (
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
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&Enabled,
			&TOTPSecret,
			&Session,
			&SessionTimeStamp,
			&Peer,
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return user, fmt.Errorf("db: scan row: %w", err)
		}
		user = User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			Peer:             Peer,
			PeerPre:          PeerPre,
			PeerPub:          PeerPub,
			AllowedIPs:       AllowedIPs,
			IP:               IP}
	}
	if err = rows.Err(); err != nil {
		return user, fmt.Errorf("db: rows: %w", err)
	}
	return user, nil
}

func (d DB) GetUsersIDs(ids *[]int64) error {
	usersIDs, err := d.GetUsers()
	if err != nil {
		return fmt.Errorf("db: failed to get user ids: %w", err)
	}
	for _, u := range usersIDs {
		if !funk.ContainsInt64(*ids, u.ID) {
			*ids = append(*ids, u.ID)
		}
	}
	return nil
}

func (d DB) GetQueueUsers() ([]QueueUser, error) {
	rows, err := d.Db.Query("SELECT * FROM registration_queue")
	if err != nil {
		return nil, fmt.Errorf("db: query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var qUsers []QueueUser

	for rows.Next() {
		var (
			ID         int64
			UserName   string
			TOTPSecret string
			Peer       string
			PeerPre    string
			PeerPub    string
			IP         int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&TOTPSecret,
			&Peer,
			&PeerPre,
			&PeerPub,
			&IP)
		if err != nil {
			return nil, fmt.Errorf("db: scan row: %w", err)
		}
		qUsers = append(qUsers, QueueUser{
			ID:       ID,
			UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return qUsers, nil
}

func (d DB) GetQueueUser(id *int64) (QueueUser, error) {
	var qUser QueueUser
	rows, err := d.Db.Query(
		"SELECT * FROM registration_queue WHERE id = $1",
		&id)
	if err != nil {
		return qUser, fmt.Errorf("db: query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	for rows.Next() {
		var (
			ID         int64
			UserName   string
			TOTPSecret string
			Peer       string
			PeerPre    string
			PeerPub    string
			IP         int
		)
		err = rows.Scan(
			&ID,
			&UserName,
			&TOTPSecret,
			&Peer,
			&PeerPre,
			&PeerPub,
			&IP)
		if err != nil {
			return qUser, fmt.Errorf("db: scan row: %w", err)
		}
		qUser = QueueUser{
			ID:         ID,
			UserName:   UserName,
			TOTPSecret: TOTPSecret,
			Peer:       Peer,
			PeerPre:    PeerPre,
			PeerPub:    PeerPub,
			IP:         IP}
	}
	if err = rows.Err(); err != nil {
		return qUser, fmt.Errorf("db: rows: %w", err)
	}

	return qUser, nil
}

func (d DB) GetQueueUsersIDs(ids *[]int64) error {
	usersIDs, err := d.GetQueueUsers()
	if err != nil {
		return fmt.Errorf("db: failed to get user ids: %w", err)
	}
	for _, u := range usersIDs {
		if !funk.ContainsInt64(*ids, u.ID) {
			*ids = append(*ids, u.ID)
		}
	}
	return nil
}

func (d DB) GetAdmins() ([]Admin, error) {
	rows, err := d.Db.Query("SELECT * FROM admins")
	if err != nil {
		return nil, fmt.Errorf("db: query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var admins []Admin

	for rows.Next() {
		var (
			ID       int64
			UserName string
		)
		err = rows.Scan(&ID, &UserName)
		if err != nil {
			return nil, fmt.Errorf("db: scan row: %w", err)
		}
		admins = append(admins, Admin{ID: ID, UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return admins, nil
}
