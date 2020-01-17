package module

import (
	"encoding/json"
	"fmt"
	"testing"
)

func readConfig() *Config {
	config, err := ReadConfig()
	if err != nil {
		panic(err)
	}
	return config
}

func TestNewMySQL(t *testing.T) {
	config := readConfig()
	mysql, err := NewMySQL(config.MySQL)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	defer mysql.Close()

	err = mysql.InsertIP("danke", "mock_ip")
	if err != nil {
		t.Fatalf("%s", err)
	}
}

func TestNewServer(t *testing.T) {
	config := readConfig()
	server, err := NewServer(config.Server, config.Access, config.MySQL)
	if err != nil {
		t.Fatalf("%s", err)
	}
	exist, err := server.hasRecord("mysql")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if !exist {
		t.Fatalf("%s", "record not exist")
	}
	exist, err = server.hasRecord("danke")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if exist {
		t.Fatalf("%s", "has resolve domain name by manual?")
	}
}

type A struct {
	Name string `json:"name"`
}

type B struct {
	Age int `json:"age"`
}

func TestJsonMarshal(t *testing.T) {
	data, _ := json.Marshal(A{Name: "mockName"})
	req := Request{
		Cmd:  SendAddr,
		Data: data,
	}
	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("%s", err)
	}
	fmt.Printf("%s---------\n", data)

	reqP := &Request{}
	err = json.Unmarshal(data, reqP)
	if err != nil {
		t.Fatalf("%s", err)
	}
	fmt.Printf("%+v\n", reqP)

	a := &A{}
	json.Unmarshal(reqP.Data, a)
	fmt.Printf("%+v\n", *a)
}
