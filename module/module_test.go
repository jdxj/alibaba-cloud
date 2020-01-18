package module

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
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
	record, err := server.getRecord("mysql")
	if err != nil {
		t.Fatalf("%s", err)
	}
	if record != nil {
		fmt.Printf("%s\n", record.Value)
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

func TestPrint(t *testing.T) {
	str := "danke"
	data, _ := json.Marshal(str)
	fmt.Printf("len: %d, data: %s\n", len(data), data)

	err := json.Unmarshal(data, &str)
	if err != nil {
		t.Fatalf("%s\n", str)
	}
	fmt.Printf("%s\n", str)
}

func TestAddRecord(t *testing.T) {
	config := readConfig()

	server, err := NewServer(config.Server, config.Access, config.MySQL)
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	//err = server.addRecord("danke", "111.0.82.121")
	//if err != nil {
	//	t.Fatalf("%s\n", err)
	//}

	server.addRecordSilently("danke", "111.0.82.121")
}

func TestDelRecord(t *testing.T) {
	config := readConfig()

	server, err := NewServer(config.Server, config.Access, config.MySQL)
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	server.delRecordSilently("danke")
}

func TestModRecord(t *testing.T) {
	config := readConfig()

	server, err := NewServer(config.Server, config.Access, config.MySQL)
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	resp, err := server.addRecord("danke", "111.0.82.121")
	if err != nil {
		t.Fatalf("%s\n", err)
	}

	time.Sleep(time.Minute)
	err = server.modRecord(resp.RecordId, "danke2", "111.0.82.121")
	if err != nil {
		t.Fatalf("%s\n", err)
	}
}
