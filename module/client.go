package module

import (
	"encoding/json"
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

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	req := &Request{
		Cmd: SendAddr,
	}
	data, err := json.Marshal(c.config.Name)
	if err != nil {
		logs.Error("marshal error when send addr: %s", err)
		return
	}
	req.Data = data

	err = encoder.Encode(req)
	if err != nil {
		logs.Error("encode error when send addr: %s", err)
		return
	}

	resp := &Response{}
	err = decoder.Decode(resp)
	if err != nil {
		logs.Error("decode error when send addr: %s", err)
		return
	}
	if resp.Stat != Success {
		logs.Warn("send addr failed")
	} else {
		logs.Info("send addr success")
	}

}
