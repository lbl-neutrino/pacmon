package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type Monitor10s struct {
	IoGroup uint8

	ADCMeanTotal float64
	ADCRMSTotal  float64

	ADCMeanPerChip [NUM_CHIPS]float64
	ADCRMSPerChip  [NUM_CHIPS]float64

	ADCMeanPerChannel [NUM_CHANNELS]float64
	ADCRMSPerChannel  [NUM_CHANNELS]float64

	NPacketsTotal      uint32
	NPacketsPerChip    [NUM_CHIPS]uint32
	NPacketsPerChannel [NUM_CHANNELS]uint32

	DataStatusCountsPerChannel   [NUM_CHANNELS]DataStatusCounts
	ConfigStatusCountsPerChannel [NUM_CHANNELS]ConfigStatusCounts
	OtherStatusCountsPerChannel  [NUM_CHANNELS]uint

	TopHotChannels []ChannelKey
	TopHotValues   []uint

	BottomColdChannels []ChannelKey
	BottomColdValues   []uint

	TopADCMeanChannels []ChannelKey
	TopADCMeanValues   []float64

	TopADCRMSChannels []ChannelKey
	TopADCRMSValues   []float64

	InvalidParityPerChip   [NUM_CHIPS]uint32
	TopInvalidParityChips  []ChipKey
	TopInvalidParityCounts []uint32

	SharedFifoFullPerChip   [NUM_CHIPS]uint32
	TopSharedFifoFullChips  []ChipKey
	TopSharedFifoFullCounts []uint32
}

func NewMonitor10s(ioGroup uint8) *Monitor10s {
	return &Monitor10s{IoGroup: ioGroup}
}

func (m10s *Monitor10s) ProcessWord(word Word) {
	m10s.RecordStatuses(word)
	m10s.RecordADC(word)
	m10s.RecordChipStats(word)
}

func (m *Monitor10s) RecordChipStats(word Word) {
	pacData := word.PacData()
	packet := pacData.Packet
	chip, ok := chipIndex(pacData)
	if !ok {
		return
	}

	isInvalid := !packet.ValidParity()
	flag := FifoFlag(packet.SharedFifoFlags())
	isSharedFifoFull := flag == FifoFull1 || flag == FifoFull2

	if isInvalid {
		m.InvalidParityPerChip[chip]++
	}

	if isSharedFifoFull {
		m.SharedFifoFullPerChip[chip]++
	}
}

func (m10s *Monitor10s) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	recordStatuses(
		m10s.DataStatusCountsPerChannel[:],
		m10s.ConfigStatusCountsPerChannel[:],
		m10s.OtherStatusCountsPerChannel[:],
		channel,
		pacData.Packet,
	)
}

func (m10s *Monitor10s) RecordADC(word Word) {
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

	m10s.ADCMeanTotal, m10s.ADCRMSTotal = UpdateMeanRMS(
		m10s.ADCMeanTotal,
		m10s.ADCRMSTotal,
		m10s.NPacketsTotal,
		adc,
	)
	m10s.NPacketsTotal++

	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}
	chip := channel / MAX_CHANNELS_PER_CHIP

	m10s.ADCMeanPerChip[chip], m10s.ADCRMSPerChip[chip] = UpdateMeanRMS(
		m10s.ADCMeanPerChip[chip],
		m10s.ADCRMSPerChip[chip],
		m10s.NPacketsPerChip[chip],
		adc,
	)
	m10s.NPacketsPerChip[chip]++

	m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel] =
		UpdateMeanRMS(
			m10s.ADCMeanPerChannel[channel],
			m10s.ADCRMSPerChannel[channel],
			m10s.NPacketsPerChannel[channel],
			adc,
		)
	m10s.NPacketsPerChannel[channel]++

}

func (m10s *Monitor10s) UpdateTopHotChannels() {
	ioGroup := m10s.IoGroup

	// A channel is reported if any packet was recorded for it
	totals := make([]uint, NUM_CHANNELS)
	for i, counts := range m10s.DataStatusCountsPerChannel[:] {
		totals[i] = counts.Total
	}
	hasStatuses := func(i int) bool {
		return AnyStatuses(
			m10s.DataStatusCountsPerChannel[i],
			m10s.ConfigStatusCountsPerChannel[i],
			m10s.OtherStatusCountsPerChannel[i],
		)
	}
	hasADC := func(i int) bool { return m10s.NPacketsPerChannel[i] > 0 }

	m10s.TopHotChannels, m10s.TopHotValues = sortByValue(
		totals, hasStatuses, 100, false, ioGroup, ChannelKeyAt,
	)
	m10s.BottomColdChannels, m10s.BottomColdValues = sortByValue(
		totals, hasStatuses, 100, true, ioGroup, ChannelKeyAt,
	)
	m10s.TopADCMeanChannels, m10s.TopADCMeanValues = sortByValue(
		m10s.ADCMeanPerChannel[:], hasADC, 100, false, ioGroup, ChannelKeyAt,
	)
	m10s.TopADCRMSChannels, m10s.TopADCRMSValues = sortByValue(
		m10s.ADCRMSPerChannel[:], hasADC, 100, false, ioGroup, ChannelKeyAt,
	)
}

func (m10s *Monitor10s) UpdateTopChips() {
	ioGroup := m10s.IoGroup
	invalidParity := m10s.InvalidParityPerChip[:]
	sharedFifoFull := m10s.SharedFifoFullPerChip[:]

	m10s.TopInvalidParityChips, m10s.TopInvalidParityCounts = sortByValue(
		invalidParity,
		func(i int) bool { return invalidParity[i] > 0 },
		100,
		false,
		ioGroup,
		ChipKeyAt,
	)
	m10s.TopSharedFifoFullChips, m10s.TopSharedFifoFullCounts = sortByValue(
		sharedFifoFull,
		func(i int) bool { return sharedFifoFull[i] > 0 },
		100,
		false,
		ioGroup,
		ChipKeyAt,
	)
}

func (m10s *Monitor10s) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {

	// point := makePoint("packet_adc_total", timeNow)
	// point.AddField("adc_mean", m10s.ADCMeanTotal)
	// point.AddField("adc_rms", m10s.ADCRMSTotal)
	// point.AddField("n_packets", m10s.NPacketsTotal)
	// writeAPI.WritePoint(point)

	// m10s.writePacketAdcPerChannel(writeAPI, timeNow)
	m10s.writePacketAdcPerChip(writeAPI, timeNow)
	// m10s.writeDataStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	// m10s.writeConfigStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	// m10s.writeOtherStatusesRatesPerChannel(writeAPI, timeNow, timeDiff)
	m10s.writeTopDataRateChannels(writeAPI, timeNow, timeDiff)
	m10s.writeBottomDataRateChannels(writeAPI, timeNow, timeDiff)
	m10s.writeTopAdcMeanChannels(writeAPI, timeNow)
	m10s.writeTopAdcRmsChannels(writeAPI, timeNow)
	m10s.writeTopInvalidParityChips(writeAPI, timeNow, timeDiff)
	m10s.writeTopSharedFifoFullChips(writeAPI, timeNow, timeDiff)

	writeAPI.Flush()

}

func (m10s *Monitor10s) writePacketAdcPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, adc := range m10s.ADCMeanPerChannel[:] {
		if m10s.NPacketsPerChannel[channel] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m10s.IoGroup, channel))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChannel[channel])
		point.AddField("n_packets", m10s.NPacketsPerChannel[channel])
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writePacketAdcPerChip(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for chip, adc := range m10s.ADCMeanPerChip[:] {
		if m10s.NPacketsPerChip[chip] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_chip", timeNow)
		addChipTags(point, ChipKeyAt(m10s.IoGroup, chip))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m10s.ADCRMSPerChip[chip])
		// point.AddField("n_packets", m10s.NPacketsPerChip[chip])
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeDataStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.DataStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("data_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m10s.IoGroup, channel))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeConfigStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.ConfigStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("config_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m10s.IoGroup, channel))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeOtherStatusesRatesPerChannel(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for channel, counts := range m10s.OtherStatusCountsPerChannel[:] {
		if !AnyStatuses(
			m10s.DataStatusCountsPerChannel[channel],
			m10s.ConfigStatusCountsPerChannel[channel],
			m10s.OtherStatusCountsPerChannel[channel],
		) {
			continue
		}
		point := makePoint("other_statuses_rates_per_channel", timeNow)
		addChannelTags(point, ChannelKeyAt(m10s.IoGroup, channel))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m10s *Monitor10s) writeTopDataRateChannels(
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

func (m10s *Monitor10s) writeBottomDataRateChannels(
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

func (m10s *Monitor10s) writeTopAdcMeanChannels(
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

func (m10s *Monitor10s) writeTopAdcRmsChannels(
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

func (m10s *Monitor10s) writeTopInvalidParityChips(
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

func (m10s *Monitor10s) writeTopSharedFifoFullChips(
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
