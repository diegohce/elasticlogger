package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/daemon/logger"
)

func TestElasticBulkWriter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)
	tsURL, _ := url.Parse(ts.URL)

	logInfo.Config["host"] = tsURL.Host
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"

	es, _ := newElasticBulkWriter(logInfo)
	defer es.Stop()

	es.write("Line 1")
	es.write("Line 2")
	es.write("Line 3")
	es.write("Line 4")

	time.Sleep(1 * time.Second)
}

func TestElasticBulkWriter400(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		fmt.Fprintf(w, "BAD REQUEST testing")
	}))
	defer ts.Close()

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)
	tsURL, _ := url.Parse(ts.URL)

	logInfo.Config["host"] = tsURL.Host
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"

	es, _ := newElasticBulkWriter(logInfo)
	defer es.Stop()

	es.write("Line 1")
	es.write("Line 2")
	es.write("Line 3")
	es.write("Line 4")

	time.Sleep(1 * time.Second)
}

func TestElasticBulkWriterGC(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	os.Setenv("GCTIMER", "1s")

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)
	tsURL, _ := url.Parse(ts.URL)

	logInfo.Config["host"] = tsURL.Host
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"

	es, _ := newElasticBulkWriter(logInfo)
	defer es.Stop()

	es.write("Line 1")

	time.Sleep(2 * time.Second)
}

func TestElasticBulkWriterNoHost(t *testing.T) {

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)

	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"

	es, err := newElasticBulkWriter(logInfo)
	if err == nil {
		es.Stop()
		t.Error(err)
	}

}

func TestElasticBulkWriterNoIndex(t *testing.T) {

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)

	logInfo.Config["host"] = "somehost:9999"
	logInfo.Config["bulksize"] = "3"

	es, err := newElasticBulkWriter(logInfo)
	if err == nil {
		es.Stop()
		t.Error("index is mandatory. Check failed.")
	}
}

func TestElasticBulkWriterWithCustomType(t *testing.T) {

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)

	logInfo.Config["host"] = "somehost:9999"
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"
	logInfo.Config["type"] = "test_type"

	es, err := newElasticBulkWriter(logInfo)
	if err != nil {
		t.Error(err)
	}
	es.Stop()

}

func TestElasticBulkWriterBadGCDuration(t *testing.T) {

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)

	os.Setenv("GCTIMER", "bad duration string")

	logInfo.Config["host"] = "somehost:9999"
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"
	logInfo.Config["type"] = "test_type"

	es, err := newElasticBulkWriter(logInfo)
	if err == nil {
		es.Stop()
		t.Error("Invalid duration check failed")
	}
}

func TestElasticBulkWriterBadHost(t *testing.T) {

	var logInfo logger.Info
	logInfo.Config = make(map[string]string)

	os.Setenv("GCTIMER", "15m")

	logInfo.Config["host"] = "localhost:9999"
	logInfo.Config["bulksize"] = "3"
	logInfo.Config["index"] = "testindex"

	es, err := newElasticBulkWriter(logInfo)
	if err != nil {
		t.Error(err)
	}
	defer es.Stop()

	es.write("Line 1")
	es.write("Line 2")
	es.write("Line 3")
	es.write("Line 4")

	time.Sleep(5 * time.Second)
}
