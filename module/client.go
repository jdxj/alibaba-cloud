package module

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/astaxie/beego/logs"
)

const (
	DefaultDialAddr   = "127.0.0.1:49164"
	SendIntervalLimit = time.Minute
)

func NewClient(config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid client config")
	}

	c := &Client{
		stop:   make(chan struct{}),
		wg:     &sync.WaitGroup{},
		config: config,
	}
	return c, nil
}

type Client struct {
	stop chan struct{}
	wg   *sync.WaitGroup

	config *ClientConfig
}

func (c *Client) Start() {
	logs.Info("start client")

	wg := c.wg
	wg.Add(1)
	go func() {
		c.SendAddrRegularly()
		wg.Done()
	}()
}

func (c *Client) Stop() {
	close(c.stop)
	c.wg.Wait()

	logs.Info("client stopped")
}

func (c *Client) SendAddrRegularly() {
	interval, err := time.ParseDuration(c.config.Interval)
	if err != nil {
		logs.Error("parse time duration error: %s", err)
		interval = SendIntervalLimit
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			logs.Info("stop send address")
			return
		case <-ticker.C:
			c.sendAddr()
		}
	}
}

// sendAddr 只是向 server 连接一次,
// 以告知其 address.
func (c *Client) sendAddr() {
	addr := c.config.DialAddr
	if addr == "" {
		addr = DefaultDialAddr
		logs.Warn("use default dial addr: %s", DefaultDialAddr)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		logs.Error("%s", err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		logs.Error("data: %s, error: %s", buf[:n], err)
		return
	}
	resp := string(buf[:n])
	if resp != "ok" {
		logs.Error("receive a invalid message")
		return
	}

	logs.Info("send address success")
}
