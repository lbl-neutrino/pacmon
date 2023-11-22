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

type Monitor struct {
	WordTypeCounts map[WordType]uint
	DataStatusCounts map[IoChannel]DataStatusCounts
	ConfigStatusCounts map[IoChannel]ConfigStatusCounts
}

func (m *Monitor) ProcessWord(word Word) {
	m.RecordType(word)
	m.RecordStatuses(word)
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
}
