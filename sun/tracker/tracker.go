package tracker

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/storage"
)

const (
	statusConnected    string = "Connected"
	statusDisconnected        = "Disconnected"
	statusOpened              = "Opened"
	statusClosed              = "Closed"
	statusIdle                = "IDLE"
	statusWorking             = "Working"
	statusError               = "Err(%s)"
	oneWeek                   = time.Hour * 24 * 7
)

// Tracker tracks agent status, also tracks tunnel status, connections and traffic.
type Tracker struct {
	connMu  sync.Mutex
	connCnt map[string]int
	logger  *zap.SugaredLogger
	db      *storage.DB
}

func New(db *storage.DB) *Tracker {
	return &Tracker{
		logger:  log.New("traker[.]"),
		connCnt: map[string]int{},
		db:      db,
	}
}

func (t *Tracker) AgentTracker(uid, hash string) *AgentTracker {
	return &AgentTracker{
		root: t,
		uid:  uid,
		hash: hash,
	}
}

func (t *Tracker) updateAgentStatus(uid, ahash, status string) {
	_, err := t.db.UpdateAgent(uid, ahash, map[string]interface{}{"status": status})
	if err != nil {
		t.logger.Errorf("Update agent[%s] status to %s failed: %v", ahash, status, err)
	}
}

func (t *Tracker) agentConnected(uid, ahash string) {
	t.updateAgentStatus(uid, ahash, statusConnected)
}

func (t *Tracker) agentDelayed(uid, ahash string, d time.Duration) {
	delayed := fmt.Sprintf("%v", d)
	_, err := t.db.UpdateAgent(uid, ahash, map[string]interface{}{"delayed": delayed})
	if err != nil {
		t.logger.Errorf("Update agent[%s] delayed status to %s failed: %v", ahash, delayed, err)
	}
}

func (t *Tracker) agentDisconnected(uid, ahash string) {
	t.updateAgentStatus(uid, ahash, statusDisconnected)
}

func (t *Tracker) updateTunnelStatus(uid, ahash, thash, status string) {
	_, err := t.db.UpdateTunnel(uid, ahash, thash, map[string]interface{}{"status": status})
	if err != nil {
		t.logger.Errorf("Update tunnel[%s/%s] status to %s failed: %v", ahash, thash, status, err)
	}
}

func (t *Tracker) tunnelOpened(uid, ahash, thash string) {
	t.updateTunnelStatus(uid, ahash, thash, statusOpened)
}

func (t *Tracker) tunnelClosed(uid, ahash, thash string) {
	t.connMu.Lock()
	delete(t.connCnt, fmt.Sprintf("%s:%s", ahash, thash))
	t.connMu.Unlock()
	t.updateTunnelStatus(uid, ahash, thash, statusClosed)
}

func (t *Tracker) tunnelIsIdle(uid, ahash, thash string) {
	t.updateTunnelStatus(uid, ahash, thash, statusIdle)
}

func (t *Tracker) tunnelIsWorking(uid, ahash, thash string) {
	t.updateTunnelStatus(uid, ahash, thash, statusWorking)
}

func (t *Tracker) tunnelOnError(uid, ahash, thash, msg string) {
	t.updateTunnelStatus(uid, ahash, thash, fmt.Sprintf(statusError, msg))
}

func (t *Tracker) updateTunnelNumConn(uid, ahash, thash string, num int) {
	_, err := t.db.UpdateTunnel(uid, ahash, thash, map[string]interface{}{"num_conn": num})
	if err != nil {
		t.logger.Errorf("Update tunnel[%s/%s] connection number(%d) failed: %v", ahash, thash, num, err)
	}
}

func (t *Tracker) tunnelIncrConn(uid, ahash, thash string) {
	key := fmt.Sprintf("%s:%s", ahash, thash)

	t.connMu.Lock()
	cnt, ok := t.connCnt[key]
	if ok {
		cnt++
	} else {
		cnt = 1
	}
	t.connCnt[key] = cnt
	t.connMu.Unlock()

	if cnt == 1 {
		t.tunnelIsWorking(uid, ahash, thash)
	}
	t.updateTunnelNumConn(uid, ahash, thash, cnt)
}

func (t *Tracker) tunnelDecrConn(uid, ahash, thash string) {
	key := fmt.Sprintf("%s:%s", ahash, thash)

	t.connMu.Lock()
	cnt, ok := t.connCnt[key]
	if ok {
		util.Assert(cnt != 0, "connection count is equals to 0") // TODO(damnever): remove it
		cnt--
	}
	t.connCnt[key] = cnt
	t.connMu.Unlock()

	if cnt == 0 {
		t.tunnelIsIdle(uid, ahash, thash)
	}
	t.updateTunnelNumConn(uid, ahash, thash, cnt)
}

func (t *Tracker) tunnelRecordTraffic(uid, ahash, thash string, in, out int64) {
	tunnel, err := t.db.QueryTunnel(uid, ahash, thash)
	if err != nil {
		t.logger.Errorf("Query tunnel[%s/%s] failed: %v", ahash, thash, err)
		return
	}

	countAt := tunnel.CountAt
	// Reset traffic every week
	if tunnel.CountAt.IsZero() || time.Now().Sub(tunnel.CountAt) >= oneWeek {
		countAt = time.Now()
	} else {
		in += tunnel.TrafficIn
		out += tunnel.TrafficOut
	}
	_, err = t.db.UpdateTunnel(uid, ahash, thash, map[string]interface{}{
		"traffic_in":  in,
		"traffic_out": out,
		"count_at":    countAt,
	})
	if err != nil {
		t.logger.Errorf("Update tunnel[%s/%s] traffic(%d|%d) failed: %v", ahash, thash, in, out, err)
	}
}

type AgentTracker struct {
	root *Tracker
	uid  string
	hash string
}

func (at *AgentTracker) TunnelTracker(hash string) *TunnelTracker {
	return &TunnelTracker{
		root:  at.root,
		uid:   at.uid,
		ahash: at.hash,
		hash:  hash,
	}
}
func (tt *AgentTracker) UID() string {
	return tt.uid
}

func (tt *AgentTracker) Hash() string {
	return tt.hash
}

func (at *AgentTracker) Connected() {
	at.root.agentConnected(at.uid, at.hash)
}

func (at *AgentTracker) Delayed(d time.Duration) {
	at.root.agentDelayed(at.uid, at.hash, d)
}

func (at *AgentTracker) Disconnected() {
	at.root.agentDisconnected(at.uid, at.hash)
}

type TunnelTracker struct {
	root  *Tracker
	uid   string
	ahash string
	hash  string
}

func (tt *TunnelTracker) UID() string {
	return tt.uid
}

func (tt *TunnelTracker) AgentHash() string {
	return tt.ahash
}

func (tt *TunnelTracker) Hash() string {
	return tt.hash
}

func (tt *TunnelTracker) Opened() {
	tt.root.tunnelOpened(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) Closed() {
	tt.root.tunnelClosed(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) IsIdle() {
	tt.root.tunnelIsIdle(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) IsWorking() {
	tt.root.tunnelIsWorking(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) OnError(msg string) {
	tt.root.tunnelOnError(tt.uid, tt.ahash, tt.hash, msg)
}

func (tt *TunnelTracker) IncrConn() {
	tt.root.tunnelIncrConn(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) DecrConn() {
	tt.root.tunnelDecrConn(tt.uid, tt.ahash, tt.hash)
}

func (tt *TunnelTracker) RecordTraffic(in, out int64) {
	tt.root.tunnelRecordTraffic(tt.uid, tt.ahash, tt.hash, in, out)
}
