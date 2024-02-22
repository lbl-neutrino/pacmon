package main

import (
	"bytes"
	// "fmt"
	"log"
	"os"
	"time"
	"sync"
	"strconv"

	cobra "github.com/spf13/cobra"
	zmq "github.com/pebbe/zmq4"
)

var PacmanURL []string
var PacmanIog []string
var InfluxURL string
var InfluxOrg string
var InfluxBucket string

var cmd = cobra.Command{
	Use: "pacmon",
	Short: "PACMAN monitor",
	Run: run,
}

func run_single(singlePacmanURL string, ioGroup uint8, wg *sync.WaitGroup){

	defer wg.Done()

	monitor := NewMonitor()
	monitor10s := NewMonitor10s()

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	socket.SetSubscribe("")
	socket.Connect(singlePacmanURL)

	writeAPI := getWriteAPI(InfluxURL, InfluxOrg, InfluxBucket)
	last := time.Now()
	last10s := time.Now()

	for {

		raw, err := socket.Recv(0)
		if err != nil {
			log.Fatal(err)
		}

		r := bytes.NewReader([]byte(raw))
		msg := Msg{}
		err = msg.Read(r)
		if err != nil {
			log.Fatal(err)
		}

		for _, word := range msg.Words {
			monitor.ProcessWord(word, ioGroup)
			monitor10s.ProcessWord(word, ioGroup)
		}

		if time.Now().Sub(last).Seconds() > 1 {
			monitor.WriteToInflux(writeAPI, time.Now().Sub(last).Seconds())
			monitor = NewMonitor() // Reset monitor
			last = time.Now()
		}

		if time.Now().Sub(last10s).Seconds() > 10 {
			monitor10s.WriteToInflux(writeAPI, time.Now().Sub(last10s).Seconds())
			monitor10s = NewMonitor10s() // Reset monitor
			last10s = time.Now()
		}
	}
}

func run(cmd *cobra.Command, args []string) {

	var wg sync.WaitGroup

	wg.Add(len(PacmanURL))

	for iPacman := 0; iPacman < len(PacmanURL); iPacman++ {
		
		ioGroup, err := strconv.ParseUint(PacmanIog[iPacman], 10, 8)
		if err != nil {
			panic(err)
		}
		
		go run_single(PacmanURL[iPacman], uint8(ioGroup), &wg)
	}

	wg.Wait()
}

func main() {
	cmd.PersistentFlags().StringSliceVar(&PacmanURL, "pacman-url", nil,
		"Comma-separated list of PACMAN data server URLs")
	cmd.PersistentFlags().StringSliceVar(&PacmanIog, "pacman-iog", nil,
		"Comma-separated list of corresponding IO groups")
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
