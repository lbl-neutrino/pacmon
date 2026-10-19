package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type TrigMonitor struct {
	IoGroup []uint8
	Time    []uint32
}

func NewTrigMonitor() *TrigMonitor {
	return &TrigMonitor{}
}

func (tm *TrigMonitor) ProcessWord(word Word, ioGroup uint8) {
	tm.RecordTrig(word, ioGroup)
}

func (tm *TrigMonitor) RecordTrig(word Word, ioGroup uint8) {
	if word.Type == WordTypeTrig {
		tm.Time = append(tm.Time, word.PacTrig().Timestamp)
		tm.IoGroup = append(tm.IoGroup, ioGroup)
	}
}

func (tm *TrigMonitor) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	tm.writeTrigger(writeAPI, timeNow)
	writeAPI.Flush()
}

func (tm *TrigMonitor) writeTrigger(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for ind, t := range tm.Time {
		point := makePoint("trigger", timeNow)
		addTag(point, "io_group", tm.IoGroup[ind])
		point.AddField("trig", float64(t))
		writeAPI.WritePoint(point)
	}
}
