package main

import (
	"bufio"
	"context"
	"expvar"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Flags
var (
	outDir      string
	idListPath  string
	idFormat    int
	nRoutines   int
	useFastHTTP bool
	timeout     time.Duration
	reportI     time.Duration
)

var ctx context.Context

var idListWriter *bufio.Writer

var stats struct {
	reqs   *expvar.Int
	done   *expvar.Int
	failed *expvar.Int
}

const (
	IDBoth = iota
	ID5
	ID7
)

func init() {
	stats.reqs = expvar.NewInt("reqs")
	stats.failed = expvar.NewInt("failed")
	stats.done = expvar.NewInt("done")
}

func main() {
	flag.StringVar(&outDir, "out-dir", "./images", "Directory containing images")
	flag.StringVar(&idListPath, "id-list", "./ids.txt", "List with downloaded IDs")
	idFormatStr := flag.String("id-format", "both", "ID format to scrape (id5, id7, both)")
	flag.BoolVar(&useFastHTTP, "fasthttp", false, "Use fasthttp (HTTP/1.1) library instead of stdlib HTTP")
	flag.IntVar(&nRoutines, "routines", runtime.NumCPU(), "Number of instances to run in parallel")
	flag.DurationVar(&timeout, "timeout", 10*time.Second, "Request timeout")
	flag.DurationVar(&reportI, "report-interval", time.Second, "Report interval")
	monBind := flag.String("expvar-bind", ":6960", "Where to run expvar HTTP server (off to disable)")
	flag.Parse()

	if err := os.MkdirAll(outDir, 0777); err != nil {
		log.Fatalf("Failed to make out dir: %s", err)
	}

	switch strings.ToLower(*idFormatStr) {
	case "id5":
		idFormat = ID5
	case "id7":
		idFormat = ID7
	case "both":
		idFormat = IDBoth
	default:
		log.Fatal("Invalid ID format specified")
	}

	switch *monBind {
	case "off", "":
		break
	default:
		go func() {
			err := http.ListenAndServe(*monBind, expvar.Handler())
			if err != nil {
				log.Fatalf("Failed to bind monitoring: %s", err)
			}
		}()
	}

	if idListPath != "" {
		idListFile, err := os.OpenFile(idListPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
		if err != nil {
			log.Fatalf("Failed to open ID list: %s", err)
		}
		defer idListFile.Close()
		idListWriter = bufio.NewWriter(idListFile)
		defer idListWriter.Flush()
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		intC := make(chan os.Signal)
		signal.Notify(intC, os.Interrupt)
		<-intC
		log.Print("Shutting down")
		cancel()
	}()
	var wg sync.WaitGroup
	wg.Add(nRoutines)
	go reporter(reportI)
	for i := 0; i < nRoutines; i++ {
		go dumper(&wg)
	}
	wg.Wait()
}

func dumper(wg *sync.WaitGroup) {
	defer wg.Done()
	var req Requester
	if useFastHTTP {
		req = NewFastHTTPRequester()
	} else {
		req = NewVanillaRequester()
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			id := nextID()
			exists, err := dumpNext(req, id)
			if err != nil {
				stats.failed.Add(1)
				log.Printf("ERR Failed to dump %s: %s", id, err)
			}
			if exists && idListWriter != nil {
				stats.done.Add(1)
				_, _ = idListWriter.WriteString(id)
				_ = idListWriter.WriteByte('\n')
			}
		}
	}
}

func dumpNext(req Requester, id string) (bool, error) {
	// Check if exists
	exists, err := req.Exists(id)
	stats.reqs.Add(1)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}

	// Open file
	f, err := os.Create(filepath.Join(outDir, fmt.Sprintf("%s.jpg", id)))
	if err != nil {
		return true, err
	}
	defer f.Close()

	// Write to file
	err = req.StreamTo(id, f)
	stats.reqs.Add(1)
	return true, err
}

func reporter(interval time.Duration) {
	start := time.Now()
	var lastCount int64
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.Tick(interval):
			reqs := stats.reqs.Value()
			done := stats.done.Value()
			failed := stats.failed.Value()
			delta := done - lastCount
			perSecond := float64(delta) / interval.Seconds()
			average := float64(done) / time.Since(start).Seconds()
			log.Printf("%10d reqs |\t %10d done |\t%6d fail |\t%6.0f dl/s cur |\t%6.0f dl/s avg",
				reqs, done, failed, perSecond, average)
			lastCount = done
		}
	}
}

func nextID() string {
	switch idFormat {
	case ID5:
		return nextID5()
	case ID7:
		return nextID7()
	default:
		if rand.Intn(2) == 0 {
			return nextID5()
		} else {
			return nextID7()
		}
	}
}

var charList = []byte("abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
	"1234567890")

func nextID5() string {
	var chars [5]byte
	for i := range chars {
		chars[i] = charList[rand.Intn(len(charList))]
	}
	return string(chars[:])
}

func nextID7() string {
	var chars [7]byte
	for i := range chars {
		chars[i] = charList[rand.Intn(len(charList))]
	}
	return string(chars[:])
}

type Requester interface {
	Exists(id string) (bool, error)
	StreamTo(id string, w io.Writer) error
}
