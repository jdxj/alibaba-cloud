package module

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"

	"github.com/astaxie/beego/logs"
)

const (
	domainNameSuffix  = ".aaronkir.xyz"
	DefaultListenAddr = ":49164"
)

func NewServer(sc *ServerConfig, access *Access, mc *MySQLConfig) (*Server, error) {
	if sc == nil {
		return nil, fmt.Errorf("invalid server config")
	}
	if access == nil {
		return nil, fmt.Errorf("invalid access")
	}

	mysql, err := NewMySQL(mc)
	if err != nil {
		return nil, err
	}

	s := &Server{
		stop:   make(chan struct{}),
		wg:     &sync.WaitGroup{},
		mutex:  &sync.Mutex{},
		config: sc,
		access: access,
		mysql:  mysql,
	}
	return s, nil
}

type Server struct {
	stop chan struct{}
	wg   *sync.WaitGroup

	mutex      *sync.Mutex
	remoteAddr net.Addr

	config *ServerConfig
	access *Access

	mysql *MySQL
}

func (s *Server) Start() {
	logs.Info("start server")

	wg := s.wg
	wg.Add(1)
	go func() {
		s.listen()
		wg.Done()
	}()
}

func (s *Server) Stop() {
	close(s.stop)
	s.wg.Wait()
	s.mysql.Close()
	logs.Info("server stopped")
}

// listen 必须在一个 goroutine 中运行.
func (s *Server) listen() {
	addr := s.config.ListenAddr
	if addr == "" {
		addr = DefaultListenAddr
		logs.Warn("use default listen addr: %s", DefaultListenAddr)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logs.Error("can not create listener: %s", err)
		return
	}

	go func() {
		select {
		case <-s.stop:
			// 只在这里停止 listen?
			listener.Close()
			logs.Info("stop listen")
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.stop:
				// 不确定在收到这个错误 (use of closed network connection) 后,
				// conn 是否有已接受的连接.
				if conn != nil {
					conn.Close()
				}
				logs.Info("listen stopped")
				return
			default:
				logs.Error("%s", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

// handleConn 需要在一个 goroutine 中运行.
func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)

	req := &Request{}
	err := decoder.Decode(req)
	if err != nil {
		logs.Error("error when decode request")
		return
	}

	switch req.Cmd {
	case SendAddr:
		s.handleSendAddr(conn)
	default:
		conn.Close()
		logs.Warn("Cmd error: not define")
	}
}

func (s *Server) handleSendAddr(conn net.Conn) {
	defer conn.Close()

	s.mutex.Lock()
	s.remoteAddr = conn.RemoteAddr()
	logs.Info("remote address: %s", s.remoteAddr)
	s.mutex.Unlock()

	encoder := json.NewEncoder(conn)
	resp := &Response{
		Stat: Success,
	}
	if err := encoder.Encode(resp); err != nil {
		logs.Error("error when encode response")
	}
}

func (s *Server) hasRecord(recordName string) (bool, error) {
	access := s.access
	client, err := alidns.NewClientWithAccessKey(access.Region, access.AccessKeyID, access.AccessSecret)
	if err != nil {
		return false, err
	}

	req := alidns.CreateDescribeDomainRecordsRequest()
	req.Scheme = "https"
	req.DomainName = s.config.DomainName

	var resp *alidns.DescribeDomainRecordsResponse
	// 重试
	for i := 0; i < 3; i++ {
		resp, err = client.DescribeDomainRecords(req)
		if err != nil {
			logs.Error("get domain records failed, retry count: %d, error: %s", i, err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	if err != nil {
		return false, err
	}

	for _, record := range resp.DomainRecords.Record {
		if record.RR == recordName {
			return true, nil
		}
	}
	return false, nil
}
