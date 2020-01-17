package module

import (
	"database/sql"
	"fmt"

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
	}
	return mysql, nil
}

type MySQL struct {
	config *MySQLConfig
	db     *sql.DB
}

func (mysql *MySQL) Close() error {
	return mysql.db.Close()
}

func (mysql *MySQL) InsertIP(name, address string) error {
	query := fmt.Sprintf(`INSERT INTO %s (name,address) VALUES (?,?)`, IPTable)
	_, err := mysql.db.Exec(query, name, address)
	return err
}
