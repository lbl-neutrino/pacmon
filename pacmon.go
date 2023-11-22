package main

import (
	"bytes"
	// "fmt"
	"log"
	"time"

	zmq "github.com/pebbe/zmq4/draft"
)


func main() {
	var state Monitor

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

	// poller := zmq.Poller{}
	// poller.Add(socket, zmq.POLLIN)

	writeAPI := getWriteAPI()
	last := time.Now()

	for {
		// poller.Poll(10000)
		raw, err := socket.Recv(0)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Printf("yo %d %x\n", len(raw), raw)

		r := bytes.NewReader([]byte(raw))
		msg := Msg{}
		err = msg.Read(r)
		if err != nil {
			log.Fatal(err)
		}

		// fmt.Println(msg)
		// fmt.Println(msg.Header)
		// fmt.Println(header.MsgTypeTag, header.Timestamp, header.NumWords)

		for _, word := range msg.Words {
			state.ProcessWord(word)
		}

		if time.Now().Sub(last).Seconds() > 1 {
			state.WriteToInflux(writeAPI)
			last = time.Now()
		}
	}
}
