[![Go Report Card](https://goreportcard.com/badge/github.com/diegohce/elasticlogger)](https://goreportcard.com/report/github.com/diegohce/elasticlogger)
[![GPLv3 license](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://github.com/diegohce/elasticlogger/blob/master/LICENSE)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/diegohce/elasticlogger/graphs/commit-activity)
[![HitCount](http://hits.dwyl.io/diegohce/elasticlogger.svg)](http://hits.dwyl.io/diegohce/elasticlogger)

# elasticlogger

# Build & install from source

```bash
git clone https://github.com/diegohce/elasticlogger.git ~/go/src/elasticlogger
cd ~/go/src/elasticlogger
make
```
## Pushing to registry

```bash
cd ~/go/src/elasticlogger
docker plugin create <registry>/elasticlogger:<tag> ,/plugin-dir
docker plugin push <registry>/elasticlogger:<tag>
```
## Installing from regisrtry
Make sure there's no previous elasticlogger installation from build process.
```bash
docker plugin ls
```
If there's any, remove them first.
```bash
docker plugin rm <plugin>:<tag>
```
Now, we can install `elasticlogger` from registry.
```bash
docker plugin install --alias elasticlogger <registry>/elasticlogger:<tag>
```
Optionally, you can set the `HOST` value at the same time.
```bash
docker plugin install --alias elasticlogger <registry>/elasticlogger:<tag> HOST=<elastichost:port>
```

# Configuration

```bash
docker plugin set elasticlogger:latest HOST=<elastic_host:port>
docker plugin enable elasticlogger:latest
```
## Mandatory plugin settings

<table>
<tr>
    <th>Option</th>
    <th>Description</th>
</tr>
<tr>
    <td>HOST</td>
    <td>Elasticsearch server host:port</td>
</tr>
</table>

## Optional plugin settings

<table>
<tr>
    <th>Option</th>
    <th>Description</th>
   <th>Default</th>
</tr>
<tr>
    <td>GCTIMER</td>
    <td>sets the garbage collector interval</td>
    <td>1m</td>
</tr>
<tr>
    <td>LOG_LEVEL</td>
    <td>sets the loglevel for the driver's own log entries</td>
    <td>info</td>
</tr>
</table>

# Usage

```bash
docker run --log-driver elasticlogger --log-opt index=myappindex ...
```

## Container level settings

<table>
<tr>
    <th>Option</th>
    <th>Description</th>
    <th>Default</th>
</tr>
<tr>
    <td>index</td>
    <td>Elasticsearch index where logs will be stored</td>
    <td>No default. Mandatory setting.</td>
</tr>
<tr>
    <td>host</td>
    <td>will override driver host</td>
    <td>plugin's HOST value</td>
</tr>
<tr>
    <td>bulksize</td>
    <td>sets how many lines of log to send at a time</td>
    <td>10</td>
</tr>
</table>


