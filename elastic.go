package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/daemon/logger"
	"github.com/sirupsen/logrus"
)

type elasticBulkWriter struct {
	esHost   string
	esIndex  string
	esType   string
	bulkSize int
	buffer   []string
	logInfo  logger.Info
	mu       *sync.Mutex
	ticker   *time.Ticker
}

func newElasticBulkWriter(logInfo logger.Info) (*elasticBulkWriter, error) {

	esHost := os.Getenv("HOST")
	if esHost == "" {
		if _, ok := logInfo.Config["host"]; !ok {
			logrus.WithField("id", logInfo.ContainerID).Error("ELASTICLOGGER_HOST is not defined so --log-opt host is mandatory")
			return nil, fmt.Errorf("ELASTICLOGGER_HOST is not defined so --log-opt host is mandatory")
		} else {
			esHost = logInfo.Config["host"]
		}
	}
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
	}

	tickDuration, err := time.ParseDuration(esGCTimer)
	if err != nil {
		return nil, err
	}
	es := &elasticBulkWriter{
		esHost:   esHost,
		esIndex:  logInfo.Config["index"],
		esType:   esType,
		bulkSize: bulkSize,
		logInfo:  logInfo,
		mu:       &sync.Mutex{},
		ticker:   time.NewTicker(tickDuration),
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

	for _, l := range buffer {
		payload.WriteString(fmt.Sprintf(`{"index": {"_index":"%s", "_type":"%s"} }`, es.esIndex, es.esType))
		payload.WriteString("\n")
		payload.WriteString(l)
		payload.WriteString("\n")
	}
	body := payload.String()

	url := fmt.Sprintf("http://%s/_bulk", es.esHost)

	r, err := http.Post(url, "application/x-ndjson", strings.NewReader(body))
	if err != nil {
		logrus.WithField("id", es.logInfo.ContainerID).
			WithField("container", es.logInfo.ContainerName).
			WithField("elasticHost", es.esHost).
			Error(err.Error())
		return
	}
	defer r.Body.Close()
	rBody, _ := ioutil.ReadAll(r.Body)

	if r.StatusCode != 200 {
		logrus.WithField("id", es.logInfo.ContainerID).
			WithField("container", es.logInfo.ContainerName).
			WithField("elastichost", es.esHost).
			WithField("StatusCode", r.StatusCode).
			Error(string(rBody))
	}
}

func (es *elasticBulkWriter) gc() {

	for _ = range es.ticker.C {
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
	}
}
