package main

import (
	"bytes"
	"fmt"
	"log"
	zmq "github.com/pebbe/zmq4/draft"
)

type MsgType int

const (
	MsgTypeData MsgType = 'D'
	MsgTypeRequest MsgType = '?'
	MsgTypeReply MsgType = '!'
)

type WordType int

const (
	WordTypeData WordType = 'D'
	WordTypeTrig WordType = 'T'
	WordTypeSync WordType = 'S'
	WordTypePing WordType = 'P'
	WordTypeWrite WordType = 'W'
	WordTypeRead WordType = 'R'
	WordTypeError WordType = 'E'
)

type Word struct {

}

type Msg struct {
	msgType MsgType
	timestamp uint32
	numWords uint16
	words []Word
}

func (m *Msg) parse(raw []byte) *Msg {
	m.msgType = MsgTypeData
	m.timestamp = 1234
	m.numWords = 0
	m.words = []Word{}
	return m
}


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
	}
}
