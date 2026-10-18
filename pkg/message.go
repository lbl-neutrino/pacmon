package pkg

import (
	"encoding/binary"
	"io"
	"slices"
	"strconv"
)

type MsgType byte

const (
	MsgTypeData    MsgType = 'D'
	MsgTypeRequest MsgType = '?'
	MsgTypeReply   MsgType = '!'
)

type WordType byte
type WordTypeLabel string

const (
	WordTypeData  WordType = 'D'
	WordTypeTrig  WordType = 'T'
	WordTypeSync  WordType = 'S'
	WordTypePing  WordType = 'P'
	WordTypeWrite WordType = 'W'
	WordTypeRead  WordType = 'R'
	WordTypeError WordType = 'E'
)

type SyncType byte

const (
	SyncTypeSync      SyncType = 'S'
	SyncTypeHeartbeat SyncType = 'H'
	SyncTypeClkSource SyncType = 'C'
)

func (wt WordType) String() string {
	switch wt {
	case WordTypeData:
		return "Data"
	case WordTypeTrig:
		return "Trig"
	case WordTypeSync:
		return "Sync"
	case WordTypePing:
		return "Ping"
	case WordTypeWrite:
		return "Write"
	case WordTypeRead:
		return "Read"
	case WordTypeError:
		return "Error"
	default:
		return strconv.Itoa(int(wt))
	}
}

var PacketTypeMap = map[PacketType]WordType{
	PacketTypeData:  WordTypeData,
	PacketTypeError: WordTypeError,
	PacketTypeWrite: WordTypeWrite,
	PacketTypeRead:  WordTypeRead,
}

// The Pac* structs are all 15 bytes
// (the Content of a Word)

type PacData struct {
	IoChannel uint8
	Timestamp uint32
	_         [2]byte
	Packet    Packet
}

type PacTrig struct {
	Type      uint8
	_         [2]byte
	Timestamp uint32
}

type PacSync struct {
	Type      SyncType
	_         [2]byte
	Timestamp uint32
}

type PacPing struct {
	_ [15]byte
}

type PacWrite struct {
	_      [3]byte
	Write1 uint32
	_      [4]byte
	Write2 uint32
}

type PacRead struct {
	_     [3]byte
	Read1 uint32
	_     [4]byte
	Read2 uint32
}

type PacError struct {
	Err uint8
	_   [14]byte
}

type Word struct { // [16]byte
	Type    WordType // byte
	Content [15]byte
}

// The decoders and encoders below are written out by hand rather than using
// binary.Read/binary.Write, which go through reflection and allocate on every
// call. Padding bytes are ignored when decoding and zeroed when encoding.

func (w *Word) PacData() PacData {
	c := &w.Content
	var d PacData
	d.IoChannel = c[0]
	d.Timestamp = binary.LittleEndian.Uint32(c[1:5])
	copy(d.Packet[:], c[7:15])
	return d
}

func (t PacData) ToWord() Word {
	w := Word{Type: WordTypeData}
	c := &w.Content
	c[0] = t.IoChannel
	binary.LittleEndian.PutUint32(c[1:5], t.Timestamp)
	copy(c[7:15], t.Packet[:])
	return w
}

func (w *Word) PacTrig() PacTrig {
	c := &w.Content
	var d PacTrig
	d.Type = c[0]
	d.Timestamp = binary.LittleEndian.Uint32(c[3:7])
	return d
}

func (t PacTrig) ToWord() Word {
	w := Word{Type: WordTypeTrig}
	c := &w.Content
	c[0] = t.Type
	binary.LittleEndian.PutUint32(c[3:7], t.Timestamp)
	return w
}

func (w *Word) PacSync() PacSync {
	c := &w.Content
	var d PacSync
	d.Type = SyncType(c[0])
	d.Timestamp = binary.LittleEndian.Uint32(c[3:7])
	return d
}

func (t PacSync) ToWord() Word {
	w := Word{Type: WordTypeSync}
	c := &w.Content
	c[0] = byte(t.Type)
	binary.LittleEndian.PutUint32(c[3:7], t.Timestamp)
	return w
}

func (w *Word) PacPing() PacPing {
	return PacPing{}
}

func (t PacPing) ToWord() Word {
	return Word{Type: WordTypePing}
}

func (w *Word) PacWrite() PacWrite {
	c := &w.Content
	var d PacWrite
	d.Write1 = binary.LittleEndian.Uint32(c[3:7])
	d.Write2 = binary.LittleEndian.Uint32(c[11:15])
	return d
}

func (t PacWrite) ToWord() Word {
	w := Word{Type: WordTypeWrite}
	c := &w.Content
	binary.LittleEndian.PutUint32(c[3:7], t.Write1)
	binary.LittleEndian.PutUint32(c[11:15], t.Write2)
	return w
}

func (w *Word) PacRead() PacRead {
	c := &w.Content
	var d PacRead
	d.Read1 = binary.LittleEndian.Uint32(c[3:7])
	d.Read2 = binary.LittleEndian.Uint32(c[11:15])
	return d
}

func (t PacRead) ToWord() Word {
	w := Word{Type: WordTypeRead}
	c := &w.Content
	binary.LittleEndian.PutUint32(c[3:7], t.Read1)
	binary.LittleEndian.PutUint32(c[11:15], t.Read2)
	return w
}

func (w *Word) PacError() PacError {
	return PacError{Err: w.Content[0]}
}

func (t PacError) ToWord() Word {
	w := Word{Type: WordTypeError}
	w.Content[0] = t.Err
	return w
}

const (
	msgHeaderSize = 8
	wordSize      = 16
)

type MsgHeader struct { // [8]byte
	Type      MsgType // byte
	Timestamp uint32
	_         byte
	NumWords  uint16
}

type Msg struct {
	Header MsgHeader
	Words  []Word
}

func (m *Msg) Read(r io.Reader) error {
	var hdr [msgHeaderSize]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	m.Header.Type = MsgType(hdr[0])
	m.Header.Timestamp = binary.LittleEndian.Uint32(hdr[1:5])
	m.Header.NumWords = binary.LittleEndian.Uint16(hdr[6:8])

	n := int(m.Header.NumWords)
	buf := make([]byte, n*wordSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}

	m.Words = slices.Grow(m.Words, n)
	for i := range n {
		b := buf[i*wordSize : (i+1)*wordSize]
		var word Word
		word.Type = WordType(b[0])
		copy(word.Content[:], b[1:])
		m.Words = append(m.Words, word)
	}

	return nil
}

func (m *Msg) Write(w io.Writer) error {
	n := int(m.Header.NumWords)
	buf := make([]byte, msgHeaderSize+n*wordSize)
	buf[0] = byte(m.Header.Type)
	binary.LittleEndian.PutUint32(buf[1:5], m.Header.Timestamp)
	binary.LittleEndian.PutUint16(buf[6:8], m.Header.NumWords)

	for i := range n {
		b := buf[msgHeaderSize+i*wordSize : msgHeaderSize+(i+1)*wordSize]
		b[0] = byte(m.Words[i].Type)
		copy(b[1:], m.Words[i].Content[:])
	}

	_, err := w.Write(buf)
	return err
}
