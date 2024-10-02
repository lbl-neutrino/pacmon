package main

import (
	"bytes"
	"log"
	"time"

	. "larpix/pacmon/pkg"

	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
)

var cmd = cobra.Command{
	Use: "pacmock",
	Short: "PACMAN mock data server",
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	socket.Bind("tcp://*:1234")

	var buf bytes.Buffer

	for {
		var p Packet
		p.SetType(PacketTypeData)
		p.SetChip(23)
		p.SetChannel(42)
		p.SetDownstream(true)
		p.SetTimestamp(987654)
		p.SetData(123)
		p.UpdateParity()

		word := PacData{
			IoChannel: 4,
			Timestamp: 987700,
			Packet: p,
		}.ToWord()

		msg := Msg{
			Header: MsgHeader{
				Type: MsgTypeData,
				Timestamp: 12345678,
				NumWords: 3,
			},
			Words: []Word{word, word, word},
		}

		msg.Write(&buf)
		socket.SendBytes(buf.Bytes(), 0)
		buf.Reset()
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}