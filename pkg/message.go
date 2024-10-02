package pkg

import (
	"bytes"
	"encoding/binary"
	"io"
	"strconv"
)

type MsgType byte

const (
	MsgTypeData MsgType = 'D'
	MsgTypeRequest MsgType = '?'
	MsgTypeReply MsgType = '!'
)

type WordType byte
type WordTypeLabel string

const (
	WordTypeData WordType = 'D'
	WordTypeTrig WordType = 'T'
	WordTypeSync WordType = 'S'
	WordTypePing WordType = 'P'
	WordTypeWrite WordType = 'W'
	WordTypeRead WordType = 'R'
	WordTypeError WordType = 'E'
)

type SyncType byte

const (
	SyncTypeSync SyncType = 'S'
	SyncTypeHeartbeat SyncType = 'H'
	SyncTypeClkSource SyncType = 'C'
)


func (wt WordType) String() string {
	switch wt {
	case WordTypeData: return "Data"
	case WordTypeTrig: return "Trig"
	case WordTypeSync: return "Sync"
	case WordTypePing: return "Ping"
	case WordTypeWrite: return "Write"
	case WordTypeRead: return "Read"
	case WordTypeError: return "Error"
	default: return strconv.Itoa(int(wt))
	}
}

var PacketTypeMap = map[PacketType]WordType {
	PacketTypeData: WordTypeData,
	PacketTypeError: WordTypeError,
	PacketTypeWrite: WordTypeWrite,
	PacketTypeRead: WordTypeRead,
}

type IoChannel uint8

// The Pac* structs are all 15 bytes
// (the Content of a Word)

type PacData struct {
	IoChannel IoChannel
	Timestamp uint32
	_ [2]byte
	Packet Packet
}

type PacTrig struct {
	Type uint8
	_ [2]byte
	Timestamp uint32
}

type PacSync struct {
	Type SyncType
	_ [2]byte
	Timestamp uint32
}

type PacPing struct {
	_ [15]byte
}

type PacWrite struct {
	_ [3]byte
	Write1 uint32
	_ [4]byte
	Write2 uint32
}

type PacRead struct {
	_ [3]byte
	Read1 uint32
	_ [4]byte
	Read2 uint32
}

type PacError struct {
	Err uint8
	_ [14]byte
}

type Word struct {				// [16]byte
	Type WordType				// byte
	Content [15]byte
}

func castWord[T any](w *Word) T {
	var ret T
	r := bytes.NewReader(w.Content[:])
	binary.Read(r, binary.LittleEndian, &ret)
	return ret
}

func packWord[T any](wordtype WordType, t T) Word {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, &t)
	var word Word
	word.Type = wordtype
	copy(word.Content[:], buf.Bytes())
	return word
	// return Word{
	// 	Type: wordtype,
	// 	Content: buf.Bytes(),
	// }
}

func (w *Word) PacData() PacData {
	return castWord[PacData](w)
}

func (t PacData) ToWord() Word {
	return packWord[PacData](WordTypeData, t)
}


func (w *Word) PacTrig() PacTrig {
	return castWord[PacTrig](w)
}

func (t PacTrig) ToWord() Word {
	return packWord[PacTrig](WordTypeTrig, t)
}


func (w *Word) PacSync() PacSync {
	return castWord[PacSync](w)
}

func (t PacSync) ToWord() Word {
	return packWord[PacSync](WordTypeSync, t)
}


func (w *Word) PacPing() PacPing {
	return castWord[PacPing](w)
}

func (t PacPing) ToWord() Word {
	return packWord[PacPing](WordTypePing, t)
}


func (w *Word) PacWrite() PacWrite {
	return castWord[PacWrite](w)
}

func (t PacWrite) ToWord() Word {
	return packWord[PacWrite](WordTypeWrite, t)
}


func (w *Word) PacRead() PacRead {
	return castWord[PacRead](w)
}

func (t PacRead) ToWord() Word {
	return packWord[PacRead](WordTypeRead, t)
}


func (w *Word) PacError() PacError {
	return castWord[PacError](w)
}

func (t PacError) ToWord() Word {
	return packWord[PacError](WordTypeError, t)
}

type MsgHeader struct {			// [8]byte
	Type MsgType				// byte
	Timestamp uint32
	_ byte
	NumWords uint16
}

type Msg struct {
	Header MsgHeader
	Words []Word
}

func (m *Msg) Read(r io.Reader) error {
	err := binary.Read(r, binary.LittleEndian, &m.Header)
	if err != nil {
		return err
	}

	for i := uint16(0); i < m.Header.NumWords; i++ {
		word := Word{}
		binary.Read(r, binary.LittleEndian, &word)
		m.Words = append(m.Words, word)
	}

	return nil
}

func (m *Msg) Write(w io.Writer) error {
	err := binary.Write(w, binary.LittleEndian, &m.Header)
	if err != nil {
		return err
	}

	for i := uint16(0); i < m.Header.NumWords; i++ {
		binary.Write(w, binary.LittleEndian, &m.Words[i])
	}

	return nil
}