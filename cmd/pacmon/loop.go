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

	fastMonitor := monitors.NewFastMonitor(ioGroup)
	slowMonitor := monitors.NewSlowMonitor(ioGroup)
	chanMonitor := monitors.NewChanMonitor(ioGroup)
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
	nowFast := time.Now()
	lastFast := time.Now()
	nowSlow := time.Now()
	lastSlow := time.Now()
	nowChan := time.Now()
	lastChan := time.Now()
	// nowPlots := time.Now()
	// nowDisabledList := time.Now()
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
			fastMonitor.ProcessWord(word)
			slowMonitor.ProcessWord(word)
			chanMonitor.ProcessWord(word)
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
			// trigMonitor.WriteToInflux(writeAPI,
			// 	time.Unix(msgTime, 0))
			trigMonitor = monitors.NewTrigMonitor()
		}

		if time.Since(lastFast).Seconds() > 1 {
			nowFast = time.Now()
			fastMonitor.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
				nowFast.Sub(lastFast).Seconds(),
			)
			fastMonitor = monitors.NewFastMonitor(ioGroup) // Reset monitor
			lastFast = nowFast
		}

		if time.Since(lastSlow).Seconds() > 10 {
			nowSlow = time.Now()
			slowMonitor.UpdateTopHotChannels() // Only sort once
			slowMonitor.UpdateTopChips()
			slowMonitor.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
				nowSlow.Sub(lastSlow).Seconds(),
			)
			slowMonitor = monitors.NewSlowMonitor(ioGroup) // Reset monitor
			lastSlow = nowSlow
		}

		if time.Since(lastChan).Seconds() > 600 {
			nowChan = time.Now()
			// Write a copy in the background so the loop isn't blocked
			chanCopy := *chanMonitor // Deep copy (struct holds only arrays)
			go chanCopy.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
				nowChan.Sub(lastChan).Seconds(),
			)
			chanMonitor = monitors.NewChanMonitor(ioGroup) // Reset monitor
			lastChan = nowChan
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
