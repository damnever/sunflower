package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/damnever/sunflower/pkg/util"
)

type DB struct {
	firstInit bool
	*sqlx.DB
}

func New(datadir string) (*DB, error) {
	if err := os.MkdirAll(datadir, 0750); err != nil {
		return nil, err
	}
	fpath := filepath.Join(datadir, "sqlite3.db")
	needInitDB := !util.FileExist(fpath)

	db, err := sqlx.Open("sqlite3", fpath)
	if err != nil {
		return nil, err
	}
	if needInitDB {
		if _, err = db.Exec(sqlToInitDB); err != nil {
			db.Close()
			return nil, err
		}
	}
	return &DB{
		DB:        db,
		firstInit: needInitDB,
	}, nil
}

func (db *DB) IsEmpty() (bool, error) {
	if db.firstInit {
		return true, nil
	}
	n, err := db.QueryUserCount()
	return (err == nil && n == 0), err
}

func (db *DB) QueryUsers() ([]User, error) {
	rows, err := db.Queryx("SELECT * FROM user")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// The slice is a pointer, no copy made no matter what the value type is.
	users := []User{}
	for rows.Next() {
		var user User
		if err = rows.StructScan(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

func (db *DB) QueryUserCount() (n int, err error) {
	row := db.QueryRowx("SELECT COUNT(id) FROM user")
	err = row.Scan(&n)
	return
}

func (db *DB) QueryUser(name string) (User, error) {
	var user User
	row := db.QueryRowx("SELECT * FROM user WHERE name=?", name)
	err := row.StructScan(&user)
	return user, err
}

func (db *DB) CreateUser(name, password, email string, isAdmin bool) error {
	sql := "INSERT INTO user (name, password, email, is_admin) VALUES (?, ?, ?, ?)"
	_, err := db.Exec(sql, name, password, email, isAdmin)
	return err
}

func (db *DB) UpdateUser(name string, args map[string]interface{}) (bool, error) {
	columns, values := buildQFromMap(args)
	q := fmt.Sprintf("UPDATE user SET %s WHERE name=?", columns)
	res, err := db.Exec(q, append(values, name)...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n != 0, nil
}

func (db *DB) DeleteUser(name string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	row := tx.QueryRowx("SELECT id FROM user where name=?", name)
	if err := row.Scan(&userID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM user WHERE id=?", userID); err != nil {
		return err
	}
	if err := db.deleteAgentsByTxUserID(tx, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) QueryAgentHashs(username string) ([]string, error) {
	sql := "SELECT hash FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?)"
	rows, err := db.Queryx(sql, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hashs := []string{}
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return nil, err
		}
		hashs = append(hashs, hash)
	}
	return hashs, rows.Err()
}

func (db *DB) QueryAgents(username string) ([]Agent, error) {
	sql := "SELECT * FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?)"
	rows, err := db.Queryx(sql, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []Agent{}
	for rows.Next() {
		var agent Agent
		if err = rows.StructScan(&agent); err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	return agents, rows.Err()
}

func (db *DB) QueryAgentCount(username string) (n int, err error) {
	sql := "SELECT COUNT(id) FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?)"
	row := db.QueryRowx(sql, username)
	err = row.Scan(&n)
	return
}

func (db *DB) DeleteAgents(username string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var userID int64
	row := tx.QueryRowx("SELECT id FROM user where name=?", username)
	if err := row.Scan(&userID); err != nil {
		return err
	}
	if err := db.deleteAgentsByTxUserID(tx, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) deleteAgentsByTxUserID(tx *sqlx.Tx, userID int64) error {
	row := tx.QueryRowx("SELECT id FROM agent WHERE user_id=?", userID)
	agentIDs, err := row.SliceScan()
	if err != nil {
		if IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err = tx.Exec("DELETE FROM agent WHERE user_id=?", userID); err != nil {
		return err
	}

	stmt, err := tx.Preparex("DELETE FROM tunnel WHERE agent_id=?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, id := range agentIDs {
		if _, err := stmt.Exec(id); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) QueryAgent(username, hash string) (Agent, error) {
	sql := "SELECT * FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?) AND hash=?"
	var agent Agent
	row := db.QueryRowx(sql, username, hash)
	err := row.StructScan(&agent)
	return agent, err
}

func (db *DB) CreateAgent(username, hash, tag string) error {
	sql := "INSERT INTO agent (user_id, hash, tag) VALUES ((SELECT id FROM user WHERE name=?), ?, ?)"
	_, err := db.Exec(sql, username, hash, tag)
	return err
}

func (db *DB) UpdateAgent(username, hash string, args map[string]interface{}) (bool, error) {
	columns, values := buildQFromMap(args)
	q := fmt.Sprintf("UPDATE agent SET %s WHERE user_id=(SELECT id FROM user WHERE name=?) AND hash=?", columns)
	res, err := db.Exec(q, append(values, username, hash)...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n != 0, nil
}

func (db *DB) DeleteAgent(username, hash string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var agentID int64
	sqlAID := "SELECT id FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?) AND hash=?"
	row := tx.QueryRowx(sqlAID, username, hash)
	if err := row.Scan(&agentID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM agent WHERE id=?", agentID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM tunnel WHERE agent_id=?", agentID); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) QueryTunnels(username, ahash string) ([]Tunnel, error) {
	sql := `SELECT * FROM tunnel WHERE
	agent_id=(SELECT id FROM agent WHERE
	user_id=(SELECT id FROM user WHERE name=?)
	AND hash=?)`
	rows, err := db.Queryx(sql, username, ahash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tunnels := []Tunnel{}
	for rows.Next() {
		var tunnel Tunnel
		if err = rows.StructScan(&tunnel); err != nil {
			return nil, err
		}
		tunnels = append(tunnels, tunnel)
	}
	return tunnels, rows.Err()
}

func (db *DB) QueryTunnelCount(username, ahash string) (n int, err error) {
	sql := `SELECT COUNT(id) FROM tunnel WHERE
	agent_id=(SELECT id FROM agent WHERE
	user_id=(SELECT id FROM user WHERE name=?)
	AND hash=?)`
	row := db.QueryRowx(sql, username, ahash)
	err = row.Scan(&n)
	return
}

func (db *DB) DeleteTunnels(username, ahash string) error {
	sql := `DELETE FROM tunnel WHERE
	agent_id=(SELECT id FROM agent WHERE
	user_id=(SELECT id FROM user WHERE name=?)
	AND hash=?)`
	_, err := db.Exec(sql, username, ahash)
	return err
}

func (db *DB) QueryTunnel(username, ahash, hash string) (Tunnel, error) {
	sql := `SELECT * FROM tunnel WHERE
	agent_id=(SELECT id FROM agent WHERE
	user_id=(SELECT id FROM user WHERE name=?)
	AND hash=?) AND hash=?`
	var tunnel Tunnel
	row := db.QueryRowx(sql, username, ahash, hash)
	err := row.StructScan(&tunnel)
	return tunnel, err
}

func (db *DB) CreateTunnel(username, ahash, hash, proto, exportAddr, serverAddr, tag string) error {
	sql := `INSERT INTO tunnel (agent_id, hash, proto, export_addr, server_addr, tag)
	VALUES ((SELECT id FROM agent WHERE user_id=(SELECT id FROM user WHERE name=?) AND hash=?),
	?, ?, ?, ?, ?)`
	_, err := db.Exec(sql, username, ahash, hash, proto, exportAddr, serverAddr, tag)
	return err
}

func (db *DB) UpdateTunnel(username, ahash, hash string, args map[string]interface{}) (bool, error) {
	columns, values := buildQFromMap(args)
	sql := `UPDATE tunnel SET %s WHERE
	agent_id=(SELECT id FROM agent WHERE user_id=
	(SELECT id FROM user WHERE name=?) AND hash=?)
	AND hash=?`
	sql = fmt.Sprintf(sql, columns)
	res, err := db.Exec(sql, append(values, username, ahash, hash)...)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n != 0, nil
}

func (db *DB) DeleteTunnel(username, ahash, hash string) error {
	sql := `DELETE FROM tunnel WHERE
	agent_id=(SELECT id FROM agent WHERE user_id=
	(SELECT id FROM user WHERE name=?) AND hash=?)
	AND hash=?`
	_, err := db.Exec(sql, username, ahash, hash)
	return err
}

func buildQFromMap(m map[string]interface{}) (string, []interface{}) {
	n := len(m)
	columns, values := make([]string, n), make([]interface{}, n)
	i := 0
	for c, v := range m {
		columns[i] = fmt.Sprintf("%s=?", c)
		values[i] = v
		i++
	}
	return strings.Join(columns, ","), values
}
