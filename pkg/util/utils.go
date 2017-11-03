package util

import (
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type TimeoutConfig struct {
	Connect time.Duration
	Read    time.Duration
	Write   time.Duration
}

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Assert(ok bool, msg string) {
	if !ok {
		panic(msg)
	}
}

func FileExist(fpath string) bool {
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		return false
	}
	return true
}

func CopyFile(src, dest string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dest, data, 0777)
}

func Hash(args ...string) string {
	s := strings.Join(append(args, RandString(14)), "-")
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

func RandString(n int) string {
	b := make([]byte, n*2)
	crand.Read(b)
	s := base64.URLEncoding.EncodeToString(b)
	return s[0:n]
}

func Host(req *http.Request) string {
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		host = req.Host
	}
	return strings.ToLower(host)
}

// Taken from https://play.golang.org/p/BDt3qEQ_2H
func HostIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			return ip.String(), nil
		}
	}
	return "localhost", nil
}

func WatchSignals() <-chan os.Signal {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT)
	return sigCh
}

func EncryptPasswd(passwd []byte) (password string, err error) {
	passwd, err = bcrypt.GenerateFromPassword(passwd, bcrypt.DefaultCost)
	if err != nil {
		return
	}
	password = string(passwd)
	return
}

func TempDir() string {
	if runtime.GOOS != "windows" {
		return "/tmp"
	}
	return os.TempDir()
}
