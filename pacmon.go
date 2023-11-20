package main

import (
	"fmt"
	"log"
	zmq "github.com/pebbe/zmq4/draft"

)

type SockOpt struct {

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
		fmt.Printf("yo %d %x\n", len(raw), raw)
	}
}
