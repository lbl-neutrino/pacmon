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
	PacmanGeom          []string
	PacmanIoJson        string
	InfluxURL           string
	InfluxOrg           string
	InfluxBucket        string
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

// A per-PACMAN flag may be given either once per IO group or as a single value
// that applies to every IO group. An empty flag is returned as-is.
func broadcastFlag(vals []string, n int, name string) []string {
	if len(vals) == 0 || len(vals) == n {
		return vals
	}
	if len(vals) == 1 {
		result := make([]string, n)
		for i := range result {
			result[i] = vals[0]
		}
		return result
	}
	log.Fatalf("%s has %d values but there are %d IO groups", name, len(vals), n)
	return nil
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

	if len(flags.PacmanURL) == 0 {
		log.Fatal("No PACMAN URLs given (use --pacman-url or --pacman-config)")
	}
	flags.PacmanURL = broadcastFlag(flags.PacmanURL, len(flags.PacmanIog), "--pacman-url")
	flags.PacmanGeom = broadcastFlag(flags.PacmanGeom, len(flags.PacmanIog), "--pacman-geom")

	// Load each distinct geometry file only once
	geometries := make(map[string]Geometry)
	for _, path := range flags.PacmanGeom {
		if _, ok := geometries[path]; !ok {
			geometries[path] = LoadGeometry(path)
		}
	}

	wg.Add(len(flags.PacmanIog))

	for iPacman := 0; iPacman < len(flags.PacmanIog); iPacman++ {

		ioGroup, err := strconv.ParseUint(
			flags.PacmanIog[iPacman],
			10,
			8,
		)
		if err != nil {
			panic(err)
		}

		var geometry Geometry
		if flags.PacmanGeom != nil {
			geometry = geometries[flags.PacmanGeom[iPacman]]
		}

		go runSingle(
			flags.PacmanURL[iPacman],
			uint8(ioGroup),
			geometry,
			flags.PlotNorms,
			flags.DisabledListOptions,
			client,
			&wg,
		)
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
		StringSliceVar(&flags.PacmanGeom, "pacman-geom", nil,
			"Comma-separated list of corresponding geometry JSON files "+
				"(a single file applies to every IO group)")
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
