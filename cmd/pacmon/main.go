package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"

	_ "net/http/pprof"

	"github.com/felixge/fgprof"

	"larpix/pacmon/cmd/pacmon/monitors"
	. "larpix/pacmon/pkg"
)

type IoConfig struct {
	IoGroupPacmanURL [][]interface{} `json:"io_group"`
}

type Norms struct {
	Mean float64
	RMS  float64
	Rate float64
	Freq float64
}

type DLOptions struct {
	RateThreshold float64
	Freq          float64
}

type Flags struct {
	PacmanURL           []string
	PacmanIog           []string
	PacmanIoJson        string
	InfluxURL           string
	InfluxOrg           string
	InfluxBucket        string
	GeometryFileMod0    string
	GeometryFileMod1    string
	GeometryFileMod2    string
	GeometryFileMod3    string
	UseSingleCube       bool
	PlotNorms           Norms
	DisabledListOptions DLOptions
	Profile             bool
}

var flags Flags

var cmd = cobra.Command{
	Use:   "pacmon",
	Short: "PACMAN monitor",
	Run:   run,
}

func runSingle(
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

		if len(syncMonitor.Time) > 0 {
			syncMonitor.WriteToInflux(
				writeAPI,
				time.Unix(msgTime, 0),
			)
			syncMonitor = monitors.NewSyncMonitor()
		}

		if len(trigMonitor.Time) > 0 {
			// trigMonitor.WriteToInflux(writeAPI,
			// 	time.Unix(msgTime, 0))
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

func run(cmd *cobra.Command, args []string) {
	if flags.Profile {
		http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())
		go func() {
			log.Println(http.ListenAndServe(":6363", nil))
		}()
	}

	token := os.Getenv("INFLUXDB_TOKEN")
	if token == "" {
		fmt.Fprintf(os.Stderr,
			"Please set the INFLUXDB_TOKEN environment variable\n")
		os.Exit(1)
	}

	client := influxdb2.NewClientWithOptions(
		flags.InfluxURL,
		token,
		influxdb2.DefaultOptions().SetPrecision(time.Millisecond),
	)

	var wg sync.WaitGroup

	content, err := os.ReadFile(flags.PacmanIoJson)
	if err == nil {
		fmt.Println(
			"Reading IO config from JSON file: ",
			flags.PacmanIoJson,
		)
		var config IoConfig

		err = json.Unmarshal([]byte(content), &config)
		if err != nil {
			fmt.Println("JSON decode error:", err)
			return
		}

		flags.PacmanURL = nil
		flags.PacmanIog = nil

		fmt.Println("Found the following PACMANs vs. IO groups: ")
		for _, iog := range config.IoGroupPacmanURL {
			url := iog[1].(string)
			if !strings.Contains(url, ":") {
				url = url + ":5556"
			}
			flags.PacmanURL = append(
				flags.PacmanURL,
				fmt.Sprintf("tcp://%s", url),
			)
			flags.PacmanIog = append(
				flags.PacmanIog,
				strconv.Itoa(int(iog[0].(float64))),
			)
			fmt.Println(
				"\tURL: ",
				flags.PacmanURL[len(flags.PacmanURL)-1],
				" - io_group = ",
				flags.PacmanIog[len(flags.PacmanIog)-1],
			)
		}

	} else {
		fmt.Println("Error when opening configuration file: ", err)
		fmt.Println("Using --pacman-url and --pacman-iog options")
	}

	geometryMod0 := LoadGeometry(flags.GeometryFileMod0)
	geometryMod1 := LoadGeometry(flags.GeometryFileMod1)
	geometryMod2 := LoadGeometry(flags.GeometryFileMod2)
	geometryMod3 := LoadGeometry(flags.GeometryFileMod3)

	if flags.UseSingleCube {
		geometryMod0 = LoadGeometry("layout/geometry_singlecube.json")
	}

	// A single --pacman-url applies to every IO group
	if len(flags.PacmanURL) == 1 && len(flags.PacmanIog) > 1 {
		url := flags.PacmanURL[0]
		flags.PacmanURL = make([]string, len(flags.PacmanIog))
		for i := range flags.PacmanURL {
			flags.PacmanURL[i] = url
		}
	}

	wg.Add(len(flags.PacmanURL))

	for iPacman := 0; iPacman < len(flags.PacmanURL); iPacman++ {

		ioGroup, err := strconv.ParseUint(
			flags.PacmanIog[iPacman],
			10,
			8,
		)
		if err != nil {
			panic(err)
		}
		if ioGroup == 1 || ioGroup == 2 { // Module 0
			go runSingle(
				flags.PacmanURL[iPacman],
				uint8(ioGroup),
				geometryMod0,
				flags.PlotNorms,
				flags.DisabledListOptions,
				client,
				&wg,
			)
		} else if ioGroup == 3 || ioGroup == 4 { // Module 1
			go runSingle(
				flags.PacmanURL[iPacman],
				uint8(ioGroup),
				geometryMod1,
				flags.PlotNorms,
				flags.DisabledListOptions,
				client,
				&wg,
			)
		} else if ioGroup == 5 || ioGroup == 6 { // Module 2
			go runSingle(
				flags.PacmanURL[iPacman],
				uint8(ioGroup),
				geometryMod2,
				flags.PlotNorms,
				flags.DisabledListOptions,
				client,
				&wg,
			)
		} else if ioGroup == 7 || ioGroup == 8 { // Module 3
			go runSingle(
				flags.PacmanURL[iPacman],
				uint8(ioGroup),
				geometryMod3,
				flags.PlotNorms,
				flags.DisabledListOptions,
				client,
				&wg,
			)
		} else { // Shouldn't get here
			fmt.Println("io_group not between 1 and 8.")
		}
	}

	wg.Wait()
}

func main() {
	cmd.PersistentFlags().
		StringSliceVar(&flags.PacmanURL, "pacman-url", nil,
			"Comma-separated list of PACMAN data server URLs "+
				"(a single URL applies to every IO group)")
	cmd.PersistentFlags().
		StringSliceVar(&flags.PacmanIog, "pacman-iog", nil,
			"Comma-separated list of corresponding IO groups")
	cmd.PersistentFlags().
		StringVar(&flags.InfluxURL, "influx-url",
			"http://localhost:18086",
			"InfluxDB URL")
	cmd.PersistentFlags().
		StringVar(&flags.InfluxOrg, "influx-org", "lbl-neutrino",
			"InfluxDB organization")
	cmd.PersistentFlags().
		StringVar(&flags.InfluxBucket, "influx-bucket", "pacmon-test",
			"InfluxDB bucket")
	cmd.PersistentFlags().
		StringVar(&flags.PacmanIoJson, "pacman-config", "",
			"JSON configuration file of the IO instead of "+
				"--pacman-url and --pacman-iog")
	cmd.PersistentFlags().
		StringVar(&flags.GeometryFileMod0, "geometry-mod0",
			"layout/geometry_mod0_v4.json",
			"JSON file with the layout of "+
				"Module 0 (io_group = 1,2)")
	cmd.PersistentFlags().
		StringVar(&flags.GeometryFileMod2, "geometry-mod2",
			"layout/geometry_mod2_v4.json",
			"JSON file with the layout of "+
				"Module 2 (io_group = 5,6)")
	cmd.PersistentFlags().
		StringVar(&flags.GeometryFileMod1, "geometry-mod1",
			"layout/geometry_mod1_v4.json",
			"JSON file with the layout of "+
				"Module 1 (io_group = 3,4)")
	cmd.PersistentFlags().
		StringVar(&flags.GeometryFileMod3, "geometry-mod3",
			"layout/geometry_mod3_v4.json",
			"JSON file with the layout of "+
				"Module 3 (io_group =  7,8)")
	cmd.PersistentFlags().
		Float64VarP(&flags.PlotNorms.Freq, "plot-freq", "f", 30.,
			"Frequency of updating plots in seconds")
	cmd.PersistentFlags().
		Float64VarP(&flags.PlotNorms.Mean, "norm-mean", "m", 50.,
			"Norm for the ADC mean plots")
	cmd.PersistentFlags().
		Float64VarP(&flags.PlotNorms.RMS, "norm-rms", "s", 5.,
			"Norm for the ADC RMS plots")
	cmd.PersistentFlags().
		Float64VarP(&flags.PlotNorms.Rate, "norm-rate", "r", 10.,
			"Norm for the rate plots")
	cmd.PersistentFlags().
		BoolVarP(&flags.UseSingleCube, "single-cube", "c", false,
			"Use single-cube geometry")
	cmd.PersistentFlags().
		Float64VarP(&flags.DisabledListOptions.Freq,
			"disable-list-freq", "d", 60.,
			"Frequency of updating the disable list in seconds")
	cmd.PersistentFlags().
		Float64VarP(&flags.DisabledListOptions.RateThreshold,
			"disable-list-threshold", "t", 5.,
			"Threshold of the data rate for the disable list in Hz")
	cmd.PersistentFlags().
		BoolVar(&flags.Profile, "profile", false, "Run fgprof profiler")

	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
