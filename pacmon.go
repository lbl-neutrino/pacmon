package main

import (
	// "bytes"
	// "encoding/binary"
	"fmt"
	"log"
	// "os"
	zmq "github.com/pebbe/zmq4/draft"
)


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
	socket.Connect("tcp://pacman32.local:5556")

	fmt.Println("CONNED")

	// poller := zmq.Poller{}
	// poller.Add(socket, zmq.POLLIN)

	for {
		// poller.Poll(10000)
		raw, err := socket.Recv(0)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("yo %d %x\n", len(raw), raw)

		// r := bytes.NewReader([]byte(raw))
		// var msg Msg
		// err = binary.Read(r, binary.LittleEndian, &msg.Header)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// fmt.Println(msg.Header)
		// fmt.Println(header.MsgTypeTag, header.Timestamp, header.NumWords)

		// for i := uint16(0); i < msg.Header.NumWords; i++ {
		// 	var word Word
		// 	binary.Read(r, binary.LittleEndian, &word)
		// 	msg.Words = append(msg.Words, word)
		// }

		// fmt.Println(msg)

		// token := os.Getenv("INFLUXDB_TOKEN")


	}
}
