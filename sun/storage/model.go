package storage

import (
	"encoding/json"
	"strings"
	"time"
)

// TODO(damnever):
//   - Store tunnel stats in other place

const (
	timeFormat = "15:04:05 01/02/2006"
)

type User struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Password  string    `json:"password" db:"password"`
	Email     string    `json:"email" db:"email"`
	IsAdmin   bool      `json:"is_admin" db:"is_admin"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type UserForJSON User // Use alias to avoid infinite recursive.

func (user User) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		UserForJSON
		Password  string `json:"password"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		UserForJSON: UserForJSON(user),
		Password:    strings.Repeat("*", 9),
		CreatedAt:   user.CreatedAt.Local().Format(timeFormat),
		UpdatedAt:   user.UpdatedAt.Local().Format(timeFormat),
	})
}

type Agent struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Hash      string    `json:"hash" db:"hash"`
	Device    string    `json:"device" db:"device"` // os, kernel, arch
	Version   string    `json:"version" db:"version"`
	Status    string    `json:"status" db:"status"`
	Delayed   string    `json:"delayed" db:"delayed"`
	Tag       string    `json:"tag" db:"tag"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	Tunnels   []Tunnel  `json:"tunnels,omitempty" db:"-"`
}

type AgentForJSON Agent // Use alias to avoid infinite recursive.

func (agent Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		AgentForJSON
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		AgentForJSON: AgentForJSON(agent),
		CreatedAt:    agent.CreatedAt.Local().Format(timeFormat),
		UpdatedAt:    agent.UpdatedAt.Local().Format(timeFormat),
	})
}

type Tunnel struct {
	ID         int       `json:"id" db:"id"`
	AgentID    int       `json:"agent_id" db:"agent_id"`
	Hash       string    `json:"hash" db:"hash"`
	Proto      string    `json:"proto" db:"proto"`
	ExportAddr string    `json:"export_addr" db:"export_addr"`
	ServerAddr string    `json:"server_addr" db:"server_addr"`
	Status     string    `json:"status" db:"status"`
	NumConn    int       `json:"num_conn" db:"num_conn"`
	TrafficIn  int64     `json:"traffic_in" db:"traffic_in"`
	TrafficOut int64     `json:"traffic_out" db:"traffic_out"`
	CountAt    time.Time `json:"count_at" db:"count_at"`
	Enabled    bool      `json:"enabled" db:"enabled"`
	Tag        string    `json:"tag" db:"tag"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type TunnelForJSON Tunnel // Use alias to avoid infinite recursive.

func (tunnel Tunnel) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		TunnelForJSON
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}{
		TunnelForJSON: TunnelForJSON(tunnel),
		CreatedAt:     tunnel.CreatedAt.Local().Format(timeFormat),
		UpdatedAt:     tunnel.UpdatedAt.Local().Format(timeFormat),
	})
}

var sqlToInitDB = `
PRAGMA encoding="UTF-8";

CREATE TABLE user (
	id INTEGER PRIMARY KEY,
	name VARCHAR(23) NOT NULL DEFAULT "",
	password VARCHAR(60) NOT NULL DEFAULT "",
	email VARCHAR(50) NOT NULL DEFAULT "",
	is_admin TINYINT(1) NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent (
	id INTEGER PRIMARY KEY,
	user_id BIGINT NOT NULL DEFAULT -1,
	hash VARCHAR(8) NOT NULL DEFAULT "",
	device VARCHAR(255) NOT NULL DEFAULT "UNKNOWN",
	version VARCHAR(25) NOT NULL DEFAULT "UNKNOWN",
	status TEXT NOT NULL DEFAULT "UNKNOWN",
	delayed VARCHAR(12) NOT NULL DEFAULT "0ms",
	tag VARCHAR(255) NOT NULL DEFAULT "",
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT unique_agent_hash UNIQUE (hash) ON CONFLICT ABORT
);

CREATE TABLE tunnel (
	id INTEGER PRIMARY KEY,
	agent_id BIGINT NOT NULL DEFAULT -1,
	hash VARCHAR(8) NOT NULL DEFAULT "",
	proto VARCHAR(10) NOT NULL DEFAULT "",
	export_addr VARCHAR(255) NOT NULL DEFAULT "",
	server_addr VARCHAR(255) NOT NULL DEFAULT "",
	status TEXT NOT NULL DEFAULT "UNKNOWN",
	num_conn INTEGER NOT NULL DEFAULT 0,
	traffic_in BIGINT NOT NULL DEFAULT 0,
	traffic_out BIGINT NOT NULL DEFAULT 0,
	count_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	tag VARCHAR(255) NOT NULL DEFAULT "",
	enabled TINYINT(1) NOT NULL DEFAULT 1,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

	CONSTRAINT unique_tunnel_addr UNIQUE (proto, server_addr) ON CONFLICT ABORT
);

-- on update feature..
CREATE TRIGGER user_update_trigger AFTER UPDATE ON user
	BEGIN
		UPDATE agent SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
	END;

CREATE TRIGGER agent_update_trigger AFTER UPDATE ON agent
	BEGIN
		UPDATE agent SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
	END;

CREATE TRIGGER tunnel_update_trigger AFTER UPDATE ON tunnel
	BEGIN
		UPDATE tunnel SET updated_at=CURRENT_TIMESTAMP WHERE id=NEW.id;
	END;

-- indexes
CREATE UNIQUE INDEX idx_user_name ON user (name);
CREATE UNIQUE INDEX idx_agent_hash_user_id ON agent (user_id, hash);
CREATE UNIQUE INDEX idx_tunnel_hash_agent_id ON tunnel (agent_id, hash);
`
