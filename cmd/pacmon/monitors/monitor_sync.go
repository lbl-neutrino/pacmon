package monitors

import (
	"time"

	api "github.com/influxdata/influxdb-client-go/v2/api"

	. "larpix/pacmon/pkg"
)

type SyncMonitor struct {
	IoGroup []uint8
	Time    []uint32
	Type    []SyncType
}

func NewSyncMonitor() *SyncMonitor {
	return &SyncMonitor{}
}

func (sm *SyncMonitor) ProcessWord(word Word, ioGroup uint8) {
	sm.RecordSync(word, ioGroup)
}

func (sm *SyncMonitor) RecordSync(word Word, ioGroup uint8) {
	if word.Type == WordTypeSync {
		sm.Time = append(sm.Time, word.PacSync().Timestamp)
		sm.IoGroup = append(sm.IoGroup, ioGroup)
		sm.Type = append(sm.Type, word.PacSync().Type)
	}
}

func (sm *SyncMonitor) WriteToInflux(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	sm.writeSync(writeAPI, timeNow)
	writeAPI.Flush()
}

func (sm *SyncMonitor) writeSync(
	writeAPI api.WriteAPI,
	timeNow time.Time,
) {
	for ind, t := range sm.Time {
		point := makePoint("sync", timeNow)
		addTag(point, "io_group", sm.IoGroup[ind])

		if sm.Type[ind] == SyncTypeSync {
			point.AddField("sync", (float64(t)-1e7)*0.1)
		}
		if sm.Type[ind] == SyncTypeHeartbeat {
			point.AddField("heartbeat", float64(t))
		}
		if sm.Type[ind] == SyncTypeClkSource {
			point.AddField("clk_source", float64(t))
		}
		writeAPI.WritePoint(point)
	}
}
