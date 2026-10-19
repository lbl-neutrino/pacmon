package monitors

import (
	. "larpix/pacmon/pkg"
)

// ioChannelIndex returns the array index for the packet's io channel, and
// false if the io channel is out of bounds
func ioChannelIndex(pacData PacData) (int, bool) {
	ioChannel := int(pacData.IoChannel)
	return ioChannel, ioChannel >= 1 && ioChannel <= MAX_IO_CHANNELS
}

// chipIndex returns the array index for the packet's chip, and false if the
// io channel or chip ID is out of bounds
func chipIndex(pacData PacData) (int, bool) {
	ioChannel, ok := ioChannelIndex(pacData)
	chipID := int(pacData.Packet.Chip())
	ok = ok && chipID < MAX_CHIPS_PER_IO_CHANNEL
	return ioChannel*MAX_CHIPS_PER_IO_CHANNEL + chipID, ok
}

// channelIndex returns the array index for the packet's channel, and false if
// the io channel or chip ID is out of bounds
func channelIndex(pacData PacData) (int, bool) {
	chip, ok := chipIndex(pacData)
	channelID := int(pacData.Packet.Channel()) // 6 bits, always in bounds
	return chip*MAX_CHANNELS_PER_CHIP + channelID, ok
}

// The inverses of the above, for reporting

func IoChannelKeyAt(ioGroup uint8, ioChannel int) IoChannelKey {
	return IoChannelKey{IoGroup: ioGroup, IoChannel: uint8(ioChannel)}
}

func ChipKeyAt(ioGroup uint8, chip int) ChipKey {
	return ChipKey{
		IoGroup:   ioGroup,
		IoChannel: uint8(chip / MAX_CHIPS_PER_IO_CHANNEL),
		ChipID:    uint8(chip % MAX_CHIPS_PER_IO_CHANNEL),
	}
}

func ChannelKeyAt(ioGroup uint8, channel int) ChannelKey {
	chip := ChipKeyAt(ioGroup, channel/MAX_CHANNELS_PER_CHIP)
	return ChannelKey{
		IoGroup:   chip.IoGroup,
		IoChannel: chip.IoChannel,
		ChipID:    chip.ChipID,
		ChannelID: uint8(channel % MAX_CHANNELS_PER_CHIP),
	}
}
