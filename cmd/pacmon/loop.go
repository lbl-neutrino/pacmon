package main

import (
	"bytes"
	"log"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	zmq "github.com/pebbe/zmq4"

	"larpix/pacmon/cmd/pacmon/monitors"
	. "larpix/pacmon/pkg"
)

func loop(
	singlePacmanURL string,
	ioGroup uint8,
	geometry Geometry,
	plotNorms Norms,
	disabledListOptions DLOptions,
	client influxdb2.Client,
	wg *sync.WaitGroup,
) {

	defer wg.Done()

	monitor := monitors.NewMonitor(ioGroup)
	monitor10s := monitors.NewMonitor10s(ioGroup)
	// monitorPlots := monitors.NewMonitorPlots(ioGroup)
	disabledListMonitor := monitors.NewDisabledListMonitor(ioGroup)
	syncMonitor := monitors.NewSyncMonitor()
	trigMonitor := monitors.NewTrigMonitor()

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	socket.SetSubscribe("")
	socket.Connect(singlePacmanURL)

	writeAPI := client.WriteAPI(flags.InfluxOrg, flags.InfluxBucket)
	now := time.Now()
	last := time.Now()
	now10s := time.Now()
	// nowPlots := time.Now()
	// nowDisabledList := time.Now()
	last10s := time.Now()
	// lastPlots := time.Now()
	// lastDisabledList := time.Now()

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

		msgTime := int64(msg.Header.Timestamp)

		for _, word := range msg.Words {
			monitor.ProcessWord(word)
			monitor10s.ProcessWord(word)
			// monitorPlots.ProcessWord(word)
			disabledListMonitor.ProcessWord(word)

			syncMonitor.ProcessWord(word, ioGroup)
			trigMonitor.ProcessWord(word, ioGroup)
		}

		if len(syncMonitor.Time) > 60 {
			syncMonitor.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
			)
			syncMonitor = monitors.NewSyncMonitor()
		}

		if len(trigMonitor.Time) > 60 {
			trigMonitor.WriteToInflux(writeAPI,
				time.Unix(msgTime, 0))
			trigMonitor = monitors.NewTrigMonitor()
		}

		if time.Since(last).Seconds() > 60 {
			now = time.Now()
			monitor.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
				now.Sub(last).Seconds(),
			)
			monitor = monitors.NewMonitor(ioGroup) // Reset monitor
			last = now
		}

		if time.Since(last10s).Seconds() > 300 {
			now10s = time.Now()
			monitor10s.UpdateTopHotChannels() // Only sort once
			monitor10s.UpdateTopChips()
			monitor10s.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
				now10s.Sub(last10s).Seconds(),
			)
			monitor10s = monitors.NewMonitor10s(ioGroup) // Reset monitor
			last10s = now10s
		}

		// if time.Since(lastPlots).Seconds() > plotNorms.Freq {
		// 	nowPlots = time.Now()
		// 	plotMetrics(monitorPlots, geometry, ioGroup,
		// 		plotNorms, nowPlots.Sub(lastPlots).Seconds())
		// 	monitorPlots = monitors.NewMonitorPlots(ioGroup) // Reset monitor
		// 	lastPlots = nowPlots
		// }

		// if time.Since(lastDisabledList).
		// 	Seconds() >
		// 	disabledListOptions.Freq {
		// 	nowDisabledList = time.Now()
		// 	writeDisabledList(
		// 		disabledListMonitor,
		// 		disabledListOptions.RateThreshold,
		// 		time.Since(lastDisabledList).Seconds(),
		// 		ioGroup,
		// 	)
		// 	// Reset monitor
		// 	disabledListMonitor = monitors.NewDisabledListMonitor(ioGroup)
		// 	lastDisabledList = nowDisabledList
		// }

	}
}
