package web

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/mholt/archiver"
	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/bufpool"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/version"
)

// FIXME(damnever): shiiiit code..

type supervisorConfig struct {
	name string
	data []byte
	size int64
}

func (f *supervisorConfig) Name() string       { return f.name }
func (f *supervisorConfig) Size() int64        { return f.size }
func (f *supervisorConfig) Mode() os.FileMode  { return 0664 }
func (f *supervisorConfig) ModTime() time.Time { return time.Now() }
func (f *supervisorConfig) IsDir() bool        { return false }
func (f *supervisorConfig) Sys() interface{}   { return nil }

var (
	systemd     = "systemd"
	supervisord = "supervisord"
	systemdConf = []byte(`# Edit path and place it in /etc/systemd/system
# sudo systemctl enable flower.service
# sudo systemctl start flower.service

[Unit]
Description=Sunflower Client
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
ExecStart=/path/to/flower # NOTE: EDIT IT
ExecStop=/bin/kill -TERM $MAINPID
TimeoutStopSec=10
Restart=always

[Install]
WantedBy=multi-user.target`)
	supervisordConf = []byte(`# Reference: http://supervisord.org/configuration.html
[program:flower]
command=/path/to/flower # NOTE: EDIT IT
autostart=true
autorestart=true
startsecs=3
startretries=3
stopsignal=TERM
stopwaitsecs=10
user=www-data
stdout_logfile=/var/log/flower/stdout.log  # NOTE: mkdir /var/log/flower
stdout_logfile_maxbytes=2MB
stdout_logfile_backups=5
stderr_logfile=/var/log/flower/stderr.log
stderr_logfile_maxbytes=2MB
stderr_logfile_backups=5
`)
	supervisorConfigs = map[string]*supervisorConfig{
		systemd:     &supervisorConfig{name: "flower.service", data: systemdConf, size: int64(len(systemdConf))},
		supervisord: &supervisorConfig{name: "flower.conf", data: supervisordConf, size: int64(len(supervisordConf))},
	}
	tmpDir  = filepath.Join(util.TempDir(), "sunflower")
	goPath  = filepath.Join(tmpDir, version.Full())
	pkgPath = filepath.Join(goPath, "src/github.com/damnever/sunflower")

	errUnknown    = errors.New("building process has problem or platform is not supported")
	errIsBuilding = errors.New("is building, please wait for a minute")
)

type platform struct {
	GOOS   string
	GOARCH string
	GOARM  string
	Ext    string
}

func (p platform) String() string {
	return fmtPlatform(p.GOOS, p.GOARCH, p.GOARM)
}

func fmtPlatform(os, arch, arm string) string {
	return fmt.Sprintf("%s/%s%s", os, arch, arm)
}

// https://golang.org/doc/install/source#introduction
// https://github.com/golang/go/wiki/GoArm
// The order is the priorities, armv4 is not supported.
// TODO(damnever): make it configurable
var platforms = []platform{
	platform{GOOS: "darwin", GOARCH: "amd64"},
	platform{GOOS: "linux", GOARCH: "amd64"},
	platform{GOOS: "windows", GOARCH: "amd64", Ext: ".exe"},
	platform{GOOS: "linux", GOARCH: "arm", GOARM: "7"}, // armv7
	platform{GOOS: "darwin", GOARCH: "386"},
	platform{GOOS: "linux", GOARCH: "386"},
	platform{GOOS: "windows", GOARCH: "386", Ext: ".exe"},
	platform{GOOS: "linux", GOARCH: "arm", GOARM: "6"}, // armv6
	platform{GOOS: "linux", GOARCH: "arm64"},           // armv8
	platform{GOOS: "linux", GOARCH: "arm", GOARM: "5"}, // armv5
	platform{GOOS: "linux", GOARCH: "mips64"},
	platform{GOOS: "linux", GOARCH: "mips64le"},
	platform{GOOS: "linux", GOARCH: "mips"},
	platform{GOOS: "linux", GOARCH: "mipsle"},
}

type Builder struct {
	logger      *zap.SugaredLogger
	done        int32
	ctx         context.Context
	bindir      string
	agentConfig string
	Cancel      func()
}

func NewBuilder(datadir string, agentConfig string) (*Builder, error) {
	if err := os.RemoveAll(tmpDir); err != nil {
		return nil, err
	}
	bindir := filepath.Join(datadir, "bin")
	if err := os.RemoveAll(bindir); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Builder{
		logger:      log.New("web[b]"),
		bindir:      bindir,
		agentConfig: agentConfig,
		done:        0,
		ctx:         ctx,
		Cancel:      cancel,
	}, nil
}

func (b *Builder) TryGetPkg(username, ahash, os, arch, arm, supervisorType string) ([]byte, error) {
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}
	binName := "flower" + ext
	binPath := filepath.Join(b.bindir, fmtPlatform(os, arch, arm), binName)

	if !util.FileExist(binPath) {
		if atomic.LoadInt32(&b.done) == int32(1) {
			return nil, errUnknown
		}
		return nil, errIsBuilding
	}

	agentConf := fmt.Sprintf("id: %s\nhash: %s\n%s", username, ahash, b.agentConfig)
	return zipBin(agentConf, supervisorType, binPath, binName)
}

func (b *Builder) StartCrossPlatformBuild() {
	b.logger.Info("Start cross platform build")

	for _, platform := range platforms {
		err := b.makeFlower(platform)
		if err == nil {
			b.logger.Infof("Build %s success", platform.String())
			continue
		}
		if err == context.Canceled || err == context.DeadlineExceeded {
			break
		}
		b.logger.Errorf("Build %s failed: %v", platform.String(), err)
	}

	atomic.StoreInt32(&b.done, 1)
	if err := os.RemoveAll(goPath); err != nil {
		b.logger.Errorf("Remove %s failed: %v", goPath, err)
	}
	b.logger.Info("Cross platform build done")
}

func (b *Builder) makeFlower(p platform) error {
	if err := tryRestorePkgPath(); err != nil {
		return err
	}
	binDir := filepath.Join(b.bindir, p.String())
	if err := os.MkdirAll(binDir, 0750); err != nil && !os.IsExist(err) {
		return err
	}

	buidlCmd := fmt.Sprintf("cd %s && go build -o '%s/flower%s' ./cmd/flower", pkgPath, binDir, p.Ext)
	cmd := exec.CommandContext(b.ctx, "/bin/bash", "-c", buidlCmd)
	cmd.Env = append(
		os.Environ(),
		"GOPATH="+goPath,
		"GOOS="+p.GOOS,
		"GOARCH="+p.GOARCH,
	)
	if p.GOARM != "" {
		cmd.Env = append(cmd.Env, "GOARM="+p.GOARM)
	}

	out := bufpool.Get()
	defer bufpool.Put(out)
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, out.String())
	}
	return nil
}

func tryRestorePkgPath() error {
	if util.FileExist(pkgPath) {
		return nil
	}
	os.RemoveAll(tmpDir)
	if err := RestoreAsset(pkgPath, "flower.zip"); err != nil {
		return err
	}
	return archiver.Zip.Open(filepath.Join(pkgPath, "flower.zip"), pkgPath)
}

func zipBin(agentConf, supervisorType, binPath, binName string) ([]byte, error) {
	/* zipBuf := bufpool.Get() */
	/* defer bufpool.Put(zipBuf) */
	// TODO(damnever): resuse the big buffer
	zipBuf := new(bytes.Buffer)
	// Zip writer for executable
	zipw := zip.NewWriter(zipBuf)

	// Supervisor config
	sconf := supervisorConfigs[supervisorType]
	scheader, err := zip.FileInfoHeader(sconf)
	if err != nil {
		return nil, err
	}
	scheader.Method = zip.Deflate
	dconfw, err := zipw.CreateHeader(scheader)
	if _, err := dconfw.Write(sconf.data); err != nil {
		return nil, err
	}

	// Copy executable data into zip file
	binf, err := os.Open(binPath)
	if err != nil {
		return nil, err
	}
	defer binf.Close()
	bfInfo, err := binf.Stat()
	if err != nil {
		return nil, err
	}
	bheader, err := zip.FileInfoHeader(bfInfo)
	if err != nil {
		return nil, err
	}
	bheader.Method = zip.Deflate
	binw, err := zipw.CreateHeader(bheader)
	if err != nil {
		return nil, err
	}
	tmpBuf := bufpool.GrowGet(32768) // 32 * 1024
	defer bufpool.Put(tmpBuf)
	if _, err := io.CopyBuffer(binw, binf, tmpBuf.Bytes()[:32768]); err != nil {
		return nil, err
	}

	// Compress agent config data
	confBuf := bufpool.Get()
	defer bufpool.Put(confBuf)
	aconfZipW := zip.NewWriter(confBuf)
	aconff, err := aconfZipW.Create("flower.yaml")
	if err != nil {
		return nil, err
	}
	if _, err := aconff.Write([]byte(agentConf)); err != nil {
		return nil, err
	}
	if err := aconfZipW.Close(); err != nil {
		return nil, err
	}

	// attach config.zip at the end of executable data
	if _, err := binw.Write(confBuf.Bytes()); err != nil {
		return nil, err
	}
	if err := zipw.Close(); err != nil {
		return nil, err
	}
	return zipBuf.Bytes(), nil
}

type supervisorConfigFileInfo struct {
	typ string
}
