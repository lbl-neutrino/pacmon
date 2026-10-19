package monitors

import (
	. "larpix/pacmon/pkg"
)

type Monitor struct {
	IoGroup uint8

	WordTypeCounts [NUM_WORD_TYPES]uint

	DataStatusCounts        [NUM_IO_CHANNELS]DataStatusCounts
	DataStatusCountsPerChip [NUM_CHIPS]DataStatusCounts

	ConfigStatusCounts        [NUM_IO_CHANNELS]ConfigStatusCounts
	ConfigStatusCountsPerChip [NUM_CHIPS]ConfigStatusCounts

	OtherStatusCounts        [NUM_IO_CHANNELS]uint
	OtherStatusCountsPerChip [NUM_CHIPS]uint

	FifoFlagCounts [NUM_CHANNELS]FifoFlagCounts
}

func NewMonitor(ioGroup uint8) *Monitor {
	return &Monitor{IoGroup: ioGroup}
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

	if ioChannel, ok := ioChannelIndex(pacData); ok {
		recordStatuses(
			m.DataStatusCounts[:],
			m.ConfigStatusCounts[:],
			m.OtherStatusCounts[:],
			ioChannel,
			pacData.Packet,
		)
	}
	if chip, ok := chipIndex(pacData); ok {
		recordStatuses(
			m.DataStatusCountsPerChip[:],
			m.ConfigStatusCountsPerChip[:],
			m.OtherStatusCountsPerChip[:],
			chip,
			pacData.Packet,
		)
	}
}

func (m *Monitor) RecordFifoFlags(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	if pacData.Packet.Type() != PacketTypeData { // Skip non-data
		return
	}

	packet := pacData.Packet
	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	fifoFlagCounts := &m.FifoFlagCounts[channel]

	// Local FIFOs
	switch FifoFlag(packet.LocalFifoFlags()) {
	case FifoLessHalfFull:
		fifoFlagCounts.LocalFifoLessHalfFull++
	case FifoMoreHalfFull:
		fifoFlagCounts.LocalFifoMoreHalfFull++
	case FifoFull1, FifoFull2:
		fifoFlagCounts.LocalFifoFull++
	}

	// Shared FIFOs
	switch FifoFlag(packet.SharedFifoFlags()) {
	case FifoLessHalfFull:
		fifoFlagCounts.SharedFifoLessHalfFull++
	case FifoMoreHalfFull:
		fifoFlagCounts.SharedFifoMoreHalfFull++
	case FifoFull1, FifoFull2:
		fifoFlagCounts.SharedFifoFull++
	}
}
