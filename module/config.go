package module

import (
	"encoding/json"
	"os"
)

type Config struct {
	Debug  bool          `json:"debug"`
	Mode   int           `json:"mode"`
	MySQL  *MySQLConfig  `json:"mysql"`
	Access *Access       `json:"access"`
	Server *ServerConfig `json:"server"`
	Client *ClientConfig `json:"client"`
}

type MySQLConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Address  string `json:"address"`
	Database string `json:"database"`
}

type Access struct {
	Region       string `json:"region"`
	AccessKeyID  string `json:"access_key_id"`
	AccessSecret string `json:"access_secret"`
}

type ServerConfig struct {
	ListenAddr string `json:"listen_addr"`
	DomainName string `json:"domain_name"`
}

type ClientConfig struct {
	Name     string `json:"name"`
	DialAddr string `json:"dial_addr"`
	Interval string `json:"interval"`
}

func ReadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decode := json.NewDecoder(file)
	config := &Config{}
	return config, decode.Decode(config)
}
