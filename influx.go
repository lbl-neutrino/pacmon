package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

func (m *Monitor) WriteToInflux(writeAPI api.WriteAPIBlocking, timeDiff float64) {
	// TODO Set tile_id properly
	tile_id := 1
	tags := map[string]string{"tile_id": strconv.Itoa(tile_id)}

	makePoint := func (name string) *write.Point {
		return influxdb2.NewPoint(name, tags, nil, time.Now())
	}

	point := makePoint("word_types_rates")
	for wordtype, count := range m.WordTypeCounts {
		point.AddField(wordtype.String(), float64(count)/timeDiff)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range m.DataStatusCounts {
		point = makePoint("data_statuses_rates")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("total", float64(counts.Total)/timeDiff)
		point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
		point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
		point.AddField("downstream", float64(counts.Downstream)/timeDiff)
		point.AddField("upstream", float64(counts.Upstream)/timeDiff)
		writeAPI.WritePoint(context.Background(), point)
	}

	for ioChannel, counts := range m.ConfigStatusCounts {
		point = makePoint("config_statuses_rates")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("total", float64(counts.Total)/timeDiff)
		point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
		point.AddField("downstream_read", float64(counts.DownstreamRead)/timeDiff)
		point.AddField("downstream_write", float64(counts.DownstreamWrite)/timeDiff)
		point.AddField("upstream_read", float64(counts.UpstreamRead)/timeDiff)
		point.AddField("upstream_write", float64(counts.UpstreamWrite)/timeDiff)
		writeAPI.WritePoint(context.Background(), point)
	}
}

func getWriteAPI(url, org, bucket string) api.WriteAPIBlocking {
	token := os.Getenv("INFLUXDB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr,
			"Please set the INFLUXDB_TOKEN environment variable\n")
		os.Exit(1)
	}
	client := influxdb2.NewClient(url, token)
	return client.WriteAPIBlocking(org, bucket)
}
