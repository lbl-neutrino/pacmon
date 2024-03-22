package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
)

type IoConfig struct {
	IoGroupPacmanURL [][]interface{} `json:"io_group"`
}

type Norms struct {
	Mean float64
	RMS  float64
	Rate float64
}

var PacmanURL []string
var PacmanIog []string
var PacmanIoJson string
var InfluxURL string
var InfluxOrg string
var InfluxBucket string
var GeometryFileMod013 string
var GeometryFileMod2 string
var UseSingleCube bool
var PlotNorms Norms

var cmd = cobra.Command{
	Use:   "pacmon",
	Short: "PACMAN monitor",
	Run:   run,
}

func runSingle(singlePacmanURL string, ioGroup uint8, geometry Geometry, plotNorms Norms, client influxdb2.Client, wg *sync.WaitGroup) {

	defer wg.Done()

	monitor := NewMonitor()
	monitor10s := NewMonitor10s()
	monitorPlots := NewMonitorPlots()
	syncMonitor := NewSyncMonitor()
	trigMonitor := NewTrigMonitor()

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	socket.SetSubscribe("")
	socket.Connect(singlePacmanURL)

	writeAPI := client.WriteAPI(InfluxOrg, InfluxBucket)
	now := time.Now()
	last := time.Now()
	now10s := time.Now()
	nowPlots := time.Now()
	last10s := time.Now()
	lastPlots := time.Now()

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
			monitorPlots.ProcessWord(word, ioGroup)

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

		if time.Since(last).Seconds() > 1 {
			now = time.Now()
			monitor.WriteToInflux(writeAPI, time.Unix(msgTime, 0), now.Sub(last).Seconds())
			monitor = NewMonitor() // Reset monitor
			last = now
		}

		if time.Since(last10s).Seconds() > 10 {
			now10s = time.Now()
			monitor10s.WriteToInflux(writeAPI, time.Unix(msgTime, 0), now10s.Sub(last10s).Seconds())
			monitor10s = NewMonitor10s() // Reset monitor
			last10s = now10s
		}

		if time.Since(lastPlots).Seconds() > 30 {
			nowPlots = time.Now()
			monitorPlots.PlotMetrics(geometry, ioGroup, plotNorms, nowPlots.Sub(lastPlots).Seconds())
			monitorPlots = NewMonitorPlots() // Reset monitor
			lastPlots = nowPlots
		}

	}
}

func run(cmd *cobra.Command, args []string) {

	token := os.Getenv("INFLUXDB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr,
			"Please set the INFLUXDB_TOKEN environment variable\n")
		os.Exit(1)
	}

	client := influxdb2.NewClientWithOptions(InfluxURL, token, influxdb2.DefaultOptions().SetPrecision(time.Millisecond))

	var wg sync.WaitGroup

	content, err := os.ReadFile(PacmanIoJson)
	if err == nil {
		fmt.Println("Reading IO config from JSON file: ", PacmanIoJson)
		var config IoConfig

		err = json.Unmarshal([]byte(content), &config)
		if err != nil {
			fmt.Println("JSON decode error:", err)
			return
		}

		PacmanURL = nil
		PacmanIog = nil

		fmt.Println("Found the following PACMANs vs. IO groups: ")
		for _, iog := range config.IoGroupPacmanURL {
			PacmanURL = append(PacmanURL, fmt.Sprintf("tcp://%s:5556", iog[1].(string)))
			PacmanIog = append(PacmanIog, strconv.Itoa(int(iog[0].(float64))))
			fmt.Println("\tURL: ", PacmanURL[len(PacmanURL)-1], " - io_group = ", PacmanIog[len(PacmanIog)-1])
		}

	} else {
		fmt.Println("Error when opening configuration file: ", err)
		fmt.Println("Using --pacman-url and --pacman-iog options")
	}

	geometryMod013 := LoadGeometry(GeometryFileMod013)
	geometryMod2 := LoadGeometry(GeometryFileMod2)

	if UseSingleCube {
		geometryMod013 = LoadGeometry("layout/geometry_singlecube.json")
	}

	wg.Add(len(PacmanURL))

	for iPacman := 0; iPacman < len(PacmanURL); iPacman++ {

		ioGroup, err := strconv.ParseUint(PacmanIog[iPacman], 10, 8)
		if err != nil {
			panic(err)
		}
		if ioGroup == 5 || ioGroup == 6 { // Module 2
			go runSingle(PacmanURL[iPacman], uint8(ioGroup), geometryMod2, PlotNorms, client, &wg)
		} else {
			go runSingle(PacmanURL[iPacman], uint8(ioGroup), geometryMod013, PlotNorms, client, &wg)
		}

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
	cmd.PersistentFlags().Float64VarP(&PlotNorms.Mean, "norm-mean", "m", 50., "Norm for the ADC mean plots")
	cmd.PersistentFlags().Float64VarP(&PlotNorms.RMS, "norm-rms", "s", 5., "Norm for the ADC RMS plots")
	cmd.PersistentFlags().Float64VarP(&PlotNorms.Rate, "norm-rate", "r", 10., "Norm for the rate plots")
	cmd.PersistentFlags().BoolVarP(&UseSingleCube, "single-cube", "c", false, "Use single-cube geometry")
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
