package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	zmq "github.com/pebbe/zmq4"
	cobra "github.com/spf13/cobra"
)

const blockSize = 10000

var gOptions struct {
	Port            uint16
	StartSecondsAgo int64
	SecondsPerFile  float64
}

var gCmd = cobra.Command{
	Use:   "pacreplay",
	Short: "Replay PACMAN binary-*.h5 files (in the current directory) over ZMQ",
	Run:   run,
}

func readTimestamp(msg []byte) uint32 {
	return binary.LittleEndian.Uint32(msg[1:5])
}

func replayFile(socket *zmq.Socket, fname string, tOffset *int64) error {
	r, err := OpenMsgs(fname)
	if err != nil {
		return err
	}
	defer r.Close()

	sent := 0
	startTime := time.Now()
	lastReport := startTime

	send := func(msg []byte) error {
		if len(msg) < 5 {
			return fmt.Errorf("%s: msg %d too short (%d bytes)",
				fname, sent, len(msg))
		}

		if gOptions.StartSecondsAgo != 0 && *tOffset == 0 {
			t0 := time.Now().Unix() - gOptions.StartSecondsAgo
			*tOffset = t0 - int64(readTimestamp(msg))
			if *tOffset <= 0 {
				return fmt.Errorf("computed non-positive time offset %d",
					*tOffset)
			}
		}

		if *tOffset != 0 {
			tNew := uint32(int64(readTimestamp(msg)) + *tOffset)
			binary.LittleEndian.PutUint32(msg[1:5], tNew)
		}

		if _, err := socket.SendBytes(msg, 0); err != nil {
			return err
		}

		sent++
		if time.Since(lastReport) > time.Second {
			fmt.Fprintf(os.Stderr, "\r  %d/%d", sent, r.Len)
			lastReport = time.Now()
		}
		return nil
	}

	for start := 0; start < r.Len; start += blockSize {
		count := min(blockSize, r.Len-start)
		if err := r.ReadBlock(start, count, send); err != nil {
			return err
		}

		// Pace so that the fraction of the file sent tracks the fraction of
		// seconds-per-file elapsed. Targeting the cumulative schedule (rather
		// than a fixed per-block sleep) absorbs variations in send/read time.
		if gOptions.SecondsPerFile > 0 {
			frac := float64(start+count) / float64(r.Len)
			target := startTime.Add(time.Duration(
				frac * gOptions.SecondsPerFile * float64(time.Second)))
			if wait := time.Until(target); wait > 0 {
				time.Sleep(wait)
			}
		}
	}

	elapsed := time.Now().Unix() - startTime.Unix()
	fmt.Fprintf(os.Stderr, "\r  %d/%d    (Elapsed: %d s)\n",
		sent, r.Len, elapsed)

	return nil
}

func run(cmd *cobra.Command, args []string) {
	socket, err := zmq.NewSocket(zmq.XPUB)
	if err != nil {
		log.Fatal(err)
	}
	// On exit, block until all queued messages are sent (like pyzmq's
	// Context.term())
	defer zmq.Term()
	defer socket.Close()

	if err := socket.SetLinger(-1); err != nil {
		log.Fatal(err)
	}
	if err := socket.SetXpubNodrop(true); err != nil {
		log.Fatal(err)
	}
	if err := socket.SetSndhwm(100000); err != nil {
		log.Fatal(err)
	}
	if err := socket.Bind(fmt.Sprintf("tcp://*:%d", gOptions.Port)); err != nil {
		log.Fatal(err)
	}

	files, err := filepath.Glob("binary-*.h5")
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(files)

	var tOffset int64
	for i, fname := range files {
		fmt.Printf("[%d/%d] %s\n", i+1, len(files), fname)
		if err := replayFile(socket, fname, &tOffset); err != nil {
			log.Fatal(err)
		}
	}
}

func main() {
	gCmd.PersistentFlags().Uint16VarP(&gOptions.Port, "port", "p", 6555,
		"port to serve from")

	gCmd.PersistentFlags().
		Int64VarP(&gOptions.StartSecondsAgo, "start-seconds-ago", "t", 0,
			"if set, add a constant offset to all UNIX timestamps such that "+
				"the first msg has a timestamp of start-seconds-ago before now")

	gCmd.PersistentFlags().
		Float64VarP(&gOptions.SecondsPerFile, "seconds-per-file", "s", 0,
			"if set, pause between blocks so that replaying each file takes "+
				"approximately this many seconds of wall time")

	if err := gCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
