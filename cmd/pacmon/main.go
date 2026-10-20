package main

import (
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
	cobra "github.com/spf13/cobra"

	_ "net/http/pprof"

	"github.com/felixge/fgprof"
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

func readPacmanIoJson() {
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

	if flags.PacmanIoJson != "" {
		readPacmanIoJson()
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

		go loop(
			flags.PacmanURL[iPacman],
			uint8(ioGroup),
			geometry,
			flags.PlotNorms,
			flags.DisabledListOptions,
			client,
			&wg,
		)
	}

	fmt.Println("hi this is pacmon")

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
			"http://np04-srv-017:32700",
			"InfluxDB URL")
	cmd.PersistentFlags().
		StringVar(&flags.InfluxOrg, "influx-org", "influxdata",
			"InfluxDB organization")
	cmd.PersistentFlags().
		StringVar(&flags.InfluxBucket, "influx-bucket", "pacmon",
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
