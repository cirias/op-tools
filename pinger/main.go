package main

import (
	"encoding/json"
	"flag"
	"fmt"
	redis "gopkg.in/redis.v3"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Log struct {
	Timestamp    time.Time
	ResponseTime int64
	Status       string
	Message      string
}

func main() {
	var name, url, interval, expiration, redisAddr string

	flag.StringVar(&name, "name", "", "name")
	flag.StringVar(&url, "url", "", "test url")
	flag.StringVar(&interval, "interval", "5m", "interval of ping")
	flag.StringVar(&expiration, "expiration", "24h", "expiration of the key in redis")
	flag.StringVar(&redisAddr, "redis-addr", "localhost:6379", "address of redis server")
	flag.Parse()

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

	tc := time.Tick(intervalDrt)

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	for t := range tc {
		log := ping(url)
		value, err := json.Marshal(log)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = client.Set(fmt.Sprintf("string:pinger:%s:%s", name, t), string(value), expirationDrt).Err()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

func ping(url string) (l Log) {
	l.Timestamp = time.Now()
	resp, err := http.Get(url)
	if err != nil {
		l.Status = "fail"
		l.Message = err.Error()
		return
	}
	defer resp.Body.Close()

	l.ResponseTime = time.Since(l.Timestamp).Nanoseconds()
	l.Status = "success"

	body, err := ioutil.ReadAll(resp.Body)
	l.Message = string(body)
	return
}
