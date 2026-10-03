package main

import (
	"strconv"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	write "github.com/influxdata/influxdb-client-go/v2/api/write"

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

func addIoChannelTags(point *write.Point, ioChannel IoChannelKey) {
	addTag(point, "io_group", ioChannel.IoGroup)
	addTag(point, "io_channel", ioChannel.IoChannel)
	point.AddTag("tile_id", strconv.Itoa(IoChannelToTileId(int(ioChannel.IoChannel))))
}

func addChipTags(point *write.Point, chip ChipKey) {
	addIoChannelTags(point, IoChannelKey{IoGroup: chip.IoGroup, IoChannel: chip.IoChannel})
	addTag(point, "chip", chip.ChipID)
}

func addChannelTags(point *write.Point, channel ChannelKey) {
	addChipTags(point, ChipKey{IoGroup: channel.IoGroup, IoChannel: channel.IoChannel, ChipID: channel.ChipID})
	addTag(point, "channel", channel.ChannelID)
}

func addDataStatusFields(point *write.Point, counts DataStatusCounts, timeDiff float64) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("valid_parity", float64(counts.ValidParity)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField("downstream", float64(counts.Downstream)/timeDiff)
	point.AddField("upstream", float64(counts.Upstream)/timeDiff)
}

func addConfigStatusFields(point *write.Point, counts ConfigStatusCounts, timeDiff float64) {
	point.AddField("total", float64(counts.Total)/timeDiff)
	point.AddField("invalid_parity", float64(counts.InvalidParity)/timeDiff)
	point.AddField("downstream_read", float64(counts.DownstreamRead)/timeDiff)
	point.AddField("downstream_write", float64(counts.DownstreamWrite)/timeDiff)
	point.AddField("upstream_read", float64(counts.UpstreamRead)/timeDiff)
	point.AddField("upstream_write", float64(counts.UpstreamWrite)/timeDiff)
}

func addFifoFields(point *write.Point, lessHalfFull, moreHalfFull, full uint, total float64) {
	point.AddField("less_half_full", float64(lessHalfFull)/total)
	point.AddField("more_half_full", float64(moreHalfFull)/total)
	point.AddField("full", float64(full)/total)
}

func (m *Monitor) WriteToInflux(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {

	point := makePoint("word_types_rates", timeNow)
	for wordtype, count := range m.WordTypeCounts {
		point.AddField(wordtype.String(), float64(count)/timeDiff)
	}
	writeAPI.WritePoint(point)

	m.writeDataStatusesRates(writeAPI, timeNow, timeDiff)
	m.writeConfigStatusesRates(writeAPI, timeNow, timeDiff)
	m.writeOtherStatusesRates(writeAPI, timeNow, timeDiff)
	m.writeDataStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	m.writeConfigStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	m.writeOtherStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	m.writeLocalFifoStatuses(writeAPI, timeNow)
	m.writeSharedFifoStatuses(writeAPI, timeNow)

	writeAPI.Flush()

}

func (m *Monitor) writeDataStatusesRates(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for ioChannel, counts := range m.DataStatusCounts {
		point := makePoint("data_statuses_rates", timeNow)
		addIoChannelTags(point, ioChannel)
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeConfigStatusesRates(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for ioChannel, counts := range m.ConfigStatusCounts {
		point := makePoint("config_statuses_rates", timeNow)
		addIoChannelTags(point, ioChannel)
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeOtherStatusesRates(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for ioChannel, counts := range m.OtherStatusCounts {
		point := makePoint("other_statuses_rates", timeNow)
		addIoChannelTags(point, ioChannel)
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeDataStatusesRatesPerChip(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for chip, counts := range m.DataStatusCountsPerChip {
		point := makePoint("data_statuses_rates_per_chip", timeNow)
		addChipTags(point, chip)
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeConfigStatusesRatesPerChip(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for chip, counts := range m.ConfigStatusCountsPerChip {
		point := makePoint("config_statuses_rates_per_chip", timeNow)
		addChipTags(point, chip)
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeOtherStatusesRatesPerChip(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for chip, counts := range m.OtherStatusCountsPerChip {
		point := makePoint("other_statuses_rates_per_chip", timeNow)
		addChipTags(point, chip)
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeLocalFifoStatuses(writeAPI api.WriteAPI, timeNow time.Time) {
	for channel, counts := range m.FifoFlagCounts {
		total := float64(counts.LocalFifoLessHalfFull + counts.LocalFifoMoreHalfFull + counts.LocalFifoFull)
		if total == 0 {
			continue
		}

		point := makePoint("local_fifo_statuses", timeNow)
		addChannelTags(point, channel)
		addFifoFields(point, counts.LocalFifoLessHalfFull, counts.LocalFifoMoreHalfFull, counts.LocalFifoFull, total)
		writeAPI.WritePoint(point)
	}
}

func (m *Monitor) writeSharedFifoStatuses(writeAPI api.WriteAPI, timeNow time.Time) {
	for channel, counts := range m.FifoFlagCounts {
		total := float64(counts.SharedFifoLessHalfFull + counts.SharedFifoMoreHalfFull + counts.SharedFifoFull)
		if total == 0 {
			continue
		}

		point := makePoint("shared_fifo_statuses", timeNow)
		addChipTags(point, ChipKey{IoGroup: channel.IoGroup, IoChannel: channel.IoChannel, ChipID: channel.ChipID})
		addFifoFields(point, counts.SharedFifoLessHalfFull, counts.SharedFifoMoreHalfFull, counts.SharedFifoFull, total)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) WriteToInflux(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {

	point := makePoint("packet_adc_total", timeNow)
	point.AddField("adc_mean", m10s.ADCMeanTotal)
	point.AddField("adc_rms", m10s.ADCRMSTotal)
	point.AddField("n_packets", m10s.NPacketsTotal)
	writeAPI.WritePoint(point)

	m10s.writePacketAdcPerChannel(writeAPI, timeNow)
	m10s.writePacketAdcPerChip(writeAPI, timeNow)
	m10s.writeDataStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	m10s.writeConfigStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	m10s.writeOtherStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	m10s.writeTopDataRateChannels(writeAPI, timeNow, timeDiff)
	m10s.writeTopAdcMeanChannels(writeAPI, timeNow)
	m10s.writeTopAdcRmsChannels(writeAPI, timeNow)
	m10s.writeTopInvalidParityChips(writeAPI, timeNow, timeDiff)
	m10s.writeTopSharedFifoFullChips(writeAPI, timeNow, timeDiff)

	writeAPI.Flush()

}

func (m10s *Monitor10s) writePacketAdcPerChannel(writeAPI api.WriteAPI, timeNow time.Time) {
	for channel, adc := range m10s.ADCMeanPerChannel {
		point := makePoint("packet_adc_per_channel", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChannel[channel])
		point.AddField("n_packets", m10s.NPacketsPerChannel[channel])
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writePacketAdcPerChip(writeAPI api.WriteAPI, timeNow time.Time) {
	for chip, adc := range m10s.ADCMeanPerChip {
		point := makePoint("packet_adc_per_chip", timeNow)
		addChipTags(point, chip)
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChip[chip])
		point.AddField("n_packets", m10s.NPacketsPerChip[chip])
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeDataStatusesRatesPerChannel(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for channel, counts := range m10s.DataStatusCountsPerChannel {
		point := makePoint("data_statuses_rates_per_channel", timeNow)
		addChannelTags(point, channel)
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeConfigStatusesRatesPerChannel(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for channel, counts := range m10s.ConfigStatusCountsPerChannel {
		point := makePoint("config_statuses_rates_per_channel", timeNow)
		addChannelTags(point, channel)
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeOtherStatusesRatesPerChannel(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for channel, counts := range m10s.OtherStatusCountsPerChannel {
		point := makePoint("other_statuses_rates_per_channel", timeNow)
		addChannelTags(point, channel)
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopDataRateChannels(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for i, channel := range m10s.TopHotChannels {
		point := makePoint("top_data_rate_channels", timeNow)
		// NOTE: Unlike the other measurements, this one has no tile_id tag
		addTag(point, "io_group", channel.IoGroup)
		addTag(point, "io_channel", channel.IoChannel)
		addTag(point, "chip", channel.ChipID)
		addTag(point, "channel", channel.ChannelID)
		point.AddField("rate", float64(m10s.TopHotValues[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopAdcMeanChannels(writeAPI api.WriteAPI, timeNow time.Time) {
	for i, channel := range m10s.TopADCMeanChannels {
		point := makePoint("top_adc_mean_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_mean", float64(m10s.TopADCMeanValues[i]))
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopAdcRmsChannels(writeAPI api.WriteAPI, timeNow time.Time) {
	for i, channel := range m10s.TopADCRMSChannels {
		point := makePoint("top_adc_rms_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_rms", float64(m10s.TopADCRMSValues[i]))
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopInvalidParityChips(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for i, chip := range m10s.TopInvalidParityChips {
		point := makePoint("top_invalid_parity_chips", timeNow)
		addChipTags(point, chip)
		point.AddField("total", float64(m10s.TopInvalidParityCounts[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopSharedFifoFullChips(writeAPI api.WriteAPI, timeNow time.Time, timeDiff float64) {
	for i, chip := range m10s.TopSharedFifoFullChips {
		point := makePoint("top_shared_fifo_full_chips", timeNow)
		addChipTags(point, chip)
		point.AddField("total", float64(m10s.TopSharedFifoFullCounts[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (sm *SyncMonitor) WriteToInflux(writeAPI api.WriteAPI, timeNow time.Time) {
	sm.writeSync(writeAPI, timeNow)
	writeAPI.Flush()
}

func (sm *SyncMonitor) writeSync(writeAPI api.WriteAPI, timeNow time.Time) {
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

func (tm *TrigMonitor) WriteToInflux(writeAPI api.WriteAPI, timeNow time.Time) {
	tm.writeTrigger(writeAPI, timeNow)
	writeAPI.Flush()
}

func (tm *TrigMonitor) writeTrigger(writeAPI api.WriteAPI, timeNow time.Time) {
	for ind, t := range tm.Time {
		point := makePoint("trigger", timeNow)
		addTag(point, "io_group", tm.IoGroup[ind])
		point.AddField("trig", float64(t))
		writeAPI.WritePoint(point)
	}
}
