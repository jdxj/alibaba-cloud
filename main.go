package main

import (
	"ali/module"
	"os"
	"os/signal"
	"syscall"

	"github.com/astaxie/beego/logs"
)

const (
	ServerMode = iota
	ClientMode
)

func main() {
	config, err := module.ReadConfig()
	if err != nil {
		panic(err)
	}

	var adapter string
	if config.Debug {
		adapter = logs.AdapterConsole
	} else {
		adapter = logs.AdapterFile
	}
	err = logs.SetLogger(adapter, `{"filename":"aliyun.log","level":7,"maxlines":0,"maxsize":0,"daily":true,"maxdays":3,"color":true}`)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal, 2)
	defer func() {
		close(sigs)
	}()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	switch config.Mode {
	case ServerMode:
		server, err := module.NewServer(config.Server, config.Access, config.MySQL)
		if err != nil {
			logs.Error("new server failed: %s", err)
			return
		}
		server.Start()

		select {
		case <-sigs:
			logs.Info("receive signal")
		}

		server.Stop()

	case ClientMode:
		client, err := module.NewClient(config.Client)
		if err != nil {
			logs.Error("new client failed: %s", err)
			return
		}
		client.Start()

		select {
		case <-sigs:
			logs.Info("receive signal")
		}

		client.Stop()

	default:
		logs.Warn("no define mode")
	}
}
