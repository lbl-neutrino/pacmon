package main

type MsgType byte

const (
	MsgTypeData MsgType = 'D'
	MsgTypeRequest MsgType = '?'
	MsgTypeReply MsgType = '!'
)

type WordType byte

const (
	WordTypeData WordType = 'D'
	WordTypeTrig WordType = 'T'
	WordTypeSync WordType = 'S'
	WordTypePing WordType = 'P'
	WordTypeWrite WordType = 'W'
	WordTypeRead WordType = 'R'
	WordTypeError WordType = 'E'
)


type PacData struct {
	IoChannel byte
	Timestamp uint32
	_ [2]byte
	Packet Packet
}

type PacTrig struct {
	Type byte
	_ [2]byte
	Timestamp uint32
}

type PacSync struct {
	Type byte
	ClkSource byte
	_ [8]byte
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
	Err byte
	_ [14]byte
}


type Word struct {
	Type WordType
	Content [15]byte
}

type MsgHeader struct {
	Type MsgType
	Timestamp uint32
	_ byte
	NumWords uint16
}

type Msg struct {
	Header MsgHeader
	Words []Word
}
