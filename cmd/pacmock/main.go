package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"time"

	. "larpix/pacmon/pkg"

	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
)

var gOptions struct {
	Port uint16
	Seed int64
}

var gCmd = cobra.Command{
	Use:   "pacmock",
	Short: "PACMAN mock data server",
	Run:   run,
}

var gRandom *rand.Rand

func run(cmd *cobra.Command, args []string) {
	gRandom = rand.New(rand.NewSource(gOptions.Seed))

	socket, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	socket.Bind(fmt.Sprintf("tcp://*:%d", gOptions.Port))

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
			Packet:    p,
		}.ToWord()

		msg := Msg{
			Header: MsgHeader{
				Type:      MsgTypeData,
				Timestamp: 12345678,
				NumWords:  3,
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
	gCmd.PersistentFlags().Uint16VarP(&gOptions.Port, "port", "p", 6555,
		"port to serve from")

	gCmd.PersistentFlags().Int64VarP(&gOptions.Seed, "seed", "s", 0,
		"random seed")

	if err := gCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
