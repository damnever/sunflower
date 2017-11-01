package web

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"

	"github.com/mholt/archiver"
	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/bufpool"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/version"
)

var (
	tmpDir   = filepath.Join(util.TempDir(), "sunflower")
	goPath   = filepath.Join(tmpDir, version.Full())
	pkgPath  = filepath.Join(goPath, "src/github.com/damnever/sunflower")
	mainPath = filepath.Join(pkgPath, "")

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
}

type Builder struct {
	logger      *zap.SugaredLogger
	done        int32
	ctx         context.Context
	agentConfig string
	Cancel      func()
}

func NewBuilder(agentConfig string) (*Builder, error) {
	if err := os.RemoveAll(tmpDir); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Builder{
		logger:      log.New("web[b]"),
		agentConfig: agentConfig,
		done:        0,
		ctx:         ctx,
		Cancel:      cancel,
	}, nil
}

func (b *Builder) TryGetPkg(username, ahash, os, arch, arm string) ([]byte, error) {
	ext := ""
	if os == "windows" {
		ext = ".exe"
	}
	binName := "flower" + ext
	binPath := filepath.Join(pkgPath, "bin", fmtPlatform(os, arch, arm), binName)

	if !util.FileExist(binPath) {
		if atomic.LoadInt32(&b.done) == int32(1) {
			return nil, errUnknown
		}
		return nil, errIsBuilding
	}

	confData := fmt.Sprintf("id: %s\nhash: %s\n%s", username, ahash, b.agentConfig)
	return zipBin(confData, binPath, binName)
}

func (b *Builder) StartCrossPlatformBuild() {
	b.logger.Info("Start cross platform build")

	for _, platform := range platforms {
		err := makeFlower(b.ctx, platform)
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
	b.logger.Info("Cross platform build done")
}

func makeFlower(ctx context.Context, p platform) error {
	if err := tryRestorePkgPath(); err != nil {
		return err
	}
	binDir := filepath.Join(pkgPath, "bin", p.String())
	if err := os.MkdirAll(binDir, 0750); err != nil && !os.IsExist(err) {
		return err
	}

	buidlCmd := fmt.Sprintf("cd %s && go build -o '%s/flower%s' ./cmd/flower", pkgPath, binDir, p.Ext)
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", buidlCmd)
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

func zipBin(confData, binPath, binName string) ([]byte, error) {
	zipBuf := bufpool.Get()
	defer bufpool.Put(zipBuf)
	// Zip writer for executable
	zipw := zip.NewWriter(zipBuf)

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
	binw, err := zipw.CreateHeader(bheader)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(binw, binf); err != nil {
		return nil, err
	}

	// Compress config data
	confBuf := bufpool.Get()
	defer bufpool.Put(confBuf)
	confZipW := zip.NewWriter(confBuf)
	conff, err := confZipW.Create("flower.yaml")
	if err != nil {
		return nil, err
	}
	if _, err := conff.Write([]byte(confData)); err != nil {
		return nil, err
	}
	if err := confZipW.Close(); err != nil {
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

/*
type binFile struct {
	name string
	size int64
}

func (b binFile) Name() string       { return b.name }
func (b binFile) Size() int64        { return b.size }
func (b binFile) Mode() os.FileMode  { return 0755 }
func (b binFile) ModTime() time.Time { return time.Now().Local() }
func (b binFile) IsDir() bool        { return false }
func (b binFile) Sys() interface{}   { return nil }
*/
