package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type ChanMonitor struct {
	IoGroup uint8

	ADCMeanPerChannel [NUM_CHANNELS]float64
	ADCRMSPerChannel  [NUM_CHANNELS]float64

	DataStatusCountsPerChannel   [NUM_CHANNELS]DataStatusCounts
	ConfigStatusCountsPerChannel [NUM_CHANNELS]ConfigStatusCounts
	OtherStatusCountsPerChannel  [NUM_CHANNELS]uint

	NPacketsPerChannel [NUM_CHANNELS]uint32
}

func NewChanMonitor(ioGroup uint8) *ChanMonitor {
	return &ChanMonitor{IoGroup: ioGroup}
}

func (m *ChanMonitor) ProcessWord(word Word) {
	m.RecordStatuses(word)
	m.RecordADC(word)
}

func (m *ChanMonitor) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	recordStatuses(
		m.DataStatusCountsPerChannel[:],
		m.ConfigStatusCountsPerChannel[:],
		m.OtherStatusCountsPerChannel[:],
		channel,
		pacData.Packet,
	)
}

func (m *ChanMonitor) RecordADC(word Word) {
	if word.Type != WordTypeData {
		return
	}
	pacData := word.PacData()
	if pacData.Packet.Type() != PacketTypeData { // Skip non-data
		return
	}
	if !pacData.Packet.ValidParity() { // Skip invalid parity
		return
	}

	packet := pacData.Packet
	adc := float64(packet.Data())

	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	m.ADCMeanPerChannel[channel], m.ADCRMSPerChannel[channel] =
		UpdateMeanRMS(
			m.ADCMeanPerChannel[channel],
			m.ADCRMSPerChannel[channel],
			m.NPacketsPerChannel[channel],
			adc,
		)
	m.NPacketsPerChannel[channel]++
}

func (m *ChanMonitor) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	m.writePacketAdcPerChannel(writeAPI, timeNow)
	m.writeDataStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	// m.writeConfigStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	// m.writeOtherStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
}

func (m *ChanMonitor) writeConfigStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m.ConfigStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChannel[channel],
			m.ConfigStatusCountsPerChannel[channel],
			m.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("config_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m.IoGroup, channel))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *ChanMonitor) writeOtherStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m.OtherStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChannel[channel],
			m.ConfigStatusCountsPerChannel[channel],
			m.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("other_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m.IoGroup, channel))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *ChanMonitor) writeDataStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m.DataStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChannel[channel],
			m.ConfigStatusCountsPerChannel[channel],
			m.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("data_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m.IoGroup, channel))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *ChanMonitor) writePacketAdcPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, adc := range m.ADCMeanPerChannel[:] {
		if m.NPacketsPerChannel[channel] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m.IoGroup, channel))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m.ADCRMSPerChannel[channel])
		point.AddField("n_packets", m.NPacketsPerChannel[channel])
		writeAPI.WritePoint(point)
	}
}
