package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/daemon/logger"
	"github.com/sirupsen/logrus"
)

type elasticBulkWriter struct {
	esHost       string
	esIndex      string
	esType       string
	esDateSuffix string
	bulkSize     int
	buffer       []string
	logInfo      logger.Info
	mu           *sync.Mutex
	ticker       *time.Ticker
	tickerDone   chan bool
}

func newElasticBulkWriter(logInfo logger.Info) (*elasticBulkWriter, error) {
	var esHost string

	if h, ok := logInfo.Config["host"]; !ok {
		esHost = os.Getenv("HOST")
		if esHost == "" {
			logrus.WithField("id", logInfo.ContainerID).Error("HOST is not defined so --log-opt host is mandatory")
			return nil, fmt.Errorf("HOST is not defined so --log-opt host is mandatory")
		}
	} else {
		esHost = h
	}

	u, err := url.Parse(esHost)
	if err != nil {
		logrus.WithField("id", logInfo.ContainerID).Error(err.Error())
		return nil, fmt.Errorf("%w. Missing scheme?", err)
	}
	if u.Scheme == "" {
		logrus.WithField("id", logInfo.ContainerID).Error("Invalid host. Missing scheme")
		return nil, fmt.Errorf("Invalid host. Missing scheme")
	}
	esHost = fmt.Sprintf("%s://%s", u.Scheme, u.Host)

	esGCTimer := os.Getenv("GCTIMER")
	if esGCTimer == "" {
		esGCTimer = "1m"
	}

	if _, ok := logInfo.Config["index"]; !ok {
		logrus.WithField("id", logInfo.ContainerID).Error("--log-opt index is mandatory")
		return nil, fmt.Errorf("--log-opt index is mandatory")
	}
	esType := "log"
	if t, ok := logInfo.Config["type"]; ok {
		esType = t
	}
	bulkSize := 10
	if bs, ok := logInfo.Config["bulksize"]; ok {
		bulkSize, _ = strconv.Atoi(bs)
	} else if bs := os.Getenv("bulksize"); bs != "" {
		bulkSize, _ = strconv.Atoi(bs)
	}

	tickDuration, err := time.ParseDuration(esGCTimer)
	if err != nil {
		return nil, err
	}
	es := &elasticBulkWriter{
		esHost:       esHost,
		esIndex:      logInfo.Config["index"],
		esType:       esType,
		bulkSize:     bulkSize,
		esDateSuffix: os.Getenv("DATESUFFIX"),
		logInfo:      logInfo,
		mu:           &sync.Mutex{},
		ticker:       time.NewTicker(tickDuration),
		tickerDone:   make(chan bool),
	}
	go es.gc()
	return es, nil
}

func (es *elasticBulkWriter) write(line string) {

	es.mu.Lock()

	if len(es.buffer) < es.bulkSize {
		es.buffer = append(es.buffer, line)
		es.mu.Unlock()
		return
	}

	fullBuffer := es.buffer
	es.buffer = nil
	es.buffer = append(es.buffer, line)
	es.mu.Unlock()

	logrus.WithField("id", es.logInfo.ContainerID).
		WithField("container", es.logInfo.ContainerName).
		WithField("elasticHost", es.esHost).
		WithField("bulksize", es.bulkSize).
		Info("Sending bulk to elastic")

	go es.send(fullBuffer)
}

func (es *elasticBulkWriter) send(buffer []string) {

	payload := strings.Builder{}

	indexDate := time.Now().Format(es.esDateSuffix)
	indexName := fmt.Sprintf("%s_%s", es.esIndex, indexDate)

	for _, l := range buffer {
		//payload.WriteString(fmt.Sprintf(`{"index": {"_index":"%s", "_type":"%s"} }`, es.esIndex, es.esType))
		payload.WriteString(fmt.Sprintf(`{"index": {"_index":"%s", "_type":"%s"} }`, indexName, es.esType))
		payload.WriteString("\n")
		payload.WriteString(l)
		payload.WriteString("\n")
	}
	body := payload.String()

	url := fmt.Sprintf("%s/_bulk", es.esHost)

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		logrus.WithField("id", es.logInfo.ContainerID).
			WithField("container", es.logInfo.ContainerName).
			WithField("elastichost", es.esHost).
			Error(err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/x-ndjson")
	if u := es.config("USER"); u != "" {
		if p := es.config("PASSWORD"); p != "" {
			req.SetBasicAuth(u, p)
		}
	}

	r, err := http.DefaultClient.Do(req)
	//r, err := http.Post(url, "application/x-ndjson", strings.NewReader(body))
	if err != nil {
		logrus.WithField("id", es.logInfo.ContainerID).
			WithField("container", es.logInfo.ContainerName).
			WithField("elasticHost", es.esHost).
			Error(err.Error())
		return
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		rBody, _ := ioutil.ReadAll(r.Body)

		logrus.WithField("id", es.logInfo.ContainerID).
			WithField("container", es.logInfo.ContainerName).
			WithField("elastichost", es.esHost).
			WithField("StatusCode", r.StatusCode).
			Error(string(rBody))
	}
}

func (es *elasticBulkWriter) gc() {

	logrus.WithField("id", es.logInfo.ContainerID).
		Debug("GC: goroutine possible leak")

	for {

		select {
		case _ = <-es.ticker.C:
			es.mu.Lock()
			if len(es.buffer) > 0 {
				fullBuffer := es.buffer
				es.buffer = nil

				logrus.WithField("id", es.logInfo.ContainerID).
					WithField("container", es.logInfo.ContainerName).
					WithField("elasticHost", es.esHost).
					WithField("bulksize", len(fullBuffer)).
					Info("GC: Sending bulk to elastic")

				go es.send(fullBuffer)
			}
			es.mu.Unlock()
		case _ = <-es.tickerDone:
			logrus.WithField("id", es.logInfo.ContainerID).
				Debug("GC: goroutine not a leak")
			return
		}
	}
}

func (es *elasticBulkWriter) Stop() {
	es.ticker.Stop()
	es.tickerDone <- true
}

func (es *elasticBulkWriter) config(key string) string {
	u := os.Getenv(key)
	if v, ok := es.logInfo.Config[key]; ok {
		u = v
	}
	return u
}
