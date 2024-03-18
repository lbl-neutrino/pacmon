package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"
	"sync"
	"io/ioutil"
	"strconv"
    "encoding/json"

	cobra "github.com/spf13/cobra"
	zmq "github.com/pebbe/zmq4"
)

var PacmanURL []string
var PacmanIog []string
var PacmanIoJson string
var InfluxURL string
var InfluxOrg string
var InfluxBucket string
var GeometryFileMod013 string
var GeometryFileMod2 string

var cmd = cobra.Command{
	Use: "pacmon",
	Short: "PACMAN monitor",
	Run: run,
}


type IoConfig struct {
	IoGroupPacmanURL [][]interface{} `json:"io_group"`
}

func runSingle(singlePacmanURL string, ioGroup uint8, geometry Geometry, wg *sync.WaitGroup){

	defer wg.Done()

	monitor := NewMonitor()
	monitor10s := NewMonitor10s()
	monitor1min := NewMonitor1min()
	syncMonitor := NewSyncMonitor()
	trigMonitor := NewTrigMonitor()

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	socket.SetSubscribe("")
	socket.Connect(singlePacmanURL)

	writeAPI := getWriteAPI(InfluxURL, InfluxOrg, InfluxBucket)
	// now := time.Now()
	// last := time.Now()
	now10s := time.Now()
	now1min := time.Now()
	last10s := time.Now()
	last1min := time.Now()

	prevMsgTime := time.Now().Unix()

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
			monitor.ProcessWord(word, ioGroup)
			monitor10s.ProcessWord(word, ioGroup)
			monitor1min.ProcessWord(word, ioGroup)
			
			syncMonitor.ProcessWord(word, ioGroup)
			trigMonitor.ProcessWord(word, ioGroup)
		}

		if len(syncMonitor.Time) > 0 {
			syncMonitor.WriteToInflux(writeAPI, time.Unix(msgTime, 0))
			syncMonitor = NewSyncMonitor()
		}

		if len(trigMonitor.Time) > 0 {
			trigMonitor.WriteToInflux(writeAPI, time.Unix(msgTime, 0))
			trigMonitor = NewTrigMonitor()
		}

		// if time.Now().Sub(last).Seconds() > 1 {
		if (msgTime - prevMsgTime) > 1 {
			// now = time.Now()
			fmt.Println(time.Now(), ": here 1s")
			// monitor.WriteToInflux(writeAPI, now, now.Sub(last).Seconds())
			monitor.WriteToInflux(writeAPI, time.Unix(msgTime, 0), float64(msgTime - prevMsgTime))
			monitor = NewMonitor() // Reset monitor
			fmt.Println(time.Now(), ": here 1s - after writing to influx")
			// last = now
			prevMsgTime = msgTime
		}

		if time.Now().Sub(last10s).Seconds() > 10 {
			now10s = time.Now()
			fmt.Println(time.Now(), ": here 10s")

			monitor10s.WriteToInflux(writeAPI, now10s, now10s.Sub(last10s).Seconds())
			monitor10s = NewMonitor10s() // Reset monitor
			fmt.Println(time.Now(), ": here 10s - after writing to influx")
			last10s = now10s
		}

		if time.Now().Sub(last1min).Seconds() > 5 {
			now1min = time.Now()
			fmt.Println(time.Now(), ": here 1m")
			monitor1min.PlotMean(geometry)
			monitor1min = NewMonitor1min() // Reset monitor
			fmt.Println(time.Now(), ": here 1m - after writing to influx")
			last1min = now1min
		}

	}
}

func run(cmd *cobra.Command, args []string) {

	var wg sync.WaitGroup

    content, err := ioutil.ReadFile(PacmanIoJson)
	if err == nil {
        fmt.Println("Reading IO config from JSON file...")
		var config IoConfig

		err = json.Unmarshal([]byte(content), &config)
		if err != nil {
			fmt.Println("JSON decode error:", err)
			return
		}

		PacmanURL = nil
		PacmanIog = nil
		for _, iog := range config.IoGroupPacmanURL {
			PacmanURL = append(PacmanURL, fmt.Sprintf("tcp://%s:5556", iog[1].(string)))
			PacmanIog = append(PacmanIog, strconv.Itoa(int(iog[0].(float64))))
		}

		fmt.Println("Read URLs: ", PacmanURL)
		fmt.Println("Corresponding to IO groups: ", PacmanIog)

	} else {
		fmt.Println("Error when opening configuration file: ", err)
		fmt.Println("Using --pacman-url and --pacman-iog options")
    }

	geometryMod013 := LoadGeometry(GeometryFileMod013) 
	// geometryMod2 := LoadGeometry(GeometryFileMod2)

	wg.Add(len(PacmanURL))

	for iPacman := 0; iPacman < len(PacmanURL); iPacman++ {
		
		ioGroup, err := strconv.ParseUint(PacmanIog[iPacman], 10, 8)
		if err != nil {
			panic(err)
		}
		
		go runSingle(PacmanURL[iPacman], uint8(ioGroup), geometryMod013, &wg)
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
	cmd.PersistentFlags().StringVar(&PacmanIoJson, "pacman-config", "",
		"JSON configuration file of the IO instead of --pacman-url and --pacman-iog")
	cmd.PersistentFlags().StringVar(&GeometryFileMod013, "geometry-mod013", "layout/geometry_mod013.json",
		"JSON file with the layout of Modules 0, 1 and 3 (io_group = 1,2,3,4,7,8)")
	cmd.PersistentFlags().StringVar(&GeometryFileMod2, "geometry-mod2", "layout/geometry_mod2.json",
		"JSON file with the layout of Module 2 (io_group = 5,6)")

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
