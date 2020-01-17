package module

import "encoding/json"

type Cmd int
type Stat int

const (
	SendAddr Cmd = iota
)

const (
	Success Stat = iota
)

type Request struct {
	Cmd  Cmd             `json:"cmd"`
	Data json.RawMessage `json:"data"`
}

type Response struct {
	Stat Stat            `json:"stat"`
	Data json.RawMessage `json:"data"`
}
