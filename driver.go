package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"syscall"

	"github.com/containerd/fifo"
	"github.com/docker/docker/api/types/plugins/logdriver"
	"github.com/docker/docker/daemon/logger"
	protoio "github.com/gogo/protobuf/io"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type driver struct {
	mu   sync.Mutex
	logs map[string]*logPair
	//	idx    map[string]*logPair
	//	logger logger.Logger
}

type logPair struct {
	es     *elasticBulkWriter
	stream io.ReadCloser
	info   logger.Info
}

func newDriver() *driver {
	return &driver{
		logs: make(map[string]*logPair),
		//		idx:  make(map[string]*logPair),
	}
}

func (d *driver) StartLogging(file string, logCtx logger.Info) error {
	d.mu.Lock()
	if _, exists := d.logs[file]; exists {
		d.mu.Unlock()
		return fmt.Errorf("logger for %q already exists", file)
	}
	d.mu.Unlock()

	es, err := newElasticBulkWriter(logCtx)
	if err != nil {
		return err
	}

	logrus.WithField("id", logCtx.ContainerID).
		WithField("container", logCtx.ContainerName).
		WithField("elasticHost", es.esHost).
		WithField("index", es.esIndex).WithField("suffix", es.esDateSuffix).
		WithField("type", es.esType).Info("Start logging")

	// DHC - f is the log stream from docker to our plugin
	f, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
	if err != nil {
		return errors.Wrapf(err, "error opening logger fifo: %q", file)
	}

	d.mu.Lock()
	lf := &logPair{es, f, logCtx}
	d.logs[file] = lf
	//	d.idx[logCtx.ContainerID] = lf
	d.mu.Unlock()

	go consumeLog(lf)
	return nil
}

func (d *driver) StopLogging(file string) error {
	logrus.WithField("file", file).Info("Stop logging")
	d.mu.Lock()
	lf, ok := d.logs[file]
	if ok {
		lf.stream.Close()
		lf.es.Stop()
		delete(d.logs, file)
	}
	d.mu.Unlock()
	return nil
}

func consumeLog(lf *logPair) {
	dec := protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
	defer dec.Close()
	var buf logdriver.LogEntry
	for {
		if err := dec.ReadMsg(&buf); err != nil {
			if err == io.EOF {
				logrus.WithField("id", lf.info.ContainerID).WithError(err).Info("shutting down log logger")
				//lf.stream.Close()
				//return
			} else {
				//dec = protoio.NewUint32DelimitedReader(lf.stream, binary.BigEndian, 1e6)
				logrus.WithField("id", lf.info.ContainerID).WithError(err).Error("NOT EOF ERROR")
			}
			lf.stream.Close()
			return
		}

		//fmt.Println("buf.Line", string(buf.Line))

		if len(buf.Line) > 0 && buf.Line[0] == '{' {
			lf.es.write(string(buf.Line))
		}

		buf.Reset()
	}
}

/*
func (d *driver) ReadLogs(info logger.Info, config logger.ReadConfig) (io.ReadCloser, error) {
	d.mu.Lock()
	lf, exists := d.idx[info.ContainerID]
	d.mu.Unlock()
	if !exists {
		return nil, fmt.Errorf("logger does not exist for %s", info.ContainerID)
	}

	r, w := io.Pipe()
	lr, ok := lf.l.(logger.LogReader)
	if !ok {
		return nil, fmt.Errorf("logger does not support reading")
	}

	go func() {
		watcher := lr.ReadLogs(config)

		enc := protoio.NewUint32DelimitedWriter(w, binary.BigEndian)
		defer enc.Close()
		defer watcher.ConsumerGone()

		var buf logdriver.LogEntry
		for {
			select {
			case msg, ok := <-watcher.Msg:
				if !ok {
					w.Close()
					return
				}

				buf.Line = msg.Line
				buf.Partial = msg.PLogMetaData != nil
				buf.TimeNano = msg.Timestamp.UnixNano()
				buf.Source = msg.Source

				if err := enc.WriteMsg(&buf); err != nil {
					w.CloseWithError(err)
					return
				}
			case err := <-watcher.Err:
				w.CloseWithError(err)
				return
			}

			buf.Reset()
		}
	}()

	return r, nil
}
*/
