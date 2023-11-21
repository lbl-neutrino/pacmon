package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	// "os"
	zmq "github.com/pebbe/zmq4/draft"
)

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
	Packet [8]byte
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
	Type byte
	Content [15]byte
}

type MsgHeader struct {
	MsgTypeTag MsgType
	Timestamp uint32
	_ byte
	NumWords uint16
}

type Msg struct {
	Header MsgHeader
	Words []Word
}

// func (m *Msg) parse(r *bytes.Reader) *Msg {
	// m.msgType = MsgTypeData
	// m.timestamp = 1234
	// m.numWords = 0
	// m.words = []Word{}
// 	return m
// }


func main() {
	fmt.Println("Hello, World!")
	// ctx := zmq.Context{}

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	// socket.SetLinger(1000)
	// socket.SetRcvtimeo(1000 * 11)
	// socket.SetSndtimeo(1000 * 11)
	socket.SetSubscribe("")
	socket.Connect("tcp://localhost:5556")

	fmt.Println("CONNED")

	// poller := zmq.Poller{}
	// poller.Add(socket, zmq.POLLIN)

	for {
		// poller.Poll(10000)
		raw, err := socket.Recv(0)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Printf("yo %d %x\n", len(raw), raw)

		r := bytes.NewReader([]byte(raw))
		var msg Msg
		err = binary.Read(r, binary.LittleEndian, &msg.Header)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Println(msg.Header)
		// fmt.Println(header.MsgTypeTag, header.Timestamp, header.NumWords)

		for i := uint16(0); i < msg.Header.NumWords; i++ {
			var word Word
			binary.Read(r, binary.LittleEndian, &word)
			msg.Words = append(msg.Words, word)
		}

		fmt.Println(msg)

		// token := os.Getenv("INFLUXDB_TOKEN")


	}
}
