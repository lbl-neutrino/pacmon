package monitors

import (
	. "larpix/pacmon/pkg"
)

type DisabledListMonitor struct {
	IoGroup uint8

	DataStatusCountsPerChannel [NUM_CHANNELS]DataStatusCounts

	TopHotChannels []ChannelKey
	TopHotValues   []uint
}

func NewDisabledListMonitor(ioGroup uint8) *DisabledListMonitor {
	return &DisabledListMonitor{IoGroup: ioGroup}
}

func (dlm *DisabledListMonitor) ProcessWord(word Word) {
	dlm.RecordStatuses(word)
}

func (dlm *DisabledListMonitor) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	if pacData.Packet.Type() != PacketTypeData { // Skip non-data
		return
	}

	if channel, ok := channelIndex(pacData); ok {
		dlm.DataStatusCountsPerChannel[channel].Record(pacData.Packet)
	}
}
