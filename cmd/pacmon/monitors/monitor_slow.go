package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type SlowMonitor struct {
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

func NewSlowMonitor(ioGroup uint8) *SlowMonitor {
	return &SlowMonitor{IoGroup: ioGroup}
}

func (m *SlowMonitor) ProcessWord(word Word) {
	m.RecordStatuses(word)
	m.RecordADC(word)
	m.RecordChipStats(word)
}

func (m *SlowMonitor) RecordChipStats(word Word) {
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

func (m *SlowMonitor) RecordStatuses(word Word) {
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

func (m *SlowMonitor) RecordADC(word Word) {
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

	m.ADCMeanTotal, m.ADCRMSTotal = UpdateMeanRMS(
		m.ADCMeanTotal,
		m.ADCRMSTotal,
		m.NPacketsTotal,
		adc,
	)
	m.NPacketsTotal++

	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}
	chip := channel / MAX_CHANNELS_PER_CHIP

	m.ADCMeanPerChip[chip], m.ADCRMSPerChip[chip] = UpdateMeanRMS(
		m.ADCMeanPerChip[chip],
		m.ADCRMSPerChip[chip],
		m.NPacketsPerChip[chip],
		adc,
	)
	m.NPacketsPerChip[chip]++

	m.ADCMeanPerChannel[channel], m.ADCRMSPerChannel[channel] =
		UpdateMeanRMS(
			m.ADCMeanPerChannel[channel],
			m.ADCRMSPerChannel[channel],
			m.NPacketsPerChannel[channel],
			adc,
		)
	m.NPacketsPerChannel[channel]++

}

func (m *SlowMonitor) UpdateTopHotChannels() {
	ioGroup := m.IoGroup

	// A channel is reported if any packet was recorded for it
	totals := make([]uint, NUM_CHANNELS)
	for i, counts := range m.DataStatusCountsPerChannel[:] {
		totals[i] = counts.Total
	}
	hasStatuses := func(i int) bool {
		return AnyStatuses(
			m.DataStatusCountsPerChannel[i],
			m.ConfigStatusCountsPerChannel[i],
			m.OtherStatusCountsPerChannel[i],
		)
	}
	hasADC := func(i int) bool { return m.NPacketsPerChannel[i] > 0 }

	m.TopHotChannels, m.TopHotValues = sortByValue(
		totals, hasStatuses, 100, false, ioGroup, ChannelKeyAt,
	)
	m.BottomColdChannels, m.BottomColdValues = sortByValue(
		totals, hasStatuses, 100, true, ioGroup, ChannelKeyAt,
	)
	m.TopADCMeanChannels, m.TopADCMeanValues = sortByValue(
		m.ADCMeanPerChannel[:], hasADC, 100, false, ioGroup, ChannelKeyAt,
	)
	m.TopADCRMSChannels, m.TopADCRMSValues = sortByValue(
		m.ADCRMSPerChannel[:], hasADC, 100, false, ioGroup, ChannelKeyAt,
	)
}

func (m *SlowMonitor) UpdateTopChips() {
	ioGroup := m.IoGroup
	invalidParity := m.InvalidParityPerChip[:]
	sharedFifoFull := m.SharedFifoFullPerChip[:]

	m.TopInvalidParityChips, m.TopInvalidParityCounts = sortByValue(
		invalidParity,
		func(i int) bool { return invalidParity[i] > 0 },
		100,
		false,
		ioGroup,
		ChipKeyAt,
	)
	m.TopSharedFifoFullChips, m.TopSharedFifoFullCounts = sortByValue(
		sharedFifoFull,
		func(i int) bool { return sharedFifoFull[i] > 0 },
		100,
		false,
		ioGroup,
		ChipKeyAt,
	)
}

func (m *SlowMonitor) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	m.writePacketAdcTotal(writeAPI, timeNow)
	m.writePacketAdcPerChip(writeAPI, timeNow)
	m.writeTopDataRateChannels(writeAPI, timeNow, timeDiff)
	m.writeBottomDataRateChannels(writeAPI, timeNow, timeDiff)
	m.writeTopAdcMeanChannels(writeAPI, timeNow)
	m.writeTopAdcRmsChannels(writeAPI, timeNow)
	m.writeTopInvalidParityChips(writeAPI, timeNow, timeDiff)
	m.writeTopSharedFifoFullChips(writeAPI, timeNow, timeDiff)
}

func (m *SlowMonitor) writePacketAdcTotal(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	point := makePoint("packet_adc_total", timeNow)
	point.AddField("adc_mean", m.ADCMeanTotal)
	point.AddField("adc_rms", m.ADCRMSTotal)
	point.AddField("n_packets", m.NPacketsTotal)
	writeAPI.WritePoint(point)
}

func (m *SlowMonitor) writePacketAdcPerChip(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for chip, adc := range m.ADCMeanPerChip[:] {
		if m.NPacketsPerChip[chip] == 0 {
			continue
		}
		point := makePoint("packet_adc_per_chip", timeNow)
		addChipTags(point, ChipKeyAt(m.IoGroup, chip))
		point.AddField("adc_mean", adc)
		point.AddField("adc_rms", m.ADCRMSPerChip[chip])
		// point.AddField("n_packets", m.NPacketsPerChip[chip])
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeTopDataRateChannels(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, channel := range m.TopHotChannels {
		point := makePoint("top_data_rate_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("rate", float64(m.TopHotValues[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeBottomDataRateChannels(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, channel := range m.BottomColdChannels {
		point := makePoint("bottom_data_rate_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("rate", float64(m.BottomColdValues[i])/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeTopAdcMeanChannels(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for i, channel := range m.TopADCMeanChannels {
		point := makePoint("top_adc_mean_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_mean", float64(m.TopADCMeanValues[i]))
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeTopAdcRmsChannels(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for i, channel := range m.TopADCRMSChannels {
		point := makePoint("top_adc_rms_channels", timeNow)
		addChannelTags(point, channel)
		point.AddField("adc_rms", float64(m.TopADCRMSValues[i]))
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeTopInvalidParityChips(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, chip := range m.TopInvalidParityChips {
		point := makePoint("top_invalid_parity_chips", timeNow)
		addChipTags(point, chip)
		point.AddField(
			"total",
			float64(m.TopInvalidParityCounts[i])/timeDiff,
		)
		writeAPI.WritePoint(point)
	}
}

func (m *SlowMonitor) writeTopSharedFifoFullChips(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for i, chip := range m.TopSharedFifoFullChips {
		point := makePoint("top_shared_fifo_full_chips", timeNow)
		addChipTags(point, chip)
		point.AddField(
			"total",
			float64(m.TopSharedFifoFullCounts[i])/timeDiff,
		)
		writeAPI.WritePoint(point)
	}
}
