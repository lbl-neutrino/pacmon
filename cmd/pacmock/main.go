package main

import (
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
	//. "larpix/pacmon/pkg"
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

	//var buf bytes.Buffer

	for {
		// msg := Msg{}
		// msg.Write(&buf)
		// socket.SendBytes(buf.Bytes(), 0)
		socket.Send("hello\n", 0)
		// buf.Reset()
		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}