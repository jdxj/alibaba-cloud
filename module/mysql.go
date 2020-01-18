package module

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/astaxie/beego/logs"

	_ "github.com/go-sql-driver/mysql"
)

const (
	IPTable = "ip"
)

func NewMySQL(config *MySQLConfig) (*MySQL, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid mysql config")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?loc=Local&parseTime=true",
		config.User, config.Password, config.Address, config.Database)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	mysql := &MySQL{
		config: config,
		db:     db,
		stop:   make(chan struct{}),
	}
	go func() {
		mysql.ping()
	}()
	return mysql, nil
}

type MySQL struct {
	config *MySQLConfig
	db     *sql.DB

	stop chan struct{}
}

func (mysql *MySQL) Close() error {
	close(mysql.stop)
	return mysql.db.Close()
}

// ping 用于避免连接失效 (心跳作用)
func (mysql *MySQL) ping() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-mysql.stop:
			logs.Info("stop mysql ping")
			return

		case <-ticker.C:
			if err := mysql.db.Ping(); err != nil {
				logs.Error("ping error when mysql ping: %s:", err)
			}
		}
	}
}

func (mysql *MySQL) InsertIP(name, address string) error {
	query := fmt.Sprintf(`INSERT INTO %s (name,address) VALUES (?,?)`, IPTable)
	_, err := mysql.db.Exec(query, name, address)
	return err
}
