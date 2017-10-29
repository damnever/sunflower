package web

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	minUsernameLen = 6
	maxUsernameLen = 12
	minPasswordLen = 8
	maxPasswordLen = 20
)

var (
	digitOnlyRe = regexp.MustCompile("^[0-9]+$")
	strOnlyRe   = regexp.MustCompile("^[a-zA-Z\\|!@#`\\$%\\^&\\*\\-+=,\\._:;\"'?]+$")
	emailRe     = regexp.MustCompile("^([a-zA-Z0-9_\\._-]+)@([a-zA-Z0-9\\.-]+)\\.([a-zA-Z\\.]+)$")
)

// TODO(damnever): reserved usernames?

func ValidateUsername(username string) error {
	if n := len(username); n < minUsernameLen || n > maxUsernameLen {
		return fmt.Errorf("username requires [%d, %d] characters", minUsernameLen, maxUsernameLen)
	}
	return nil
}

func ValidatePassword(password string) error {
	if n := len(password); n < minPasswordLen || n > maxPasswordLen {
		return fmt.Errorf("password requires [%d, %d] characters", minPasswordLen, maxPasswordLen)
	}
	if digitOnlyRe.MatchString(password) || strOnlyRe.MatchString(password) {
		return fmt.Errorf("password must contain both digital and character")
	}
	return nil
}

func ValidateEmail(email string) error {
	if !emailRe.MatchString(email) {
		return fmt.Errorf("bad email address")
	}
	return nil
}

func ValidateTag(tag string) error {
	if strings.Contains(tag, "|") {
		return fmt.Errorf("tag cannot contain character '|'")
	}
	return nil
}

var supportedProtos = map[string]bool{
	"HTTP": true,
	"TCP":  true,
}

func ValidteProtocol(proto string) error {
	if supportedProtos[proto] {
		return nil
	}
	return fmt.Errorf("unsupported protocol %s", proto)
}

func ValidateAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return err
	}
	if n < 1024 || n > 65535 {
		return fmt.Errorf("port(%d) must range in [1024, 65535]", n)
	}
	if host == "localhost" {
		return nil
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("invalid ip address or host name: %s", host)
	}
	return nil
}

type counter struct {
	sync.Mutex
	max      int
	counters map[string]*counterItem
}

func newCounster(max int) *counter {
	return &counter{max: max}
}

func (c *counter) Incr(names ...string) bool {
	name := strings.Join(names, "/")
	c.Lock()
	defer c.Unlock()

	item, in := c.counters[name]
	if !in {
		item = newCounterItem(c.max)
		c.counters[name] = item
	}
	return item.Incr()
}

type counterItem struct {
	max       int
	count     int
	countTime time.Time
}

func newCounterItem(max int) *counterItem {
	return &counterItem{
		max:   max,
		count: 0,
	}
}

func (c *counterItem) Incr() bool {
	if c.count == 0 {
		c.countTime = time.Now()
		c.count = 1
		return true
	}

	if time.Now().Sub(c.countTime).Hours() < float64(1) {
		c.count += 1
		if c.count >= c.max {
			return false
		}
	} else {
		c.countTime = time.Now()
		c.count = 1
	}
	return true
}
