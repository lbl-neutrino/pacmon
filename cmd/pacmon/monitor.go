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

type Monitor10s struct {
	IoGroup uint8

	ADCMeanTotal float64
	ADCRMSTotal  float64

	ADCMeanPerChip [NUM_CHIPS]float64
	ADCRMSPerChip  [NUM_CHIPS]float64

	ADCMeanPerChannel [NUM_CHANNELS]float64
	ADCRMSPerChannel  [NUM_CHANNELS]float64

	NPacketsTotal      uint32
	NPacketsPerChip    [NUM_CHIPS]uint32
	NPacketsPerChannel [NUM_CHANNELS]uint32

	DataStatusCountsPerChannel   [NUM_CHANNELS]DataStatusCounts
	ConfigStatusCountsPerChannel [NUM_CHANNELS]ConfigStatusCounts
	OtherStatusCountsPerChannel  [NUM_CHANNELS]uint

	TopHotChannels []ChannelKey
	TopHotValues   []uint

	BottomColdChannels []ChannelKey
	BottomColdValues   []uint

	TopADCMeanChannels []ChannelKey
	TopADCMeanValues   []float64

	TopADCRMSChannels []ChannelKey
	TopADCRMSValues   []float64

	InvalidParityPerChip   [NUM_CHIPS]uint32
	TopInvalidParityChips  []ChipKey
	TopInvalidParityCounts []uint32

	SharedFifoFullPerChip   [NUM_CHIPS]uint32
	TopSharedFifoFullChips  []ChipKey
	TopSharedFifoFullCounts []uint32
}

type MonitorPlots struct {
	IoGroup uint8

	ADCMeanPerChannel          [NUM_CHANNELS]float64
	ADCRMSPerChannel           [NUM_CHANNELS]float64
	NPacketsPerChannel         [NUM_CHANNELS]uint32
	DataStatusCountsPerChannel [NUM_CHANNELS]DataStatusCounts
}

type DisabledListMonitor struct {
	IoGroup uint8

	DataStatusCountsPerChannel [NUM_CHANNELS]DataStatusCounts

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

func NewMonitor(ioGroup uint8) *Monitor {
	return &Monitor{IoGroup: ioGroup}
}

func NewMonitor10s(ioGroup uint8) *Monitor10s {
	return &Monitor10s{IoGroup: ioGroup}
}

func NewMonitorPlots(ioGroup uint8) *MonitorPlots {
	return &MonitorPlots{IoGroup: ioGroup}
}

func NewDisabledListMonitor(ioGroup uint8) *DisabledListMonitor {
	return &DisabledListMonitor{IoGroup: ioGroup}
}

func NewSyncMonitor() *SyncMonitor {
	return &SyncMonitor{}
}

func NewTrigMonitor() *TrigMonitor {
	return &TrigMonitor{}
}

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

func ioChannelKeyAt(ioGroup uint8, ioChannel int) IoChannelKey {
	return IoChannelKey{IoGroup: ioGroup, IoChannel: uint8(ioChannel)}
}

func chipKeyAt(ioGroup uint8, chip int) ChipKey {
	return ChipKey{
		IoGroup:   ioGroup,
		IoChannel: uint8(chip / MAX_CHIPS_PER_IO_CHANNEL),
		ChipID:    uint8(chip % MAX_CHIPS_PER_IO_CHANNEL),
	}
}

func channelKeyAt(ioGroup uint8, channel int) ChannelKey {
	chip := chipKeyAt(ioGroup, channel/MAX_CHANNELS_PER_CHIP)
	return ChannelKey{
		IoGroup:   chip.IoGroup,
		IoChannel: chip.IoChannel,
		ChipID:    chip.ChipID,
		ChannelID: uint8(channel % MAX_CHANNELS_PER_CHIP),
	}
}

// anyStatuses reports whether any packet (data, config or other) was recorded
// for an index, i.e. whether it would have had an entry in the old maps
func anyStatuses(data DataStatusCounts, config ConfigStatusCounts, other uint) bool {
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

func (m *Monitor) ProcessWord(word Word) {
	m.RecordType(word)
	m.RecordStatuses(word)
	m.RecordFifoFlags(word)
}

func (m10s *Monitor10s) ProcessWord(word Word) {
	m10s.RecordStatuses(word)
	m10s.RecordADC(word)
	m10s.RecordChipStats(word)
}

func (mPlots *MonitorPlots) ProcessWord(word Word) {
	// mPlots.RecordStatuses(word)
	mPlots.RecordADC(word)
}

func (sm *SyncMonitor) ProcessWord(word Word, ioGroup uint8) {
	sm.RecordSync(word, ioGroup)
}

func (dlm *DisabledListMonitor) ProcessWord(word Word) {
	dlm.RecordStatuses(word)
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

func (m *Monitor10s) RecordChipStats(word Word) {
	pacData := word.PacData()
	packet := pacData.Packet
	chip, ok := chipIndex(pacData)
	if !ok {
		return
	}

	isInvalid := !packet.ValidParity()
	flag := FifoFlag(packet.SharedFifoFlags())
	isSharedFifoFull := flag == FifoFull1 || flag == FifoFull2

	if isInvalid {
		m.InvalidParityPerChip[chip]++
	}

	if isSharedFifoFull {
		m.SharedFifoFullPerChip[chip]++
	}
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

func (m10s *Monitor10s) RecordStatuses(word Word) {
	if word.Type != WordTypeData {
		return
	}

	pacData := word.PacData()
	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	recordStatuses(
		m10s.DataStatusCountsPerChannel[:],
		m10s.ConfigStatusCountsPerChannel[:],
		m10s.OtherStatusCountsPerChannel[:],
		channel,
		pacData.Packet,
	)
}

func (m10s *Monitor10s) RecordADC(word Word) {
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
	adc := float64(packet.Data())

	m10s.ADCMeanTotal, m10s.ADCRMSTotal = UpdateMeanRMS(
		m10s.ADCMeanTotal,
		m10s.ADCRMSTotal,
		m10s.NPacketsTotal,
		adc,
	)
	m10s.NPacketsTotal++

	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}
	chip := channel / MAX_CHANNELS_PER_CHIP

	m10s.ADCMeanPerChip[chip], m10s.ADCRMSPerChip[chip] = UpdateMeanRMS(
		m10s.ADCMeanPerChip[chip],
		m10s.ADCRMSPerChip[chip],
		m10s.NPacketsPerChip[chip],
		adc,
	)
	m10s.NPacketsPerChip[chip]++

	m10s.ADCMeanPerChannel[channel], m10s.ADCRMSPerChannel[channel] =
		UpdateMeanRMS(
			m10s.ADCMeanPerChannel[channel],
			m10s.ADCRMSPerChannel[channel],
			m10s.NPacketsPerChannel[channel],
			adc,
		)
	m10s.NPacketsPerChannel[channel]++

}

func (m10s *Monitor10s) UpdateTopHotChannels() {
	ioGroup := m10s.IoGroup

	// A channel is reported if any packet was recorded for it
	totals := make([]uint, NUM_CHANNELS)
	for i, counts := range m10s.DataStatusCountsPerChannel[:] {
		totals[i] = counts.Total
	}
	hasStatuses := func(i int) bool {
		return anyStatuses(
			m10s.DataStatusCountsPerChannel[i],
			m10s.ConfigStatusCountsPerChannel[i],
			m10s.OtherStatusCountsPerChannel[i],
		)
	}
	hasADC := func(i int) bool { return m10s.NPacketsPerChannel[i] > 0 }

	m10s.TopHotChannels, m10s.TopHotValues = sortByValue(
		totals, hasStatuses, 100, false, ioGroup, channelKeyAt,
	)
	m10s.BottomColdChannels, m10s.BottomColdValues = sortByValue(
		totals, hasStatuses, 100, true, ioGroup, channelKeyAt,
	)
	m10s.TopADCMeanChannels, m10s.TopADCMeanValues = sortByValue(
		m10s.ADCMeanPerChannel[:], hasADC, 100, false, ioGroup, channelKeyAt,
	)
	m10s.TopADCRMSChannels, m10s.TopADCRMSValues = sortByValue(
		m10s.ADCRMSPerChannel[:], hasADC, 100, false, ioGroup, channelKeyAt,
	)
}

func (m10s *Monitor10s) UpdateTopChips() {
	ioGroup := m10s.IoGroup
	invalidParity := m10s.InvalidParityPerChip[:]
	sharedFifoFull := m10s.SharedFifoFullPerChip[:]

	m10s.TopInvalidParityChips, m10s.TopInvalidParityCounts = sortByValue(
		invalidParity,
		func(i int) bool { return invalidParity[i] > 0 },
		100,
		false,
		ioGroup,
		chipKeyAt,
	)
	m10s.TopSharedFifoFullChips, m10s.TopSharedFifoFullCounts = sortByValue(
		sharedFifoFull,
		func(i int) bool { return sharedFifoFull[i] > 0 },
		100,
		false,
		ioGroup,
		chipKeyAt,
	)
}

func (mPlots *MonitorPlots) RecordADC(word Word) {
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
	channel, ok := channelIndex(pacData)
	if !ok {
		return
	}

	adc := float64(packet.Data())

	mPlots.ADCMeanPerChannel[channel], mPlots.ADCRMSPerChannel[channel] =
		UpdateMeanRMS(
			mPlots.ADCMeanPerChannel[channel],
			mPlots.ADCRMSPerChannel[channel],
			mPlots.NPacketsPerChannel[channel],
			adc,
		)
	mPlots.NPacketsPerChannel[channel]++

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
