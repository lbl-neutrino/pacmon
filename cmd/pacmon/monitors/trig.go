package monitors

import (
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
