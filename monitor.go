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

type FifoFlags struct {
	LocalFifoFlags uint8
	SharedFifoFlags uint8
}

type Monitor struct {
	WordTypeCounts map[WordType]uint
	DataStatusCounts map[IoChannel]DataStatusCounts
	ConfigStatusCounts map[IoChannel]ConfigStatusCounts
	FifoFlags map[IoChannel]FifoFlags
}

func NewMonitor() *Monitor {
	return &Monitor{
		WordTypeCounts: make(map[WordType]uint),
		DataStatusCounts: make(map[IoChannel]DataStatusCounts),
		ConfigStatusCounts: make(map[IoChannel]ConfigStatusCounts),
		FifoFlags: make(map[IoChannel]FifoFlags),
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
	ioChannel := pacData.IoChannel

	var fifoFlags FifoFlags

	fifoFlags.LocalFifoFlags = pacData.Packet.LocalFifoFlags()
	fifoFlags.SharedFifoFlags = pacData.Packet.SharedFifoFlags()

	m.FifoFlags[ioChannel] = fifoFlags
	
}