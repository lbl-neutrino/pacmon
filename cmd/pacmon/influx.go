package main

import (
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	write "github.com/influxdata/influxdb-client-go/v2/api/write"

	"larpix/pacmon/cmd/pacmon/monitors"
	. "larpix/pacmon/pkg"
)

func IoChannelToTileId(ioChannel int) int {
	return (ioChannel-1)/4 + 1
}

func makePoint(name string, timeNow time.Time) *write.Point {
	return influxdb2.NewPoint(name, nil, nil, timeNow)
}

func addTag(point *write.Point, key string, value uint8) {
	point.AddTag(key, strconv.Itoa(int(value)))
}

func addIoChannelTags(point *write.Point, ioChannel monitors.IoChannelKey) {
	addTag(point, "io_group", ioChannel.IoGroup)
	addTag(point, "io_channel", ioChannel.IoChannel)
	point.AddTag(
		"tile_id",
		strconv.Itoa(IoChannelToTileId(int(ioChannel.IoChannel))),
	)
}

func addChipTags(point *write.Point, chip monitors.ChipKey) {
	addIoChannelTags(
		point,
		monitors.IoChannelKey{IoGroup: chip.IoGroup, IoChannel: chip.IoChannel},
	)
	addTag(point, "chip", chip.ChipID)
}

func addChannelTags(point *write.Point, channel monitors.ChannelKey) {
	addChipTags(
		point,
		monitors.ChipKey{
			IoGroup:   channel.IoGroup,
			IoChannel: channel.IoChannel,
			ChipID:    channel.ChipID,
		},
	)
	addTag(point, "channel", channel.ChannelID)
}

func addDataStatusFields(
	point *write.Point,
	counts monitors.DataStatusCounts,
	timeDiff float64,
) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField("downstream", float64(counts.Downstream)/timeDiff)
	point.AddField("upstream", float64(counts.Upstream)/timeDiff)
}

func addConfigStatusFields(
	point *write.Point,
	counts monitors.ConfigStatusCounts,
	timeDiff float64,
) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField(
		"downstream_read",
		float64(counts.DownstreamRead)/timeDiff,
	)
	point.AddField(
		"downstream_write",
		float64(counts.DownstreamWrite)/timeDiff,
	)
	point.AddField("upstream_read", float64(counts.UpstreamRead)/timeDiff)
	point.AddField("upstream_write", float64(counts.UpstreamWrite)/timeDiff)
}

func addFifoFields(
	point *write.Point,
	lessHalfFull, moreHalfFull, full uint,
	total float64,
) {
	point.AddField("less_half_full", float64(lessHalfFull)/total)
	point.AddField("more_half_full", float64(moreHalfFull)/total)
	point.AddField("full", float64(full)/total)
}

func writeMonitorToInflux(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {

	// point := makePoint("word_types_rates", timeNow)
	// for wordtype, count := range m.WordTypeCounts[:] {
	// 	point.AddField(WordType(wordtype).String(), float64(count)/timeDiff)
	// }
	// writeAPI.WritePoint(point)

	writeDataStatusesRates(m, writeAPI, timeNow, timeDiff)
	writeConfigStatusesRates(m, writeAPI, timeNow, timeDiff)
	// writeOtherStatusesRates(m, writeAPI, timeNow, timeDiff)
	writeDataStatusesRatesPerChip(m, writeAPI, timeNow, timeDiff)
	writeConfigStatusesRatesPerChip(m, writeAPI, timeNow, timeDiff)
	// writeOtherStatusesRatesPerChip(m, writeAPI, timeNow, timeDiff)
	// writeLocalFifoStatuses(m, writeAPI, timeNow)
	// writeSharedFifoStatuses(m, writeAPI, timeNow)

	writeAPI.Flush()

}

func writeDataStatusesRates(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.DataStatusCounts[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("data_statuses_rates", timeNow)
		addIoChannelTags(point, monitors.IoChannelKeyAt(m.IoGroup, ioChannel))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeConfigStatusesRates(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.ConfigStatusCounts[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("config_statuses_rates", timeNow)
		addIoChannelTags(point, monitors.IoChannelKeyAt(m.IoGroup, ioChannel))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeOtherStatusesRates(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.OtherStatusCounts[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("other_statuses_rates", timeNow)
		addIoChannelTags(point, monitors.IoChannelKeyAt(m.IoGroup, ioChannel))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeDataStatusesRatesPerChip(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.DataStatusCountsPerChip[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("data_statuses_rates_per_chip", timeNow)
		addChipTags(point, monitors.ChipKeyAt(m.IoGroup, chip))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeConfigStatusesRatesPerChip(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.ConfigStatusCountsPerChip[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("config_statuses_rates_per_chip", timeNow)
		addChipTags(point, monitors.ChipKeyAt(m.IoGroup, chip))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeOtherStatusesRatesPerChip(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.OtherStatusCountsPerChip[:] {
		if !monitors.AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("other_statuses_rates_per_chip", timeNow)
		addChipTags(point, monitors.ChipKeyAt(m.IoGroup, chip))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeLocalFifoStatuses(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, counts := range m.FifoFlagCounts[:] {
		total := float64(
			counts.LocalFifoLessHalfFull +
				counts.LocalFifoMoreHalfFull +
				counts.LocalFifoFull,
		)
		if total == 0 {
			continue
		}

		point := makePoint("local_fifo_statuses", timeNow)
		addChannelTags(point, monitors.ChannelKeyAt(m.IoGroup, channel))
		addFifoFields(
			point,
			counts.LocalFifoLessHalfFull,
			counts.LocalFifoMoreHalfFull,
			counts.LocalFifoFull,
			total,
		)
		writeAPI.WritePoint(point)
	}
}

func writeSharedFifoStatuses(
	m *monitors.Monitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, counts := range m.FifoFlagCounts[:] {
		total := float64(
			counts.SharedFifoLessHalfFull +
				counts.SharedFifoMoreHalfFull +
				counts.SharedFifoFull,
		)
		if total == 0 {
			continue
		}

		point := makePoint("shared_fifo_statuses", timeNow)
		addChipTags(
			point,
			monitors.ChipKeyAt(m.IoGroup, channel/monitors.MAX_CHANNELS_PER_CHIP),
		)
		addFifoFields(
			point,
			counts.SharedFifoLessHalfFull,
			counts.SharedFifoMoreHalfFull,
			counts.SharedFifoFull,
			total,
		)
		writeAPI.WritePoint(point)
	}
}

func writeMonitor10sToInflux(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {

	// point := makePoint("packet_adc_total", timeNow)
	// point.AddField("adc_mean", m10s.ADCMeanTotal)
	// point.AddField("adc_rms", m10s.ADCRMSTotal)
	// point.AddField("n_packets", m10s.NPacketsTotal)
	// writeAPI.WritePoint(point)

	// writePacketAdcPerChannel(m10s, writeAPI, timeNow)
	writePacketAdcPerChip(m10s, writeAPI, timeNow)
	// writeDataStatusesRatesPerChannel(m10s, writeAPI, timeNow, timeDiff)
	// writeConfigStatusesRatesPerChannel(m10s, writeAPI, timeNow, timeDiff)
	// writeOtherStatusesRatesPerChannel(m10s, writeAPI, timeNow, timeDiff)
	writeTopDataRateChannels(m10s, writeAPI, timeNow, timeDiff)
	writeBottomDataRateChannels(m10s, writeAPI, timeNow, timeDiff)
	writeTopAdcMeanChannels(m10s, writeAPI, timeNow)
	writeTopAdcRmsChannels(m10s, writeAPI, timeNow)
	writeTopInvalidParityChips(m10s, writeAPI, timeNow, timeDiff)
	writeTopSharedFifoFullChips(m10s, writeAPI, timeNow, timeDiff)

	writeAPI.Flush()

}

func writePacketAdcPerChannel(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, adc := range m10s.ADCMeanPerChannel[:] {
		if m10s.NPacketsPerChannel[channel] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_channel", timeNow)
		addChannelTags(point, monitors.ChannelKeyAt(m10s.IoGroup, channel))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChannel[channel])
		point.AddField("n_packets", m10s.NPacketsPerChannel[channel])
		writeAPI.WritePoint(point)
	}
}

func writePacketAdcPerChip(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for chip, adc := range m10s.ADCMeanPerChip[:] {
		if m10s.NPacketsPerChip[chip] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_chip", timeNow)
		addChipTags(point, monitors.ChipKeyAt(m10s.IoGroup, chip))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChip[chip])
		// point.AddField("n_packets", m10s.NPacketsPerChip[chip])
		writeAPI.WritePoint(point)
	}
}

func writeDataStatusesRatesPerChannel(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.DataStatusCountsPerChannel[:] {
		if !monitors.AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("data_statuses_rates_per_channel", timeNow)
		addChannelTags(point, monitors.ChannelKeyAt(m10s.IoGroup, channel))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeConfigStatusesRatesPerChannel(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.ConfigStatusCountsPerChannel[:] {
		if !monitors.AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("config_statuses_rates_per_channel", timeNow)
		addChannelTags(point, monitors.ChannelKeyAt(m10s.IoGroup, channel))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeOtherStatusesRatesPerChannel(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.OtherStatusCountsPerChannel[:] {
		if !monitors.AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("other_statuses_rates_per_channel", timeNow)
		addChannelTags(point, monitors.ChannelKeyAt(m10s.IoGroup, channel))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeTopDataRateChannels(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, channel := range m10s.TopHotChannels {
		point := makePoint("top_data_rate_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("rate", float64(m10s.TopHotValues[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeBottomDataRateChannels(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, channel := range m10s.BottomColdChannels {
		point := makePoint("bottom_data_rate_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("rate", float64(m10s.BottomColdValues[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func writeTopAdcMeanChannels(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for i, channel := range m10s.TopADCMeanChannels {
		point := makePoint("top_adc_mean_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_mean", float64(m10s.TopADCMeanValues[i]))
		writeAPI.WritePoint(point)
	}
}

func writeTopAdcRmsChannels(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for i, channel := range m10s.TopADCRMSChannels {
		point := makePoint("top_adc_rms_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_rms", float64(m10s.TopADCRMSValues[i]))
		writeAPI.WritePoint(point)
	}
}

func writeTopInvalidParityChips(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, chip := range m10s.TopInvalidParityChips {
		point := makePoint("top_invalid_parity_chips", timeNow)
		addChipTags(point, chip)
		point.AddField(
			"total",
			float64(m10s.TopInvalidParityCounts[i])/timeDiff,
		)
		writeAPI.WritePoint(point)
	}
}

func writeTopSharedFifoFullChips(
	m10s *monitors.Monitor10s,
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, chip := range m10s.TopSharedFifoFullChips {
		point := makePoint("top_shared_fifo_full_chips", timeNow)
		addChipTags(point, chip)
		point.AddField(
			"total",
			float64(m10s.TopSharedFifoFullCounts[i])/timeDiff,
		)
		writeAPI.WritePoint(point)
	}
}

func writeSyncMonitorToInflux(
	sm *monitors.SyncMonitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	writeSync(sm, writeAPI, timeNow)
	writeAPI.Flush()
}

func writeSync(
	sm *monitors.SyncMonitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for ind, t := range sm.Time {
		point := makePoint("sync", timeNow)
		addTag(point, "io_group", sm.IoGroup[ind])

		if sm.Type[ind] == SyncTypeSync {
			point.AddField("sync", (float64(t)-1e7)*0.1)
		}
		if sm.Type[ind] == SyncTypeHeartbeat {
			point.AddField("heartbeat", float64(t))
		}
		if sm.Type[ind] == SyncTypeClkSource {
			point.AddField("clk_source", float64(t))
		}
		writeAPI.WritePoint(point)
	}
}

func writeTrigMonitorToInflux(
	tm *monitors.TrigMonitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	writeTrigger(tm, writeAPI, timeNow)
	writeAPI.Flush()
}

func writeTrigger(
	tm *monitors.TrigMonitor,
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for ind, t := range tm.Time {
		point := makePoint("trigger", timeNow)
		addTag(point, "io_group", tm.IoGroup[ind])
		point.AddField("trig", float64(t))
		writeAPI.WritePoint(point)
	}
}
