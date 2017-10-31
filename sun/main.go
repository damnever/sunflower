package sun

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/damnever/cc"
	"go.uber.org/zap"

	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/debug"
	"github.com/damnever/sunflower/pkg/input"
	"github.com/damnever/sunflower/pkg/open"
	"github.com/damnever/sunflower/pkg/util"
	"github.com/damnever/sunflower/sun/pubsub"
	"github.com/damnever/sunflower/sun/storage"
	"github.com/damnever/sunflower/sun/web"
	"github.com/damnever/sunflower/version"
)

var (
	a = flag.Bool("a", false, "Add an administrator.")
	b = flag.Bool("b", false, "Open control panel in browser, for quick start.")
	c = flag.String("c", "sun.server.yaml", "Path to server configuration file.")
)

func Run() {
	flag.Parse()
	logger := log.New("M")

	ip, err := util.HostIP()
	if err != nil {
		logger.Fatalf("Resolve host IP failed: %v", err)
	}
	cconf, err := cc.NewConfigFromFile(*c)
	if err != nil {
		logger.Fatalf("Load config failed: %v", err)
	}
	cconf.Set("host_ip", ip)
	debugAddr := cconf.String("debug_addr")
	datadir := cconf.String("datadir")
	coreconf := buildCoreConfig(cconf)
	_, port, err := net.SplitHostPort(coreconf.RPCConf.ListenAddr)
	if err != nil {
		logger.Fatal("Parse control address failed: %v", err)
	}
	webconf := buildWebConfig(port, cconf)
	cconf = nil

	fatalF := func(err error, ignoreEOF bool, format string, args ...interface{}) {
		if err != nil {
			if err == io.EOF {
				os.Exit(1)
			}
			logger.Fatalf(format+": %+v", append(args, err)...)
		}
	}
	tryStartDebugServer(debugAddr, logger)

	// Init db
	db, err := storage.New(datadir)
	fatalF(err, false, "Init database failed")
	fatalF(tryAddAdmin(db), true, "Unexpected error")

	ps := pubsub.New()
	errCh := make(chan error, 2)

	ctls, err := NewCtlServer(coreconf, ps, db)
	fatalF(err, false, "Init core server failed")
	go func() { errCh <- ctls.Run() }()

	webserver, err := web.New(webconf, db, ps)
	fatalF(err, false, "Init web server failed")
	go func() { errCh <- webserver.Serve() }()

	logger.Infof("The sun(%s) has risen, bring out your flowers.", version.Full())
	openInBrowser(webserver.Endpoint(), logger)

	sigCh := util.WatchSignals()
	select {
	case err := <-errCh:
		if err != nil {
			logger.Errorf("Got error: %v", err)
		}
	case sig := <-sigCh:
		logger.Infof("Got signal: %v", sig)
	}

	webserver.Close()
	ctls.GracefulShutdown()
}

func tryAddAdmin(db *storage.DB) error {
	if !*a {
		// XXX(damnever): count admin users? sonmeone may do...
		isEmpty, err := db.IsEmpty()
		if err != nil {
			return err
		}
		if !isEmpty {
			return nil
		}
		*a = true
		fmt.Println("You need an administrator!")
	} else {
		fmt.Println("Add an administrator.")
	}
	if *a {
		return addAdmin(db)
	}
	return nil
}

func addAdmin(db *storage.DB) error {
INPUT_USERNAME:
	username, err := input.Readln("> Username: ")
	if err != nil {
		return err
	}
	if err := web.ValidateUsername(username); err != nil {
		fmt.Printf("Try again: %s.\n", err.Error())
		goto INPUT_USERNAME
	}

INPUT_EMAIL:
	email, err := input.Readln("> E-mail: ")
	if err != nil {
		return err
	}
	if err := web.ValidateEmail(email); err != nil {
		fmt.Printf("Try again: %s.\n", err.Error())
		goto INPUT_EMAIL
	}

INPUT_PASSWORD:
	password, err := input.GetPasswd("> Password: ")
	if err != nil {
		return err
	}
	if err := web.ValidatePassword(password); err != nil {
		fmt.Printf("Try again: %s.\n", err.Error())
		goto INPUT_PASSWORD
	}

	password2, err := input.GetPasswd("> Password(confirm): ")
	if err != nil {
		return err
	}
	if password2 != password {
		fmt.Println("Try again: two passwords doesn't match.")
		goto INPUT_PASSWORD
	}

	password, err = util.EncryptPasswd([]byte(password))
	if err != nil {
		return err
	}
	err = db.CreateUser(username, password, email, true)
	if err != nil {
		if storage.IsExist(err) {
			fmt.Println("Try again: user already exist.")
			goto INPUT_USERNAME
		}
		return err
	}

	yes, err := input.Readln("> Start server right now? [y/N]: ")
	if err != nil {
		return err
	}
	if !strings.HasPrefix(strings.ToLower(yes), "y") {
		fmt.Println("Done.")
		os.Exit(0)
	}

	fmt.Println("You are good to go.")
	return nil
}

func tryStartDebugServer(debugAddr string, logger *zap.SugaredLogger) {
	if debugAddr == "" {
		return
	}
	debugServer := debug.NewServer(debugAddr)
	go func() {
		if err := debugServer.ListenAndServe(); err != nil {
			logger.Errorf("Start debug server failed: %v", err)
		}
	}()
}

func openInBrowser(endpoint string, logger *zap.SugaredLogger) {
	if *b {
		go func() {
			if err := open.Open(endpoint); err != nil {
				logger.Warnf("Open %s in browser failed: %v", endpoint, err)
			}
		}()
	}
}
