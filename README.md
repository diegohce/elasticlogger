[![Go Report Card](https://goreportcard.com/badge/github.com/diegohce/elasticlogger)](https://goreportcard.com/report/github.com/diegohce/elasticlogger)
[![GPLv3 license](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://github.com/diegohce/elasticlogger/blob/master/LICENSE)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/diegohce/elasticlogger/graphs/commit-activity)
[![HitCount](http://hits.dwyl.io/diegohce/elasticlogger.svg)](http://hits.dwyl.io/diegohce/elasticlogger)

# elasticlogger

# Build & install

```bash
git clone https://github.com/diegohce/elasticlogger.git ~/go/src/elasticlogger
cd ~/go/src/elasticlogger
make
```

# Configuration

```bash
docker plugin set elasticlogger:latest HOST=<elastic_host:port>
docker plugin enable elasticlogger:latest
```
## Optional

- `GCTIMER` sets the garbage collector interval. Default: 1m (one minute).
- `LOG_LEVEL` sets the loglevel for the driver's own log entries. Default: info


# Usage

```bash
docker run --log-driver elasticlogger --log-opt index=myappindex ...
```

Other log options:
- host : will override driver host.
- bulksize: sets how many lines of log to send at a time. Default 10.


