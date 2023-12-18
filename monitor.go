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

func NewMonitor() *Monitor {
	return &Monitor{
		WordTypeCounts: make(map[WordType]uint),
		DataStatusCounts: make(map[IoChannel]DataStatusCounts),
		ConfigStatusCounts: make(map[IoChannel]ConfigStatusCounts),
		FifoFlagCounts: make(map[Channel]FifoFlagCounts),
	}
}

func (m *Monitor) ProcessWord(word Word) {
	m.RecordType(word)
	m.RecordStatuses(word)
	m.RecordFifoFlags(word)
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

	// Initialize with current values in monitor
	dataStatuses := m.DataStatusCounts[ioChannel]
	configStatuses := m.ConfigStatusCounts[ioChannel]

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