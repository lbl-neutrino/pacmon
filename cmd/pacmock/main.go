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

func genMsg() Msg {
	chip := uint8(gRandom.Intn(160))
	channel := uint8(gRandom.Intn(64))
	io_channel := channel / 40
	timestamp := gRandom.Uint32()
	t_receipt := (timestamp + uint32(gRandom.Intn(50))) % 10000000
	adc := uint8(gRandom.Intn(256))

	var p Packet
	p.SetType(PacketTypeData)
	p.SetChip(chip)
	p.SetChannel(channel)
	p.SetDownstream(true)
	p.SetTimestamp(timestamp)
	p.SetData(adc)
	p.UpdateParity()

	word := PacData{
		IoChannel: IoChannel(io_channel),
		Timestamp: t_receipt,
		Packet:    p,
	}.ToWord()

	return Msg{
		Header: MsgHeader{
			Type:      MsgTypeData,
			Timestamp: uint32(time.Now().Unix()),
			NumWords:  3,
		},
		Words: []Word{word, word, word},
	}
}

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
		msg := genMsg()
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
