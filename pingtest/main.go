package main

import (
	"encoding/json"
	"flag"
	"fmt"
	redis "gopkg.in/redis.v3"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	OK = iota
	FAIL
)

type Ping struct {
	ResponseTime int64
	Status       int
}

type Log struct {
	Timestamp       int64
	AvgResponseTime float64
	Avaliable       float64
}

func main() {
	var name, url, duration, interval, expiration, redisAddr string

	flag.StringVar(&name, "name", "", "name")
	flag.StringVar(&url, "url", "", "test url")
	flag.StringVar(&duration, "duration", "1m", "duration of ping test average result")
	flag.StringVar(&interval, "interval", "10s", "interval of ping test")
	flag.StringVar(&expiration, "expiration", "2h", "expiration of the key in redis")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "address of redis server")
	flag.Parse()

	durationDrt, err := time.ParseDuration(duration)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	intervalDrt, err := time.ParseDuration(interval)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	expirationDrt, err := time.ParseDuration(expiration)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	itc := time.Tick(intervalDrt)

	dtc := time.Tick(durationDrt)

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	pings := make([]*Ping, 0, 4)
	for {
		select {
		case <-itc:
			ping, err := pingtest(url)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			pings = append(pings, &ping)
		case dt := <-dtc:
			var sumResponseTime int64 = 0
			sumAvaliable := len(pings)
			for _, ping := range pings {
				if ping.Status == 0 {
					sumResponseTime += ping.ResponseTime
				}
				sumAvaliable -= ping.Status
			}

			log := &Log{
				Timestamp:       dt.Unix() * 1000,
				AvgResponseTime: float64(sumResponseTime) / float64(sumAvaliable) / float64(1000000),
				Avaliable:       float64(sumAvaliable) / float64(len(pings)),
			}

			pings = make([]*Ping, 0, 4)

			value, err := json.Marshal(log)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			err = client.Set(fmt.Sprintf("pingtest:log:%s:%s:%s", name, duration, strconv.FormatInt(log.Timestamp, 10)), string(value), expirationDrt).Err()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}

func pingtest(url string) (p Ping, err error) {
	timestamp := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		p.Status = FAIL
		return p, nil
	}
	defer resp.Body.Close()

	p.ResponseTime = time.Since(timestamp).Nanoseconds()
	p.Status = OK

	_, err = ioutil.ReadAll(resp.Body)
	return p, err
}
