package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	zmq "github.com/pebbe/zmq4/draft"
    influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	api "github.com/influxdata/influxdb-client-go/v2/api"
	write "github.com/influxdata/influxdb-client-go/v2/api/write"

)

type DataStatusCounts struct {
	Total uint
	ValidParity uint
	InvalidParity uint
	Downstream uint
	Upstream uint
}

type ConfigStatusCounts struct {
	Total uint
	InvalidParity uint
	DownstreamRead uint
	DownstreamWrite uint
	UpstreamRead uint
	UpstreamWrite uint
}

type MonitorState struct {
	WordTypeCounts map[WordType]uint
	DataStatusCounts map[IoChannel]DataStatusCounts
	ConfigStatusCounts map[IoChannel]ConfigStatusCounts
}

func NewMonitorState() MonitorState {
	state := MonitorState{}
	// for wordtype, _label := range WordTypeLabels {
	// 	state.WordTypeCounts[wordtype] = 0
	// }
	return state
}

func (m *MonitorState) RecordType(word Word) {
}

func (m *MonitorState) RecordStatuses(word Word) {
}

func (s *MonitorState) WriteToInflux(writeAPI api.WriteAPIBlocking) {
	tile_id := 1
	tags := map[string]string{"tile_id": strconv.Itoa(tile_id)}

	makePoint := func (name string) *write.Point {
		return influxdb2.NewPoint(name, tags, nil, time.Now())
	}

	point := makePoint("word_types")
	for wordtype, count := range s.WordTypeCounts {
		point.AddField(wordtype.String(), count)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range s.DataStatusCounts {
		point = makePoint("data_statuses")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("Total", counts.Total)
		point.AddField("ValidParity", counts.ValidParity)
		point.AddField("InvalidParity", counts.InvalidParity)
		point.AddField("Downstream", counts.Downstream)
		point.AddField("Upstream", counts.Upstream)
	}
	writeAPI.WritePoint(context.Background(), point)

	for ioChannel, counts := range s.ConfigStatusCounts {
		point = makePoint("config_statuses")
		point.AddTag("io_channel", strconv.Itoa(int(ioChannel)))
		point.AddField("Total", counts.Total)
		point.AddField("InvalidParity", counts.InvalidParity)
		point.AddField("DownstreamRead", counts.DownstreamRead)
		point.AddField("DownstreamWrite", counts.DownstreamWrite)
		point.AddField("UpstreamRead", counts.UpstreamRead)
		point.AddField("UpstreamWrite", counts.UpstreamWrite)
	}
	writeAPI.WritePoint(context.Background(), point)
}

func main() {
	fmt.Println("sdf")
	state := NewMonitorState()

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

	org := "lbl-neutrino"
	bucket := "pacmon-go"
	token := os.Getenv("INFLUXDB_TOKEN")
	url := "http://localhost:18086"
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

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
			state.RecordType(word)
			state.RecordStatuses(word)
		}

		if time.Now().Sub(last).Seconds() > 1 {
			state.WriteToInflux(writeAPI)
			last = time.Now()
		}

	}
}
