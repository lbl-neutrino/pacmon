package main

type DataStatusCounts struct {
	Total uint
	ValidParity uint
	InvalidParity uint
	Downstream uint
	Upstream uint
}

type ConfigStatusCounts struct {
	Total uint
	InvalidParity uint
	DownstreamRead uint
	DownstreamWrite uint
	UpstreamRead uint
	UpstreamWrite uint
}

type FifoFlag uint8

type Channel struct {
	IoChannel IoChannel
	ChipID uint8
	ChannelID uint8
}

type FifoFlagCounts struct {
	LocalFifoLessHalfFull uint
	LocalFifoMoreHalfFull uint
	LocalFifoFull uint
	
	SharedFifoLessHalfFull uint
	SharedFifoMoreHalfFull uint
	SharedFifoFull uint
}

const (
	FifoLessHalfFull FifoFlag = 0
	FifoMoreHalfFull FifoFlag = 1
	FifoFull FifoFlag = 2
)

type Monitor struct {
	WordTypeCounts map[WordType]uint
	DataStatusCounts map[IoChannel]DataStatusCounts
	ConfigStatusCounts map[IoChannel]ConfigStatusCounts
	FifoFlagCounts map[Channel]FifoFlagCounts
}

type Monitor10s struct {
	ADCMeanTotal float64
	ADCRMSTotal float64

	ADCMeanPerChannel map[Channel]float64
	ADCRMSPerChannel map[Channel]float64

	NPacketsTotal uint32
	NPacketsPerChannel map[Channel]uint32

	DataStatusCountsPerChannel map[Channel]DataStatusCounts
	ConfigStatusCountsPerChannel map[Channel]ConfigStatusCounts
}

func NewMonitor() *Monitor {
	return &Monitor{
		WordTypeCounts: make(map[WordType]uint),
		DataStatusCounts: make(map[IoChannel]DataStatusCounts),
		ConfigStatusCounts: make(map[IoChannel]ConfigStatusCounts),
		FifoFlagCounts: make(map[Channel]FifoFlagCounts),
	}
}

func NewMonitor10s() *Monitor10s {
	return &Monitor10s{
		ADCMeanPerChannel: make(map[Channel]float64),
		ADCRMSPerChannel: make(map[Channel]float64),
		NPacketsPerChannel: make(map[Channel]uint32),
		DataStatusCountsPerChannel: make(map[Channel]DataStatusCounts),
		ConfigStatusCountsPerChannel: make(map[Channel]ConfigStatusCounts),
	}
}

func (m *Monitor) ProcessWord(word Word) {
	m.RecordType(word)
	m.RecordStatuses(word)
	m.RecordFifoFlags(word)
}

func (m10s *Monitor10s) ProcessWord(word Word) {
	m10s.RecordStatuses(word)
	m10s.RecordADC(word)
}

func (m *Monitor) RecordType(word Word) {
	newWordType := word.Type
	if word.Type == WordTypeData {
		packetType := word.PacData().Packet.Type()
		newWordType = PacketTypeMap[packetType]
	}
	m.WordTypeCounts[newWordType]++
}

func (m *Monitor) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()

	ioChannel := pacData.IoChannel

	var dataStatuses DataStatusCounts
	var configStatuses ConfigStatusCounts
	// Initialize with current values in monitor
	dataStatuses = m.DataStatusCounts[ioChannel]
	configStatuses = m.ConfigStatusCounts[ioChannel]

	packet := pacData.Packet
	isConfigRead := packet.Type() == PacketTypeRead
	isConfigWrite := packet.Type() == PacketTypeWrite
	isConfig := isConfigRead || isConfigWrite

	dataStatuses.Total++
	if isConfig {
		configStatuses.Total++
	}

	if packet.ValidParity() {
		dataStatuses.ValidParity++
	} else {
		dataStatuses.InvalidParity++
		configStatuses.InvalidParity++
	}

	if packet.Downstream() {
		dataStatuses.Downstream++
		if isConfigRead {
			configStatuses.DownstreamRead++
		} else if isConfigWrite {
			configStatuses.DownstreamWrite++
		}
	} else {
		dataStatuses.Upstream++
		if isConfigRead {
			configStatuses.UpstreamRead++
		} else if isConfigWrite {
			configStatuses.UpstreamWrite++
		}
	}

	// Update monitor
	m.DataStatusCounts[ioChannel] = dataStatuses
	m.ConfigStatusCounts[ioChannel] = configStatuses

}

func (m *Monitor) RecordFifoFlags(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet
	var channel Channel
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

func (m10s *Monitor10s) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet

	var channel Channel
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	var dataStatuses DataStatusCounts
	var configStatuses ConfigStatusCounts

	// Initialize with current values in monitor
	dataStatuses = m10s.DataStatusCountsPerChannel[channel]
	configStatuses = m10s.ConfigStatusCountsPerChannel[channel]

	isConfigRead := packet.Type() == PacketTypeRead
	isConfigWrite := packet.Type() == PacketTypeWrite
	isConfig := isConfigRead || isConfigWrite

	dataStatuses.Total++
	if isConfig {
		configStatuses.Total++
	}

	if packet.ValidParity() {
		dataStatuses.ValidParity++
	} else {
		dataStatuses.InvalidParity++
		configStatuses.InvalidParity++
	}

	if packet.Downstream() {
		dataStatuses.Downstream++
		if isConfigRead {
			configStatuses.DownstreamRead++
		} else if isConfigWrite {
			configStatuses.DownstreamWrite++
		}
	} else {
		dataStatuses.Upstream++
		if isConfigRead {
			configStatuses.UpstreamRead++
		} else if isConfigWrite {
			configStatuses.UpstreamWrite++
		}
	}

	// Update monitor
	m10s.DataStatusCountsPerChannel[channel] = dataStatuses
	m10s.ConfigStatusCountsPerChannel[channel] = configStatuses

}

func (m10s *Monitor10s) RecordADC(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	packet := pacData.Packet
	var channel Channel
	channel.IoChannel = pacData.IoChannel
	channel.ChipID = packet.Chip()
	channel.ChannelID = packet.Channel()

	adc := float64(packet.Data())

	m10s.ADCMeanTotal, m10s.ADCRMSTotal = UpdateMeanRMS(m10s.ADCMeanTotal, m10s.ADCRMSTotal, m10s.NPacketsTotal, adc)
	m10s.NPacketsTotal++

	m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel] = UpdateMeanRMS(m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel], m10s.NPacketsPerChannel[channel], adc)
	m10s.NPacketsPerChannel[channel]++

}