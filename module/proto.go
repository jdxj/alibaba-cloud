package module

import "encoding/json"

type Cmd string
type Stat string

const (
	SendAddr Cmd = "send_addr"
)

const (
	Success Stat = "success"
)

type Request struct {
	Cmd  Cmd             `json:"cmd"`
	Data json.RawMessage `json:"data"`
}

type Response struct {
	Stat Stat            `json:"stat"`
	Data json.RawMessage `json:"data"`
}
