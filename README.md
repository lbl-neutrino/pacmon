# pacmon: The PACMAN Monitor

`pacmon` subscribes via ZeroMQ to the data and command servers of one or more
PACMAN cards. Various data quality metrics are calculated from these streams and
loaded into an InfluxDB for further display using e.g. Grafana.

## Setup

### Installing Go

First, install the toolchain for the Go programming language:

``` bash
goURL="https://go.dev/dl/go1.21.4.linux-amd64.tar.gz"
mkdir -p ~/.local && curl -L "$goURL" | tar zxf - -C ~/.local
```

To enable the toolchain (only needed for building `pacmon`, not running it):

``` bash
export PATH=~/.local/go/bin:$PATH
```

If you want, put this in your `~/.bashrc` to automatically enable the toolchain
upon login.

### Building `pacmon`

``` bash
git clone https://github.com/lbl-neutrino/pacmon.git
cd pacmon
go build -o pacmon .
```

This will create the `pacmon` executable.

## Usage

Before running `pacmon`, do

``` bash
export INFLUXDB_TOKEN=...
```

where `...` should be replaced by your InfluxDB API token.

The connection parameters (for PACMAN and InfluxDB) can be controlled via
command-line options. Run `./pacmon --help` for a full list of options. To 
specify multiple PACMAN boards, use the command-line options `--pacman-url` 
and `--pacman-iog` to specify the the connection parameters for PACMAN cards 
and the corresponding IO group. For instance:

``` bash
./pacmon \
    --pacman-url "tcp://pacman35.local:5556,tcp://pacman22.local:5556" \
    --pacman-iog "1,2"
```


If a [JSON IO configuration file](https://github.com/larpix/crs_daq/blob/master/io/pacman.json) is available, containing the mapping between 
pacman addresses and IO groups, it can simply be provided with the command-line 
option `--pacman-config` (instead of `--pacman-url` and `--pacman-iog`): 

``` bash
./pacmon --pacman-config path/to/crs_daq/io/pacman.json
```

## Existing metrics

Currently `pacmon` writes the same set of metrics displayed by Kevin's original
`pacmon.py`. See `monitor.go` for how they're calculated.

## Adding metrics

In `monitor.go`, first add the necessary data fields (e.g. `NoiseRate`) to the
`Monitor` structure. If you're adding something that requires initialization,
such as a `map`, then also edit the `NewMonitor` function.

Then add a function like

``` go
func (m *Monitor) RecordNoiseRate(word Word) {
    // do stuff and finally set m.NoiseRate
}
```

and edit the `ProcessWord` function so that it calls `RecordNoiseRate`. Finally,
in `influx.go`, edit the `WriteToInflux` function so that it writes a data point
containing the noise rate.

Once patterns start emerging in the code, we can switch to a more modular
structure that doesn't require editing so many existing functions.

## Copyright and Licensing

Copyright © 2023 FERMI NATIONAL ACCELERATOR LABORATORY for the benefit of the
DUNE Collaboration.

This repository, and all software contained within, is licensed under
the Apache License, Version 2.0 (the "License"); you may not use this
file except in compliance with the License. You may obtain a copy of
the License at

    http://www.apache.org/licenses/LICENSE-2.0

Copyright is granted to FERMI NATIONAL ACCELERATOR LABORATORY on behalf
of the Deep Underground Neutrino Experiment (DUNE). Unless required by
applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for
the specific language governing permissions and limitations under the
License.
