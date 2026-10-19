package monitors

import (
	. "larpix/pacmon/pkg"
)

type MonitorPlots struct {
	IoGroup uint8

	ADCMeanPerChannel          [NUM_CHANNELS]float64
	ADCRMSPerChannel           [NUM_CHANNELS]float64
	NPacketsPerChannel         [NUM_CHANNELS]uint32
	DataStatusCountsPerChannel [NUM_CHANNELS]DataStatusCounts
}

func NewMonitorPlots(ioGroup uint8) *MonitorPlots {
	return &MonitorPlots{IoGroup: ioGroup}
}

func (mPlots *MonitorPlots) ProcessWord(word Word) {
	// mPlots.RecordStatuses(word)
	mPlots.RecordADC(word)
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
