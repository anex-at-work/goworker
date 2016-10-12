package goworker

type PayloadAt struct {
	Class string        `json:"class"`
	Args  []interface{} `json:"args"`
	RunAt float64       `json:"run_at"`
}
