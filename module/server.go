package module

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"

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
		stop:        make(chan struct{}),
		wg:          &sync.WaitGroup{},
		mutex:       &sync.Mutex{},
		remoteAddrs: make(map[string]net.Addr),
		config:      sc,
		access:      access,
		mysql:       mysql,
	}
	return s, nil
}

type Server struct {
	stop chan struct{}
	wg   *sync.WaitGroup

	mutex       *sync.Mutex
	remoteAddrs map[string]net.Addr

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
		logs.Error("decode error when handle conn")
		return
	}

	switch req.Cmd {
	case SendAddr:
		s.handleSendAddr(conn, req)
	default:
		logs.Warn("Cmd error: not define")
	}
}

func (s *Server) handleSendAddr(conn net.Conn, req *Request) {
	defer conn.Close()

	var name string
	err := json.Unmarshal(req.Data, &name)
	if err != nil {
		logs.Error("unmarshal error when handle send addr")
		return
	}

	remoteAddr := conn.RemoteAddr()
	remoteIP := getIP(remoteAddr)

	// 避免在锁中有耗时操作
	var needAdd bool
	s.mutex.Lock()
	if addr, ok := s.remoteAddrs[name]; !ok {
		s.remoteAddrs[name] = remoteAddr
		needAdd = true
	} else {
		if getIP(addr) != remoteIP {
			s.remoteAddrs[name] = remoteAddr
			needAdd = true
		}
	}
	logs.Info("remote name: %s, address: %s", name, conn.RemoteAddr())
	s.mutex.Unlock()

	if needAdd {
		if err := s.mysql.InsertIP(name, remoteIP); err != nil {
			logs.Error("insert ip error when handle send addr: %s", err)
		}

		s.addRecordSilently(name, remoteIP)
	}

	encoder := json.NewEncoder(conn)
	resp := &Response{
		Stat: Success,
	}
	if err := encoder.Encode(resp); err != nil {
		logs.Error("error when encode response")
	}
}

func (s *Server) getRecord(recordName string) (*alidns.Record, error) {
	access := s.access
	client, err := alidns.NewClientWithAccessKey(access.Region, access.AccessKeyID, access.AccessSecret)
	if err != nil {
		return nil, err
	}

	req := alidns.CreateDescribeDomainRecordsRequest()
	req.Scheme = "https"
	req.DomainName = s.config.DomainName

	resp, err := client.DescribeDomainRecords(req)
	if err != nil {
		return nil, err
	}

	for i, record := range resp.DomainRecords.Record {
		if record.RR == recordName {
			return &resp.DomainRecords.Record[i], nil
		}
	}
	return nil, nil
}

func getIP(addr net.Addr) string {
	addrStr := addr.String()
	idx := strings.Index(addrStr, ":")
	if idx < 0 {
		return ""
	}
	return addrStr[:idx]
}

func (s *Server) addRecord(recordName, value string) (*alidns.AddDomainRecordResponse, error) {
	access := s.access
	client, err := alidns.NewClientWithAccessKey(access.Region, access.AccessKeyID, access.AccessSecret)
	if err != nil {
		return nil, err
	}

	req := alidns.CreateAddDomainRecordRequest()
	req.Scheme = "https"
	req.DomainName = s.config.DomainName
	req.Type = "A"
	req.RR = recordName
	req.Value = value

	return client.AddDomainRecord(req)
}

func (s *Server) delRecord(recordID string) error {
	access := s.access
	client, err := alidns.NewClientWithAccessKey(access.Region, access.AccessKeyID, access.AccessSecret)
	if err != nil {
		return err
	}

	req := alidns.CreateDeleteDomainRecordRequest()
	req.Scheme = "https"
	req.RecordId = recordID

	resp, err := client.DeleteDomainRecord(req)
	if err != nil {
		return err
	}
	logs.Debug("del record resp: %+v", resp)
	return nil
}

// addRecordSilently 如果不存在则添加, 否则修改.
func (s *Server) addRecordSilently(recordName, value string) {
	record, err := s.getRecord(recordName)
	if err != nil {
		logs.Error("get record error when insert records silently: %s", err)
		return
	}

	if record != nil {
		if err := s.modRecord(record.RecordId, recordName, value); err != nil {
			logs.Error("mod record error when add record silently: %s", err)
		}
		return
	}

	_, err = s.addRecord(recordName, value)
	if err != nil {
		logs.Error("add record error when insert records silently: %s", err)
		return
	}
}

func (s *Server) modRecord(recordID, recordName, value string) error {
	access := s.access
	client, err := alidns.NewClientWithAccessKey(access.Region, access.AccessKeyID, access.AccessSecret)
	if err != nil {
		return err
	}

	req := alidns.CreateUpdateDomainRecordRequest()
	req.Scheme = "https"

	req.RecordId = recordID
	req.Type = "A"
	req.RR = recordName
	req.Value = value

	resp, err := client.UpdateDomainRecord(req)
	if err != nil {
		return err
	}
	logs.Info("mod record resp: %+v", resp)
	return nil
}

func (s *Server) delRecordSilently(recordName string) {
	record, err := s.getRecord(recordName)
	if err != nil {
		logs.Error("get record error when del record silently: %s", err)
		return
	}

	if record == nil {
		logs.Warn("record not exist: %s", recordName)
		return
	}

	err = s.delRecord(record.RecordId)
	if err != nil {
		logs.Error("del record error when del record silently: %s", err)
		return
	}
}
