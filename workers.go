package goworker

import (
	"encoding/json"
	"fmt"
)

var (
	workers map[string]workerFunc
)

func init() {
	workers = make(map[string]workerFunc)
}

// Register registers a goworker worker function. Class
// refers to the Ruby name of the class which enqueues the
// job. Worker is a function which accepts a queue and an
// arbitrary array of interfaces as arguments.
func Register(class string, worker workerFunc) {
	workers[class] = worker
}

func Enqueue(job *Job) error {
	err := Init()
	if err != nil {
		return err
	}

	conn, err := GetConn()
	if err != nil {
		logger.Criticalf("Error on getting connection on enqueue")
		return err
	}
	defer PutConn(conn)

	buffer, err := json.Marshal(job.Payload)
	if err != nil {
		logger.Criticalf("Cant marshal payload on enqueue")
		return err
	}

	err = conn.Send("RPUSH", fmt.Sprintf("%squeue:%s", workerSettings.Namespace, job.Queue), buffer)
	if err != nil {
		logger.Criticalf("Cant push to queue")
		return err
	}

	err = conn.Send("SADD", fmt.Sprintf("%squeues", workerSettings.Namespace), job.Queue)
	if err != nil {
		logger.Criticalf("Cant register queue to list of use queues")
		return err
	}

	return conn.Flush()
}

func EnqueueAt(job *JobAt) error {
	err := Init()
	if err != nil {
		return err
	}

	conn, err := GetConn()
	if err != nil {
		logger.Criticalf("Error on getting connection on enqueue")
		return err
	}
	defer PutConn(conn)
	runAt := float64(job.RunAt.UnixNano()) / 1e+9
	buffer, err := json.Marshal(PayloadAt{
		Class: job.Payload.Class,
		Args:  job.Payload.Args,
		RunAt: runAt,
	})
	if err != nil {
		logger.Criticalf("Cant marshal payload on enqueue")
		return err
	}
	err = conn.Send("ZADD", fmt.Sprintf("%szqueue:%s", workerSettings.Namespace, job.Queue), runAt, buffer)
	if err != nil {
		logger.Criticalf("Cant add to sorted set")
		return err
	}

	return conn.Flush()
}
