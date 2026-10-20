package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type FastMonitor struct {
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

func NewFastMonitor(ioGroup uint8) *FastMonitor {
	return &FastMonitor{IoGroup: ioGroup}
}

func (m *FastMonitor) ProcessWord(word Word) {
	m.RecordType(word)
	m.RecordStatuses(word)
	m.RecordFifoFlags(word)
}

func (m *FastMonitor) RecordType(word Word) {
	newWordType := word.Type
	if word.Type == WordTypeData {
		packetType := word.PacData().Packet.Type()
		newWordType = PacketTypeMap[packetType]
	}
	m.WordTypeCounts[newWordType]++
}

func (m *FastMonitor) RecordStatuses(word Word) {
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

func (m *FastMonitor) RecordFifoFlags(word Word) {
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

func (m *FastMonitor) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {

	// point := makePoint("word_types_rates", timeNow)
	// for wordtype, count := range m.WordTypeCounts[:] {
	// 	point.AddField(WordType(wordtype).String(), float64(count)/timeDiff)
	// }
	// writeAPI.WritePoint(point)

	m.writeDataStatusesRates(writeAPI, timeNow, timeDiff)
	m.writeConfigStatusesRates(writeAPI, timeNow, timeDiff)
	// m.writeOtherStatusesRates(writeAPI, timeNow, timeDiff)
	m.writeDataStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	m.writeConfigStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	// m.writeOtherStatusesRatesPerChip(writeAPI, timeNow, timeDiff)
	// m.writeLocalFifoStatuses(writeAPI, timeNow)
	// m.writeSharedFifoStatuses(writeAPI, timeNow)
}

func (m *FastMonitor) writeDataStatusesRates(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.DataStatusCounts[:] {
		if !AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("data_statuses_rates", timeNow)
		addIoChannelTags(point, IoChannelKeyAt(m.IoGroup, ioChannel))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeConfigStatusesRates(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.ConfigStatusCounts[:] {
		if !AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("config_statuses_rates", timeNow)
		addIoChannelTags(point, IoChannelKeyAt(m.IoGroup, ioChannel))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeOtherStatusesRates(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for ioChannel, counts := range m.OtherStatusCounts[:] {
		if !AnyStatuses(
			m.DataStatusCounts[ioChannel],
			m.ConfigStatusCounts[ioChannel],
			m.OtherStatusCounts[ioChannel],
		) {
			continue
		}
		point := makePoint("other_statuses_rates", timeNow)
		addIoChannelTags(point, IoChannelKeyAt(m.IoGroup, ioChannel))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeDataStatusesRatesPerChip(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.DataStatusCountsPerChip[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("data_statuses_rates_per_chip", timeNow)
		addChipTags(point, ChipKeyAt(m.IoGroup, chip))
		addDataStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeConfigStatusesRatesPerChip(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.ConfigStatusCountsPerChip[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("config_statuses_rates_per_chip", timeNow)
		addChipTags(point, ChipKeyAt(m.IoGroup, chip))
		addConfigStatusFields(point, counts, timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeOtherStatusesRatesPerChip(
	writeAPI api.WriteAPI,
	timeNow time.Time,
	timeDiff float64,
) {
	for chip, counts := range m.OtherStatusCountsPerChip[:] {
		if !AnyStatuses(
			m.DataStatusCountsPerChip[chip],
			m.ConfigStatusCountsPerChip[chip],
			m.OtherStatusCountsPerChip[chip],
		) {
			continue
		}
		point := makePoint("other_statuses_rates_per_chip", timeNow)
		addChipTags(point, ChipKeyAt(m.IoGroup, chip))
		point.AddField("total", float64(counts)/timeDiff)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeLocalFifoStatuses(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, counts := range m.FifoFlagCounts[:] {
		total := float64(
			counts.LocalFifoLessHalfFull +
				counts.LocalFifoMoreHalfFull +
				counts.LocalFifoFull,
		)
		if total == 0 {
			continue
		}

		point := makePoint("local_fifo_statuses", timeNow)
		addChannelTags(point, ChannelKeyAt(m.IoGroup, channel))
		addFifoFields(
			point,
			counts.LocalFifoLessHalfFull,
			counts.LocalFifoMoreHalfFull,
			counts.LocalFifoFull,
			total,
		)
		writeAPI.WritePoint(point)
	}
}

func (m *FastMonitor) writeSharedFifoStatuses(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for channel, counts := range m.FifoFlagCounts[:] {
		total := float64(
			counts.SharedFifoLessHalfFull +
				counts.SharedFifoMoreHalfFull +
				counts.SharedFifoFull,
		)
		if total == 0 {
			continue
		}

		point := makePoint("shared_fifo_statuses", timeNow)
		addChipTags(
			point,
			ChipKeyAt(m.IoGroup, channel/MAX_CHANNELS_PER_CHIP),
		)
		addFifoFields(
			point,
			counts.SharedFifoLessHalfFull,
			counts.SharedFifoMoreHalfFull,
			counts.SharedFifoFull,
			total,
		)
		writeAPI.WritePoint(point)
	}
}
