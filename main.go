package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"Revelio3/librevelio"
	"Revelio3/revelioprofile"
	"Revelio3/revelioscanner"
)

func errorWorker(r *librevelio.Revelio, wg *sync.WaitGroup) {
	defer wg.Done()
	for e := range r.Errors() {
		log.Printf("[!] %v", e)
	}
}

func resultWorker(r *librevelio.Revelio, wg *sync.WaitGroup) {
	defer wg.Done()
	for res := range r.Results() {
		if r.Opts.LairImport == true {
			r.ImportToLair(res.FileName)
		}
		if res.Message != "" {
			r.ClearProgress()
			fmt.Println(res.Message)
		}
		if r.Opts.Profile != true {
			outputfile := r.Opts.Directory + "livehosts.txt"
			sep := "\n"

			f, err := os.OpenFile(outputfile, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				log.Fatalf("[!] Unable to open livehost file for writing: %v", err)
			}
			defer f.Close()
			if _, err = f.WriteString(res.IP + sep); err != nil {
				log.Fatalf("[!] Error writing to livehosts file: %v", err)
			}
		}
	}
}

func progressWorker(c context.Context, r *librevelio.Revelio) {
	tick := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-tick.C:
			r.PrintProgress()
		case <-c.Done():
			return
		}
	}
}

func main() {
	o := librevelio.NewOptions()
	flag.IntVar(&o.Threads, "t", 1, "Number of concurrent threads")
	flag.StringVar(&o.LairAPI, "api", "", "Lair API String")
	flag.StringVar(&o.LairPID, "pid", "", "PID for Lair Project")
	flag.BoolVar(&o.LairImport, "l", false, "Import results to lair")
	flag.BoolVar(&o.Profile, "p", false, "Perform port profile scan on hosts")
	flag.StringVar(&o.Directory, "D", "./", "Output Directory for all files")
	flag.StringVar(&o.InputFile, "f", "", "Input file of new line seperated IP's (Format: 127.0.0.1, 127.0.0.0/24, 127.0.0.1-100)")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var plugin librevelio.RevelioPlugin

	if o.Profile == true {
		plugin = revelioprofile.RevelioProfile{}
	} else {
		plugin = revelioscanner.RevelioScanner{}
	}

	revelio, err := librevelio.NewRevelio(ctx, o, plugin)
	if err != nil {
		log.Fatalf("[!] %v", err)
	}

	if revelio.Opts.Profile != true {
		outputfile := revelio.Opts.Directory + "livehosts.txt"
		_, err := os.Create(outputfile)
		if err != nil {
			log.Fatalf("\n[!] Unable to create livehosts file: %v", err)
		}
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			fmt.Println("\n[!] Keyboard interrupt detected, terminating.")
			cancel()
		}
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go errorWorker(revelio, &wg)
	go resultWorker(revelio, &wg)
	go progressWorker(ctx, revelio)

	if err := revelio.Start(); err != nil {
		log.Printf("[!] %v", err)
	} else {
		cancel()
		wg.Wait()
	}

	revelio.ClearProgress()
	log.Println("Finished")
}
