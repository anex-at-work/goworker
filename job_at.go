package goworker

import (
	"time"
)

type JobAt struct {
	Queue   string
	Payload Payload
	RunAt   time.Time
}
