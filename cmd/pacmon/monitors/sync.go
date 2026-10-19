package monitors

import (
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
