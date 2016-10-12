package goworker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"strings"
	"time"
)

type poller struct {
	process
	isStrict bool
}

func newPoller(queues []string, isStrict bool) (*poller, error) {
	process, err := newProcess("poller", queues)
	if err != nil {
		return nil, err
	}
	return &poller{
		process:  *process,
		isStrict: isStrict,
	}, nil
}

func (p *poller) getJob(conn *RedisConn) (*Job, error) {
	for _, queue := range p.queues(p.isStrict) {
		logger.Debugf("Checking %s", queue)

		reply, err := conn.Do("LPOP", fmt.Sprintf("%squeue:%s", workerSettings.Namespace, queue))
		if err != nil {
			return nil, err
		}
		if reply != nil {
			logger.Debugf("Found job on %s", queue)

			job := &Job{Queue: queue}

			decoder := json.NewDecoder(bytes.NewReader(reply.([]byte)))
			if workerSettings.UseNumber {
				decoder.UseNumber()
			}

			if err := decoder.Decode(&job.Payload); err != nil {
				return nil, err
			}
			return job, nil
		} else {
			key := fmt.Sprintf("%szqueue:%s", workerSettings.Namespace, queue)
			if _, err := conn.Do("WATCH", key); nil != err {
				return nil, err
			}

			queues, err := redis.Strings(conn.Do("ZRANGEBYSCORE", key, 0, time.Now().Unix()))
			if nil != err {
				return nil, err
			}
			if 0 == len(queues) {
				continue
			}
			conn.Send("MULTI")
			conn.Send("ZREM", key, queues[0])
			if _, err = conn.Do("Exec"); nil != err {
				return nil, err
			}

			logger.Debugf("Found job on %s", queue)
			job := &Job{Queue: queue}
			decoder := json.NewDecoder(strings.NewReader(queues[0]))
			if workerSettings.UseNumber {
				decoder.UseNumber()
			}
			if err := decoder.Decode(&job.Payload); err != nil {
				return nil, err
			}
			return job, nil
		}
	}

	return nil, nil
}

func (p *poller) poll(interval time.Duration, quit <-chan bool) <-chan *Job {
	jobs := make(chan *Job)

	conn, err := GetConn()
	if err != nil {
		logger.Criticalf("Error on getting connection in poller %s", p)
		close(jobs)
		return jobs
	} else {
		p.open(conn)
		p.start(conn)
		PutConn(conn)
	}

	go func() {
		defer func() {
			close(jobs)

			conn, err := GetConn()
			if err != nil {
				logger.Criticalf("Error on getting connection in poller %s", p)
				return
			} else {
				p.finish(conn)
				p.close(conn)
				PutConn(conn)
			}
		}()

		for {
			select {
			case <-quit:
				return
			default:
				conn, err := GetConn()
				if err != nil {
					logger.Criticalf("Error on getting connection in poller %s", p)
					return
				}

				job, err := p.getJob(conn)
				if err != nil {
					logger.Criticalf("Error on %v getting job from %v: %v", p, p.Queues, err)
					PutConn(conn)
					return
				}
				if job != nil {
					conn.Send("INCR", fmt.Sprintf("%sstat:processed:%v", workerSettings.Namespace, p))
					conn.Flush()
					PutConn(conn)
					select {
					case jobs <- job:
					case <-quit:
						buf, err := json.Marshal(job.Payload)
						if err != nil {
							logger.Criticalf("Error requeueing %v: %v", job, err)
							return
						}
						conn, err := GetConn()
						if err != nil {
							logger.Criticalf("Error on getting connection in poller %s", p)
							return
						}
						conn.Send("LPUSH", fmt.Sprintf("%squeue:%s", workerSettings.Namespace, job.Queue), buf)
						conn.Flush()
						return
					}
				} else {
					PutConn(conn)
					if workerSettings.ExitOnComplete {
						return
					}
					logger.Debugf("Sleeping for %v", interval)
					logger.Debugf("Waiting for %v", p.Queues)

					timeout := time.After(interval)
					select {
					case <-quit:
						return
					case <-timeout:
					}
				}
			}
		}
	}()

	return jobs
}
