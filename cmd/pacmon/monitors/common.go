package monitors

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
	IoChannel uint8
	ChipID    uint8
	ChannelID uint8
}

type ChipKey struct {
	IoGroup   uint8
	IoChannel uint8
	ChipID    uint8
}

type IoChannelKey struct {
	IoGroup   uint8
	IoChannel uint8
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
	FifoFull1        FifoFlag = 2
	FifoFull2        FifoFlag = 3
)

// The per-io-channel, per-chip and per-channel counters are stored in flat
// arrays rather than maps. Each monitor is bound to a single io group, so the
// io group is not part of the index. Io channels are numbered from 1. Words
// whose io channel or chip ID fall outside these bounds are not recorded in
// the corresponding arrays.
const (
	MAX_IO_CHANNELS          = 40
	MAX_CHIPS_PER_IO_CHANNEL = 160
	MAX_CHANNELS_PER_CHIP    = 64

	// Io channels index directly, so slot 0 is unused
	NUM_IO_CHANNELS = MAX_IO_CHANNELS + 1
	NUM_CHIPS       = NUM_IO_CHANNELS * MAX_CHIPS_PER_IO_CHANNEL
	NUM_CHANNELS    = NUM_CHIPS * MAX_CHANNELS_PER_CHIP

	NUM_WORD_TYPES = 256
)

// AnyStatuses reports whether any packet (data, config or other) was recorded
// for an index, i.e. whether it would have had an entry in the old maps
func AnyStatuses(data DataStatusCounts, config ConfigStatusCounts, other uint) bool {
	return data.Total+config.Total+other > 0
}

// Record increments the counters for a data packet
func (c *DataStatusCounts) Record(packet Packet) {
	c.Total++

	if packet.ValidParity() {
		c.ValidParity++
	} else {
		c.InvalidParity++
	}

	if packet.Downstream() {
		c.Downstream++
	} else {
		c.Upstream++
	}
}

// Record increments the counters for a config read/write packet
func (c *ConfigStatusCounts) Record(packet Packet) {
	c.Total++

	if !packet.ValidParity() {
		c.InvalidParity++
	}

	isRead := packet.Type() == PacketTypeRead
	switch {
	case packet.Downstream() && isRead:
		c.DownstreamRead++
	case packet.Downstream():
		c.DownstreamWrite++
	case isRead:
		c.UpstreamRead++
	default:
		c.UpstreamWrite++
	}
}

// recordStatuses classifies the packet as data, config or other and updates
// the corresponding counts at index i
func recordStatuses(
	dataCounts []DataStatusCounts,
	configCounts []ConfigStatusCounts,
	otherCounts []uint,
	i int,
	packet Packet,
) {
	switch packet.Type() {
	case PacketTypeRead, PacketTypeWrite:
		configCounts[i].Record(packet)
	case PacketTypeData:
		dataCounts[i].Record(packet)
	default:
		otherCounts[i]++
	}
}
