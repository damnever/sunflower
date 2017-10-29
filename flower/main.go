package flower

import (
	"flag"

	"github.com/damnever/cc"
	"github.com/damnever/sunflower/log"
	"github.com/damnever/sunflower/pkg/debug"
)

var (
	c = flag.String("c", "", "Path to client configuration file, useful for self build client.")
)

func Run() {
	flag.Parse()
	logger := log.New("M")

	var (
		cconf cc.Configer
		err   error
	)
	if *c != "" {
		cconf, err = cc.NewConfigFromFile(*c)
	} else {
		var data []byte
		if data, err = loadConfigFromExec(); err == nil {
			cconf, err = cc.NewConfigFromYAML(data)
		}
	}
	if err != nil {
		logger.Fatalf("Load config failed: %v", err)
	}
	debugAddr := cconf.String("debug_addr")
	conf := buildConfig(cconf)
	cconf = nil

	if debugAddr != "" {
		debugServer := debug.NewServer(debugAddr)
		go func() {
			if err := debugServer.ListenAndServe(); err != nil {
				logger.Errorf("Start debug server failed: %v", err)
			}
		}()
		defer debugServer.Close()
	}

	logger.Infof("The flower is blooming..")
	ctl, err := NewControler(conf)
	if err != nil {
		logger.Fatalf("Init failed: %v", err)
	}
	if err = ctl.Run(); err != nil {
		logger.Fatalf("Stopped with: %v", err)
	}
}
