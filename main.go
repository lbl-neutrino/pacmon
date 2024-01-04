package main

import (
	"bytes"
	// "fmt"
	"log"
	"os"
	"time"

	cobra "github.com/spf13/cobra"
	zmq "github.com/pebbe/zmq4/draft"
)

var PacmanURL string
var InfluxURL string
var InfluxOrg string
var InfluxBucket string

var cmd = cobra.Command{
	Use: "pacmon",
	Short: "PACMAN monitor",
	Run: run,
}

func run(cmd *cobra.Command, args []string) {
	monitor := NewMonitor()
	monitor10s := NewMonitor10s()

	// ctx := zmq.Context{}

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	// socket.SetLinger(1000)
	// socket.SetRcvtimeo(1000 * 11)
	// socket.SetSndtimeo(1000 * 11)
	socket.SetSubscribe("")
	socket.Connect(PacmanURL)

	// poller := zmq.Poller{}
	// poller.Add(socket, zmq.POLLIN)

	writeAPI := getWriteAPI(InfluxURL, InfluxOrg, InfluxBucket)
	last := time.Now()
	last10s := time.Now()

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
			monitor.ProcessWord(word)
			monitor10s.ProcessWord(word)
		}

		if time.Now().Sub(last).Seconds() > 1 {
			monitor.WriteToInflux(writeAPI, time.Now().Sub(last).Seconds())
			monitor = NewMonitor() // Reset monitor
			last = time.Now()
		}

		if time.Now().Sub(last10s).Seconds() > 10 {
			monitor10s.WriteToInflux(writeAPI, time.Now().Sub(last).Seconds())
			monitor10s = NewMonitor10s() // Reset monitor
			last10s = time.Now()
		}
	}
}

func main() {
	cmd.PersistentFlags().StringVar(&PacmanURL, "pacman-url", "tcp://pacman32.local:5556",
		"PACMAN data server URL")
	cmd.PersistentFlags().StringVar(&InfluxURL, "influx-url", "http://localhost:18086",
		"InfluxDB URL")
	cmd.PersistentFlags().StringVar(&InfluxOrg, "influx-org", "lbl-neutrino",
		"InfluxDB organization")
	cmd.PersistentFlags().StringVar(&InfluxBucket, "influx-bucket", "pacmon-test",
		"InfluxDB bucket")

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
