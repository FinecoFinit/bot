package storage

import (
	"bot/concierge"
	"bot/service/wg"
	"database/sql"
	"encoding/json"
	"fmt"
	tele "gopkg.in/telebot.v3"
	"time"

	"github.com/pquerna/otp/totp"
)

type MySql struct {
	MySql     *sql.DB
	Wireguard *wg.WireGuard
	Config    concierge.Config
}

func (d MySql) GetUsers() ([]concierge.User, error) {
	rows, err := d.MySql.Query("SELECT * FROM users")
	if err != nil {
		return nil, fmt.Errorf("db: query failed: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var users []concierge.User

	for rows.Next() {
		var (
			ID               int64
			UserName         string
			Enabled          int
			TOTPSecret       string
			Session          int
			SessionTimeStamp string
			SessionMessageID string
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
			&SessionMessageID,
			&Peer,
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return nil, fmt.Errorf("db: failed to scan row: %w", err)
		}
		users = append(users, concierge.User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			SessionMessageID: SessionMessageID,
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

func (d MySql) GetUser(id *int64) (concierge.User, error) {
	var user concierge.User
	rows, err := d.MySql.Query(
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
			SessionMessageID string
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
			&SessionMessageID,
			&Peer,
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return user, fmt.Errorf("db: scan row: %w", err)
		}
		user = concierge.User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			SessionMessageID: SessionMessageID,
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

func (d MySql) GetUserName(u *string) (concierge.User, error) {
	var user concierge.User
	rows, err := d.MySql.Query(
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
			SessionMessageID string
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
			&SessionMessageID,
			&Peer,
			&PeerPre,
			&PeerPub,
			&AllowedIPs,
			&IP)
		if err != nil {
			return user, fmt.Errorf("db: scan row: %w", err)
		}
		user = concierge.User{
			ID:               ID,
			UserName:         UserName,
			Enabled:          Enabled,
			TOTPSecret:       TOTPSecret,
			Session:          Session,
			SessionTimeStamp: SessionTimeStamp,
			SessionMessageID: SessionMessageID,
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

func (d MySql) GetUsersIDs(ids *[]int64) error {
	usersIDs, err := d.GetUsers()
	if err != nil {
		return fmt.Errorf("db: failed to get user ids: %w", err)
	}
	*ids = nil
	for _, u := range usersIDs {
		*ids = append(*ids, u.ID)
	}
	return nil
}

func (d MySql) GetQueueUsers() ([]concierge.QueueUser, error) {
	rows, err := d.MySql.Query("SELECT * FROM registration_queue")
	if err != nil {
		return nil, fmt.Errorf("db: query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var qUsers []concierge.QueueUser

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
		qUsers = append(qUsers, concierge.QueueUser{
			ID:       ID,
			UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return qUsers, nil
}

func (d MySql) GetQueueUser(id *int64) (concierge.QueueUser, error) {
	var qUser concierge.QueueUser
	rows, err := d.MySql.Query(
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
		qUser = concierge.QueueUser{
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

func (d MySql) GetQueueUsersIDs(ids *[]int64) error {
	usersIDs, err := d.GetQueueUsers()
	if err != nil {
		return fmt.Errorf("db: failed to get queue users ids: %w", err)
	}
	*ids = nil
	for _, u := range usersIDs {
		*ids = append(*ids, u.ID)
	}
	return nil
}

func (d MySql) GetAdmins() ([]concierge.Admin, error) {
	rows, err := d.MySql.Query("SELECT * FROM admins")
	if err != nil {
		return nil, fmt.Errorf("db: query: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var admins []concierge.Admin

	for rows.Next() {
		var (
			ID       int64
			UserName string
		)
		err = rows.Scan(&ID, &UserName)
		if err != nil {
			return nil, fmt.Errorf("db: scan row: %w", err)
		}
		admins = append(admins, concierge.Admin{ID: ID, UserName: UserName})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return admins, nil
}

func (d MySql) GetAdminsIDs(ids *[]int64) error {
	adminsIDs, err := d.GetAdmins()
	if err != nil {
		return fmt.Errorf("db: failed to get admins ids: %w", err)
	}
	*ids = nil
	for _, a := range adminsIDs {
		*ids = append(*ids, a.ID)
	}
	return nil
}

func (d MySql) GetUsersIPs() ([]int, error) {
	var pool []int
	qIProw, err := d.MySql.Query("SELECT IP from users")
	if err != nil {
		return nil, fmt.Errorf("RegisterQueue: db: failed to query IPs from registration_queue: %w", err)
	}
	defer func(qIProw *sql.Rows) {
		err := qIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(qIProw)
	for qIProw.Next() {
		var IP int
		err = qIProw.Scan(&IP)
		if err != nil {
			return nil, fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		pool = append(pool, IP)
	}
	return pool, nil
}

func (d MySql) GetQUsersIPs() ([]int, error) {
	var pool []int
	qIProw, err := d.MySql.Query("SELECT IP from registration_queue")
	if err != nil {
		return nil, fmt.Errorf("RegisterQueue: db: failed to query IPs from registration_queue: %w", err)
	}
	defer func(qIProw *sql.Rows) {
		err := qIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(qIProw)
	for qIProw.Next() {
		var IP int
		err = qIProw.Scan(&IP)
		if err != nil {
			return nil, fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		pool = append(pool, IP)
	}
	return pool, nil
}

func (d MySql) RegisterQueue(id int64, user string) error {
	// Create TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      d.Config.TotpVendor,
		AccountName: user,
	})
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get generate totp key: %w", err)
	}

	keys, err := d.Wireguard.GenKeys()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to generate wireguard key: %w", err)
	}

	uPool, err := d.GetUsersIPs()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get users ips: %w", err)
	}
	quPool, err := d.GetQUsersIPs()
	if err != nil {
		return fmt.Errorf("RegisterQueue: failed to get queue users ips: %w", err)
	}

	_, err = d.MySql.Exec(
		"INSERT INTO registration_queue(ID, UserName, TOTPSecret, Peer, PeerPre, PeerPub, IP) VALUES($1,$2,$3,$4,$5,$6,$7)",
		id,
		user,
		key.Secret(),
		keys.Private,
		keys.PreShared,
		keys.Public,
		concierge.CalculateIP(uPool, quPool)[0])
	if err != nil {
		return fmt.Errorf("db: insert into registration_queue: %w", err)
	}
	return nil
}

func (d MySql) UnRegisterQUser(qUser *concierge.QueueUser) error {
	_, err := d.MySql.Exec("DELETE FROM registration_queue WHERE id = $1",
		qUser.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d MySql) RegisterUser(user *concierge.User) error {
	_, err := d.MySql.Exec(
		"INSERT INTO users(ID, UserName, Enabled, TOTPSecret, Session, SessionTimeStamp, SessionMessageID, Peer, PeerPre, PeerPub, AllowedIPs, IP) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		&user.ID,
		&user.UserName,
		&user.Enabled,
		&user.TOTPSecret,
		&user.Session,
		&user.SessionTimeStamp,
		&user.SessionMessageID,
		&user.Peer,
		&user.PeerPre,
		&user.PeerPub,
		&user.AllowedIPs,
		&user.IP)
	if err != nil {
		return fmt.Errorf("db: insert into users: %w", err)
	}
	_, err = d.MySql.Exec(
		"DELETE FROM registration_queue WHERE ID = $1",
		&user.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d MySql) UnregisterUser(user *concierge.User) error {
	_, err := d.MySql.Exec("DELETE FROM users WHERE id = $1",
		user.ID)
	if err != nil {
		return fmt.Errorf("db: delete from registration_queue: %w", err)
	}
	return nil
}

func (d MySql) EnableUser(id *int64) error {
	_, err := d.MySql.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		1,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to enable user: %w", err)
	}
	return nil
}

func (d MySql) DisableUser(id *int64) error {
	_, err := d.MySql.Exec(
		"UPDATE users SET Enabled = $1 WHERE ID = $2",
		0,
		&id)
	if err != nil {
		return fmt.Errorf("db: failed to disable user: %w", err)
	}
	return nil
}

func (d MySql) SetAllowedIPs(id int64, val string) error {
	_, err := d.MySql.Exec("UPDATE users SET AllowedIPs = $1 WHERE ID = $2", val, id)
	if err != nil {
		return fmt.Errorf("db: failed to edit row: %w", err)
	}
	return nil
}

func (d MySql) SetIp(id int64, val string) error {
	_, err := d.MySql.Exec("UPDATE users SET IP = $1 WHERE ID = $2", val, id)
	if err != nil {
		return fmt.Errorf("db: failed to edit row: %w", err)
	}
	return nil
}

func (d MySql) SessionStarted(id int64, t time.Time, message *tele.Message) error {
	j, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("db: failed to marshal session started: %w", err)
	}
	_, err = d.MySql.Exec("UPDATE users SET Session = $1,SessionTimeStamp = $2,SessionMessageID = $3 WHERE id = $4", 1, t.Format(time.DateTime), string(j[:]), id)
	if err != nil {
		return fmt.Errorf("db: failed to set start session: %w", err)
	}
	return nil
}

func (d MySql) SessionEnded(id int64) error {
	_, err := d.MySql.Exec("UPDATE users SET Session = $1 WHERE id = $2", 0, id)
	if err != nil {
		return fmt.Errorf("db: failed to set start session: %w", err)
	}
	return nil
}

func (d MySql) UpdateMessage(message *tele.Message, id int64) error {
	j, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("db: failed to marshal update message: %w", err)
	}
	_, err = d.MySql.Exec("UPDATE users SET SessionMessageID = $1 WHERE id = $2", j, id)
	if err != nil {
		return fmt.Errorf("db: failed to set start session: %w", err)
	}
	return nil
}

func (d MySql) AddTimedEnable(id int64, t time.Time) error {
	_, err := d.MySql.Exec("INSERT INTO timed_enable(id, date) VALUES($1, $2)", id, t.Format("2006-01-02 15:04:05 Z0700"))
	if err != nil {
		return fmt.Errorf("db: failed to add timed enable: %w", err)
	}
	return nil
}

func (d MySql) GetTimedEnable() ([]concierge.TimedEnable, error) {
	rows, err := d.MySql.Query("SELECT * FROM timed_enable")
	if err != nil {
		return nil, fmt.Errorf("db: query failed: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var timedUsers []concierge.TimedEnable

	for rows.Next() {
		var (
			ID   int64
			Date string
		)
		err = rows.Scan(
			&ID,
			&Date)
		if err != nil {
			return nil, fmt.Errorf("db: failed to scan row: %w", err)
		}
		timedUsers = append(timedUsers, concierge.TimedEnable{
			ID:   ID,
			Date: Date})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows: %w", err)
	}

	return timedUsers, nil
}

func (d MySql) DelTimedEnable(id int64) error {
	_, err := d.MySql.Exec("DELETE FROM timed_enable WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("db: failed to del timed enable: %w", err)
	}
	return nil
}
