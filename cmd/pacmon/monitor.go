package main

import (
	. "larpix/pacmon/pkg"
)

type DataStatusCounts struct {
	Total         uint
	ValidParity   uint
	InvalidParity uint
	Downstream    uint
	Upstream      uint
}

type ConfigStatusCounts struct {
	Total           uint
	InvalidParity   uint
	DownstreamRead  uint
	DownstreamWrite uint
	UpstreamRead    uint
	UpstreamWrite   uint
}

type FifoFlag uint8

type ChannelKey struct {
	IoGroup   uint8
	IoChannel IoChannel
	ChipID    uint8
	ChannelID uint8
}

type ChipKey struct {
	IoGroup   uint8
	IoChannel IoChannel
	ChipID    uint8
}

type IoChannelKey struct {
	IoGroup   uint8
	IoChannel IoChannel
}

type FifoFlagCounts struct {
	LocalFifoLessHalfFull uint
	LocalFifoMoreHalfFull uint
	LocalFifoFull         uint

	SharedFifoLessHalfFull uint
	SharedFifoMoreHalfFull uint
	SharedFifoFull         uint
}

const (
	FifoLessHalfFull FifoFlag = 0
	FifoMoreHalfFull FifoFlag = 1
	FifoFull         FifoFlag = 2
)

type Monitor struct {
	WordTypeCounts map[WordType]uint

	DataStatusCounts        map[IoChannelKey]DataStatusCounts
	DataStatusCountsPerChip map[ChipKey]DataStatusCounts

	ConfigStatusCounts        map[IoChannelKey]ConfigStatusCounts
	ConfigStatusCountsPerChip map[ChipKey]ConfigStatusCounts

	OtherStatusCounts        map[IoChannelKey]uint
	OtherStatusCountsPerChip map[ChipKey]uint

	FifoFlagCounts map[ChannelKey]FifoFlagCounts
}

type Monitor10s struct {
	ADCMeanTotal float64
	ADCRMSTotal  float64

	ADCMeanPerChip map[ChipKey]float64
	ADCRMSPerChip  map[ChipKey]float64

	ADCMeanPerChannel map[ChannelKey]float64
	ADCRMSPerChannel  map[ChannelKey]float64

	NPacketsTotal      uint32
	NPacketsPerChip    map[ChipKey]uint32
	NPacketsPerChannel map[ChannelKey]uint32

	DataStatusCountsPerChannel   map[ChannelKey]DataStatusCounts
	ConfigStatusCountsPerChannel map[ChannelKey]ConfigStatusCounts
	OtherStatusCountsPerChannel  map[ChannelKey]uint

	TopHotChannels []ChannelKey
	TopHotValues   []uint

	TopADCMeanChannels []ChannelKey
	TopADCMeanValues   []float64

	TopADCRMSChannels []ChannelKey
	TopADCRMSValues   []float64
}

type MonitorPlots struct {
	ADCMeanPerChannel          map[ChannelKey]float64
	ADCRMSPerChannel           map[ChannelKey]float64
	NPacketsPerChannel         map[ChannelKey]uint32
	DataStatusCountsPerChannel map[ChannelKey]DataStatusCounts
}

type DisabledListMonitor struct {
	DataStatusCountsPerChannel map[ChannelKey]DataStatusCounts

	TopHotChannels []ChannelKey
	TopHotValues   []uint
}

type SyncMonitor struct {
	IoGroup []uint8
	Time    []uint32
	Type    []SyncType
}

type TrigMonitor struct {
	IoGroup []uint8
	Time    []uint32
}

func NewMonitor() *Monitor {
	return &Monitor{
		WordTypeCounts:            make(map[WordType]uint),
		DataStatusCounts:          make(map[IoChannelKey]DataStatusCounts),
		DataStatusCountsPerChip:   make(map[ChipKey]DataStatusCounts),
		ConfigStatusCounts:        make(map[IoChannelKey]ConfigStatusCounts),
		ConfigStatusCountsPerChip: make(map[ChipKey]ConfigStatusCounts),
		OtherStatusCounts:         make(map[IoChannelKey]uint),
		OtherStatusCountsPerChip:  make(map[ChipKey]uint),
		FifoFlagCounts:            make(map[ChannelKey]FifoFlagCounts),
	}
}

func NewMonitor10s() *Monitor10s {
	return &Monitor10s{
		ADCMeanPerChip:               make(map[ChipKey]float64),
		ADCRMSPerChip:                make(map[ChipKey]float64),
		NPacketsPerChip:              make(map[ChipKey]uint32),
		ADCMeanPerChannel:            make(map[ChannelKey]float64),
		ADCRMSPerChannel:             make(map[ChannelKey]float64),
		NPacketsPerChannel:           make(map[ChannelKey]uint32),
		DataStatusCountsPerChannel:   make(map[ChannelKey]DataStatusCounts),
		ConfigStatusCountsPerChannel: make(map[ChannelKey]ConfigStatusCounts),
		OtherStatusCountsPerChannel:  make(map[ChannelKey]uint),
	}
}
func NewMonitorPlots() *MonitorPlots {
	return &MonitorPlots{
		ADCMeanPerChannel:          make(map[ChannelKey]float64),
		ADCRMSPerChannel:           make(map[ChannelKey]float64),
		NPacketsPerChannel:         make(map[ChannelKey]uint32),
		DataStatusCountsPerChannel: make(map[ChannelKey]DataStatusCounts),
	}
}

func NewDisabledListMonitor() *DisabledListMonitor {
	return &DisabledListMonitor{
		DataStatusCountsPerChannel: make(map[ChannelKey]DataStatusCounts),
	}
}

func NewSyncMonitor() *SyncMonitor {
	return &SyncMonitor{}
}

func NewTrigMonitor() *TrigMonitor {
	return &TrigMonitor{}
}

func (m *Monitor) ProcessWord(word Word, ioGroup uint8) {
	m.RecordType(word)
	m.RecordStatuses(word, ioGroup)
	m.RecordFifoFlags(word, ioGroup)
}

func (m10s *Monitor10s) ProcessWord(word Word, ioGroup uint8) {
	m10s.RecordStatuses(word, ioGroup)
	m10s.RecordADC(word, ioGroup)
}

func (mPlots *MonitorPlots) ProcessWord(word Word, ioGroup uint8) {
	// mPlots.RecordStatuses(word, ioGroup)
	mPlots.RecordADC(word, ioGroup)
}

func (sm *SyncMonitor) ProcessWord(word Word, ioGroup uint8) {
	sm.RecordSync(word, ioGroup)
}

func (dlm *DisabledListMonitor) ProcessWord(word Word, ioGroup uint8) {
	dlm.RecordStatuses(word, ioGroup)
}

func (tm *TrigMonitor) ProcessWord(word Word, ioGroup uint8) {
	tm.RecordTrig(word, ioGroup)
}

func (m *Monitor) RecordType(word Word) {
	newWordType := word.Type
	if word.Type == WordTypeData {
		packetType := word.PacData().Packet.Type()
		newWordType = PacketTypeMap[packetType]
	}
	m.WordTypeCounts[newWordType]++
}

func (m *Monitor) RecordStatuses(word Word, ioGroup uint8) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet

	var ioChannelKey IoChannelKey
	ioChannelKey.IoGroup = ioGroup
	ioChannelKey.IoChannel = pacData.IoChannel

	var chipKey ChipKey
	chipKey.IoGroup = ioGroup
	chipKey.IoChannel = pacData.IoChannel
	chipKey.ChipID = packet.Chip()

	// Get current values in monitor
	dataStatuses := m.DataStatusCounts[ioChannelKey]
	configStatuses := m.ConfigStatusCounts[ioChannelKey]
	otherStatuses := m.OtherStatusCounts[ioChannelKey]

	dataStatusesPerChip := m.DataStatusCounts[ioChannelKey]
	configStatusesPerChip := m.ConfigStatusCounts[ioChannelKey]
	otherStatusesPerChip := m.OtherStatusCounts[ioChannelKey]

	isConfigRead := packet.Type() == PacketTypeRead
	isConfigWrite := packet.Type() == PacketTypeWrite
	isConfig := isConfigRead || isConfigWrite

	isData := packet.Type() == PacketTypeData

	if isConfig {

		configStatuses.Total++
		configStatusesPerChip.Total++

		if !packet.ValidParity() {
			configStatuses.InvalidParity++
			configStatusesPerChip.InvalidParity++
		}

		if packet.Downstream() {

			if isConfigRead {
				configStatuses.DownstreamRead++
				configStatusesPerChip.DownstreamRead++
			} else if isConfigWrite {
				configStatuses.DownstreamWrite++
				configStatusesPerChip.DownstreamWrite++
			}

		} else {

			if isConfigRead {
				configStatuses.UpstreamRead++
				configStatusesPerChip.UpstreamRead++
			} else if isConfigWrite {
				configStatuses.UpstreamWrite++
				configStatusesPerChip.UpstreamWrite++
			}

		}

	} else if isData {

		dataStatuses.Total++
		dataStatusesPerChip.Total++

		if packet.ValidParity() {
			dataStatuses.ValidParity++
			dataStatusesPerChip.ValidParity++
		} else {
			dataStatuses.InvalidParity++
			dataStatusesPerChip.InvalidParity++
		}

		if packet.Downstream() {
			dataStatuses.Downstream++
			dataStatusesPerChip.Downstream++
		} else {
			dataStatuses.Upstream++
			dataStatusesPerChip.Upstream++
		}

	} else {
		otherStatuses++
		otherStatusesPerChip++
	}

	// Update monitor

	m.DataStatusCounts[ioChannelKey] = dataStatuses
	m.ConfigStatusCounts[ioChannelKey] = configStatuses
	m.OtherStatusCounts[ioChannelKey] = otherStatuses

	m.DataStatusCountsPerChip[chipKey] = dataStatusesPerChip
	m.ConfigStatusCountsPerChip[chipKey] = configStatusesPerChip
	m.OtherStatusCountsPerChip[chipKey] = otherStatusesPerChip

}

func (m *Monitor) RecordFifoFlags(word Word, ioGroup uint8) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	if pacData.Packet.Type() != PacketTypeData { // Skip non-data
		return
	}

	packet := pacData.Packet
	var channel ChannelKey
	channel.IoGroup = ioGroup
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	fifoFlagCounts := m.FifoFlagCounts[channel]

	// Local FIFOs
	switch FifoFlag(packet.LocalFifoFlags()) {
	case FifoLessHalfFull:
		fifoFlagCounts.LocalFifoLessHalfFull++
	case FifoMoreHalfFull:
		fifoFlagCounts.LocalFifoMoreHalfFull++
	case FifoFull:
		fifoFlagCounts.LocalFifoFull++
	}

	// Shared FIFOs
	switch FifoFlag(packet.SharedFifoFlags()) {
	case FifoLessHalfFull:
		fifoFlagCounts.SharedFifoLessHalfFull++
	case FifoMoreHalfFull:
		fifoFlagCounts.SharedFifoMoreHalfFull++
	case FifoFull:
		fifoFlagCounts.SharedFifoFull++
	}

	// Update monitor
	m.FifoFlagCounts[channel] = fifoFlagCounts

}

func (m10s *Monitor10s) RecordStatuses(word Word, ioGroup uint8) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet

	var channel ChannelKey
	channel.IoGroup = ioGroup
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	// Get current values in monitor
	dataStatuses := m10s.DataStatusCountsPerChannel[channel]
	configStatuses := m10s.ConfigStatusCountsPerChannel[channel]
	otherStatuses := m10s.OtherStatusCountsPerChannel[channel]

	isConfigRead := packet.Type() == PacketTypeRead
	isConfigWrite := packet.Type() == PacketTypeWrite
	isConfig := isConfigRead || isConfigWrite

	isData := packet.Type() == PacketTypeData

	if isConfig {

		configStatuses.Total++

		if !packet.ValidParity() {
			configStatuses.InvalidParity++
		}

		if packet.Downstream() {

			if isConfigRead {
				configStatuses.DownstreamRead++
			} else if isConfigWrite {
				configStatuses.DownstreamWrite++
			}

		} else {

			if isConfigRead {
				configStatuses.UpstreamRead++
			} else if isConfigWrite {
				configStatuses.UpstreamWrite++
			}

		}

	} else if isData {

		dataStatuses.Total++

		if packet.ValidParity() {
			dataStatuses.ValidParity++
		} else {
			dataStatuses.InvalidParity++
		}

		if packet.Downstream() {
			dataStatuses.Downstream++
		} else {
			dataStatuses.Upstream++
		}

	} else {
		otherStatuses++
	}

	// Update monitor
	m10s.DataStatusCountsPerChannel[channel] = dataStatuses
	m10s.ConfigStatusCountsPerChannel[channel] = configStatuses
	m10s.OtherStatusCountsPerChannel[channel] = otherStatuses

}

func (m10s *Monitor10s) RecordADC(word Word, ioGroup uint8) {
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

	var channel ChannelKey
	channel.IoGroup = ioGroup
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	var chip ChipKey
	chip.IoGroup = ioGroup
	chip.IoChannel = pacData.IoChannel
	chip.ChipID = packet.Chip()

	adc := float64(packet.Data())

	m10s.ADCMeanTotal, m10s.ADCRMSTotal = UpdateMeanRMS(m10s.ADCMeanTotal, m10s.ADCRMSTotal, m10s.NPacketsTotal, adc)
	m10s.NPacketsTotal++

	m10s.ADCMeanPerChip[chip], m10s.ADCRMSPerChip[chip] = UpdateMeanRMS(m10s.ADCMeanPerChip[chip], m10s.ADCRMSPerChip[chip], m10s.NPacketsPerChip[chip], adc)
	m10s.NPacketsPerChip[chip]++

	m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel] = UpdateMeanRMS(m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel], m10s.NPacketsPerChannel[channel], adc)
	m10s.NPacketsPerChannel[channel]++

}

func (m10s *Monitor10s) UpdateTopHotChannels() {

	m10s.TopHotChannels, m10s.TopHotValues = sortByDataRates(m10s.DataStatusCountsPerChannel, 100)
	m10s.TopADCMeanChannels, m10s.TopADCMeanValues = sortByADC(m10s.ADCMeanPerChannel, 100)
	m10s.TopADCRMSChannels, m10s.TopADCRMSValues = sortByADC(m10s.ADCRMSPerChannel, 100)

}

func (mPlots *MonitorPlots) RecordADC(word Word, ioGroup uint8) {
	if word.Type != WordTypeData {
		return
	}
	pacData := word.PacData()
	if !pacData.Packet.ValidParity() { // Skip invalid parity
		return
	}
	if pacData.Packet.Type() != PacketTypeData { // Skip non-data
		return
	}
	packet := pacData.Packet

	var channel ChannelKey
	channel.IoGroup = ioGroup
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	adc := float64(packet.Data())

	mPlots.ADCMeanPerChannel[channel], mPlots.ADCRMSPerChannel[channel] = UpdateMeanRMS(mPlots.ADCMeanPerChannel[channel], mPlots.ADCRMSPerChannel[channel], mPlots.NPacketsPerChannel[channel], adc)
	mPlots.NPacketsPerChannel[channel]++

}

func (dlm *DisabledListMonitor) RecordStatuses(word Word, ioGroup uint8) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet

	var channel ChannelKey
	channel.IoGroup = ioGroup
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	// Get current values in monitor
	dataStatuses := dlm.DataStatusCountsPerChannel[channel]

	isData := packet.Type() == PacketTypeData

	if isData {

		dataStatuses.Total++

		if packet.ValidParity() {
			dataStatuses.ValidParity++
		} else {
			dataStatuses.InvalidParity++
		}

		if packet.Downstream() {
			dataStatuses.Downstream++
		} else {
			dataStatuses.Upstream++
		}

	} else {
		return
	}

	// Update monitor
	dlm.DataStatusCountsPerChannel[channel] = dataStatuses
}

func (sm *SyncMonitor) RecordSync(word Word, ioGroup uint8) {
	if word.Type == WordTypeSync {
		sm.Time = append(sm.Time, word.PacSync().Timestamp)
		sm.IoGroup = append(sm.IoGroup, ioGroup)
		sm.Type = append(sm.Type, word.PacSync().Type)
	}
}

func (tm *TrigMonitor) RecordTrig(word Word, ioGroup uint8) {
	if word.Type == WordTypeTrig {
		tm.Time = append(tm.Time, word.PacTrig().Timestamp)
		tm.IoGroup = append(tm.IoGroup, ioGroup)
	}
}
