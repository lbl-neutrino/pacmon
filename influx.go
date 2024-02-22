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

func (m *Monitor) WriteToInflux(writeAPI api.WriteAPIBlocking, timeNow time.Time, timeDiff float64) {
	// TODO Set tile_id properly
	tile_id := 1
	tags := map[string]string{"tile_id": strconv.Itoa(tile_id)}

	makePoint := func (name string) *write.Point {
		return influxdb2.NewPoint(name, tags, nil, timeNow)
	}

	point := makePoint("word_types_rates")
	for wordtype, count := range m.WordTypeCounts {
		point.AddField(wordtype.String(), float64(count)/timeDiff)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range m.DataStatusCounts {
		point = makePoint("data_statuses_rates")

		point.AddTag("io_group", strconv.Itoa(int(ioChannel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel.IoChannel)))
		
		point.AddField("total", float64(counts.Total)/timeDiff)
		point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
		point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
		point.AddField("downstream", float64(counts.Downstream)/timeDiff)
		point.AddField("upstream", float64(counts.Upstream)/timeDiff)
		writeAPI.WritePoint(context.Background(), point)
	}

	for ioChannel, counts := range m.ConfigStatusCounts {
		point = makePoint("config_statuses_rates")

		point.AddTag("io_group", strconv.Itoa(int(ioChannel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel.IoChannel)))
		
		point.AddField("total", float64(counts.Total)/timeDiff)
		point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
		point.AddField("downstream_read", float64(counts.DownstreamRead)/timeDiff)
		point.AddField("downstream_write", float64(counts.DownstreamWrite)/timeDiff)
		point.AddField("upstream_read", float64(counts.UpstreamRead)/timeDiff)
		point.AddField("upstream_write", float64(counts.UpstreamWrite)/timeDiff)
		writeAPI.WritePoint(context.Background(), point)
	}

	for channel, counts := range m.FifoFlagCounts {
		total := float64(counts.LocalFifoLessHalfFull + counts.LocalFifoMoreHalfFull + counts.LocalFifoFull)
		if total == 0 {
			continue
		}

		point = makePoint("local_fifo_statuses")
		point.AddTag("io_group", strconv.Itoa(int(channel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(channel.IoChannel)))
		point.AddTag("chip", strconv.Itoa(int(channel.ChipID)))
		point.AddTag("channel", strconv.Itoa(int(channel.ChannelID)))

		point.AddField("less_half_full", float64(counts.LocalFifoLessHalfFull)/total)
		point.AddField("more_half_full", float64(counts.LocalFifoMoreHalfFull)/total)
		point.AddField("full", float64(counts.LocalFifoFull)/total)

		writeAPI.WritePoint(context.Background(), point)
	}

	for channel, counts := range m.FifoFlagCounts {
		total := float64(counts.SharedFifoLessHalfFull + counts.SharedFifoMoreHalfFull + counts.SharedFifoFull)
		if total == 0 {
			continue
		}

		point = makePoint("shared_fifo_statuses")
		point.AddTag("io_group", strconv.Itoa(int(channel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(channel.IoChannel)))
		point.AddTag("chip", strconv.Itoa(int(channel.ChipID)))

		point.AddField("less_half_full", float64(counts.SharedFifoLessHalfFull)/total)
		point.AddField("more_half_full", float64(counts.SharedFifoMoreHalfFull)/total)
		point.AddField("full", float64(counts.SharedFifoFull)/total)

		writeAPI.WritePoint(context.Background(), point)
	}


}

func (m10s *Monitor10s) WriteToInflux(writeAPI api.WriteAPIBlocking, timeNow time.Time, timeDiff float64) {
	// TODO Set tile_id properly
	tile_id := 1
	tags := map[string]string{"tile_id": strconv.Itoa(tile_id)}

	makePoint := func (name string) *write.Point {
		return influxdb2.NewPoint(name, tags, nil, timeNow)
	}

	point := makePoint("packet_adc_total")
	point.AddField("adc_mean", m10s.ADCMeanTotal)
	point.AddField("adc_rms", m10s.ADCRMSTotal)
	point.AddField("n_packets", m10s.NPacketsTotal)
	writeAPI.WritePoint(context.Background(), point)

	for channel, adc := range m10s.ADCMeanPerChannel {
		point = makePoint("packet_adc_per_channel")

		point.AddTag("io_group", strconv.Itoa(int(channel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(channel.IoChannel)))
		point.AddTag("chip", strconv.Itoa(int(channel.ChipID)))
		point.AddTag("channel", strconv.Itoa(int(channel.ChannelID)))

		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChannel[channel])
		point.AddField("n_packets", m10s.NPacketsPerChannel[channel])

		writeAPI.WritePoint(context.Background(), point)
	}

	for channel, counts := range m10s.DataStatusCountsPerChannel {
		point = makePoint("data_statuses_rates_per_channel")

		point.AddTag("io_group", strconv.Itoa(int(channel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(channel.IoChannel)))
		point.AddTag("chip", strconv.Itoa(int(channel.ChipID)))
		point.AddTag("channel", strconv.Itoa(int(channel.ChannelID)))

		point.AddField("total", float64(counts.Total)/timeDiff)
		point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
		point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
		point.AddField("downstream", float64(counts.Downstream)/timeDiff)
		point.AddField("upstream", float64(counts.Upstream)/timeDiff)

		writeAPI.WritePoint(context.Background(), point)
	}

	for channel, counts := range m10s.ConfigStatusCountsPerChannel {
		point = makePoint("config_statuses_rates_per_channel")

		point.AddTag("io_group", strconv.Itoa(int(channel.IoGroup)))
		point.AddTag("io_channel", strconv.Itoa(int(channel.IoChannel)))
		point.AddTag("chip", strconv.Itoa(int(channel.ChipID)))
		point.AddTag("channel", strconv.Itoa(int(channel.ChannelID)))

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
