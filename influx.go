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

func (m *Monitor) WriteToInflux(writeAPI api.WriteAPIBlocking) {
	// TODO Set tile_id properly
	tile_id := 1
	tags := map[string]string{"tile_id": strconv.Itoa(tile_id)}

	makePoint := func (name string) *write.Point {
		return influxdb2.NewPoint(name, tags, nil, time.Now())
	}

	point := makePoint("word_types")
	for wordtype, count := range m.WordTypeCounts {
		point.AddField(wordtype.String(), count)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range m.DataStatusCounts {
		point = makePoint("data_statuses")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("total", counts.Total)
		point.AddField("valid_parity", counts.ValidParity)
		point.AddField("invalid_parity", counts.InvalidParity)
		point.AddField("downstream", counts.Downstream)
		point.AddField("upstream", counts.Upstream)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range m.ConfigStatusCounts {
		point = makePoint("config_statuses")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("total", counts.Total)
		point.AddField("invalid_parity", counts.InvalidParity)
		point.AddField("downstream_read", counts.DownstreamRead)
		point.AddField("downstream_write", counts.DownstreamWrite)
		point.AddField("upstream_read", counts.UpstreamRead)
		point.AddField("upstream_write", counts.UpstreamWrite)
	}
	writeAPI.WritePoint(context.Background(), point)
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
