package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
)

var gOptions struct {
	PacmanURL []string
}

var gCmd = cobra.Command{
	Use:   "pacdump",
	Short: "Hex dump of the messages received from one or more PACMANs",
	Run:   run,
}

// Serializes output so that dumps from different PACMANs don't interleave
var gPrintMutex sync.Mutex

func dump(url string, wg *sync.WaitGroup) {
	defer wg.Done()

	socket, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal(err)
	}
	defer socket.Close()

	socket.SetSubscribe("")
	if err := socket.Connect(url); err != nil {
		log.Fatalf("%s: %v", url, err)
	}

	for {
		raw, err := socket.RecvBytes(0)
		if err != nil {
			log.Fatalf("%s: %v", url, err)
		}

		gPrintMutex.Lock()
		fmt.Printf("%s %s (%d bytes)\n",
			time.Now().Format(time.RFC3339Nano), url, len(raw))
		fmt.Print(hex.Dump(raw))
		fmt.Println()
		gPrintMutex.Unlock()
	}
}

func run(cmd *cobra.Command, args []string) {
	if len(gOptions.PacmanURL) == 0 {
		log.Fatal("No PACMAN URLs given (use --pacman-url)")
	}

	var wg sync.WaitGroup
	wg.Add(len(gOptions.PacmanURL))

	for _, url := range gOptions.PacmanURL {
		go dump(url, &wg)
	}

	wg.Wait()
}

func main() {
	gCmd.PersistentFlags().
		StringSliceVar(&gOptions.PacmanURL, "pacman-url", nil,
			"Comma-separated list of PACMAN data server URLs")

	if err := gCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
