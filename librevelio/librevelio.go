package librevelio

import (
	"bufio"
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/lair-framework/go-nmap"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type SetupFunc func(*Revelio) error

type ProcessFunc func(*Revelio, string) ([]Result, error)

type Result struct {
	FileName string
	Message  string
	IP		 string
	Status   int
}

type Revelio struct {
	Opts            *Options
	context         context.Context
	mu              *sync.RWMutex
	expectedIpCount int
	ipScanned       int
	plugin          RevelioPlugin
	resultChan      chan Result
	errorChan       chan error
}

type RevelioPlugin interface {
	Process(*Revelio, string, *sync.WaitGroup) ([]Result, error)
}

type Options struct {
	LairPID 	string
	LairAPI		string
	LairImport	bool
	InputFile	string
	Directory	string
	Profile 	bool
	Threads		int
}

func NewOptions() *Options {
	return &Options{}
}

func (opt *Options) validate() *multierror.Error {
	var errorList *multierror.Error

	if opt.InputFile == "" {
		errorList = multierror.Append(errorList, fmt.Errorf("input File (-f): Must be specified"))
	}

	if opt.Threads < 0 {
		errorList = multierror.Append(errorList, fmt.Errorf("threads (-t): Invalid value: %d", opt.Threads))
	}

	if opt.LairImport == true {
		if opt.LairPID == "" {
			errorList = multierror.Append(errorList, fmt.Errorf("lair PID (-PID): Must be specified to import into lair"))
		}
		if opt.LairAPI == "" {
			errorList = multierror.Append(errorList, fmt.Errorf("lair API URL (-api): Must be specified to import into lair"))
		}
	}

	if opt.Directory != "" {
		if _, err := os.Stat(opt.Directory); os.IsNotExist(err) {
			errorList = multierror.Append(errorList, fmt.Errorf("please create the %s directory", opt.Directory))
		}
	}
	return errorList
}

func NewRevelio(c context.Context, opts *Options, plugin RevelioPlugin) (*Revelio, error) {
	multiError := opts.validate()
	if multiError != nil {
		return nil, multiError
	}

	var r Revelio
	r.plugin = plugin
	r.context = c
	r.Opts = opts
	r.mu = new(sync.RWMutex)
	r.resultChan = make(chan Result)
	r.errorChan = make(chan error)

	return &r, nil
}

// Returns a channel of errors
func (r *Revelio) Errors() <-chan error {
	return r.errorChan
}

// Returns a channel of files
func (r *Revelio) Results() <-chan Result {
	return r.resultChan
}

func (r *Revelio) PrintProgress() {
	r.mu.RLock()
	if r.Opts.InputFile == "-" {
		fmt.Fprintf(os.Stderr, "\rScanning: %d", r.ipScanned)
		// only print status if we already read in the wordlist
	} else if r.expectedIpCount > 0 {
		fmt.Fprintf(os.Stderr, "\rScanning: %d / %d (%3.2f%%)", r.ipScanned, r.expectedIpCount, float32(r.ipScanned)*100.0/float32(r.expectedIpCount))
	}
	r.mu.RUnlock()
}

func (r *Revelio) ClearProgress() {
	fmt.Fprint(os.Stderr, resetTerminal())
}

func resetTerminal() string {
	return "\r\x1b[2K"
}

// Increments to next IP
func (r *Revelio) incrementRequest() {
	r.mu.Lock()
	r.ipScanned++
	r.mu.Unlock()
}

func (r *Revelio) worker(ipChan <-chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-r.context.Done():
			return
		case ip, ok := <-ipChan:
			if !ok {
				return
			}
			r.incrementRequest()
			res, err := r.plugin.Process(r, ip, wg)
			if err != nil {
				r.errorChan <- err
				continue
			} else {
				for _, res := range res {
					r.resultChan <- res
				}
			}
		}
	}
}

func (r *Revelio) getIpList() (*bufio.Scanner, error) {
	if r.Opts.InputFile == "-" {
		return bufio.NewScanner(os.Stdin), nil
	}

	iplist, err := os.Open(r.Opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file: %v", err)
	}

	lines, err := lineCounter(iplist)
	if err != nil {
		return nil, fmt.Errorf("failed to get number of lines in input file: %v", err)
	}

	r.expectedIpCount = lines
	r.ipScanned = 0

	_, err = iplist.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to rewind wordlist: %v", err)
	}

	return bufio.NewScanner(iplist), nil
}

func (r *Revelio) Start() error {
	var wg sync.WaitGroup
	wg.Add(r.Opts.Threads)

	ipChan := make(chan string, r.Opts.Threads)

	for i := 0; i< r.Opts.Threads; i++ {
		go r.worker(ipChan, &wg)
	}
	scanner, err := r.getIpList()
	if err != nil {
		return err
	}

Scan:
	for scanner.Scan() {
		select {
		case <-r.context.Done():
			break Scan
		default:
			ip := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(ip, "#") && len(ip) > 0 {
				ipChan <- ip
			}
		}
	}
	close(ipChan)
	wg.Wait()
	close(r.resultChan)
	close(r.errorChan)
	return nil
}

func (r *Revelio) NmapParse(filename string) (*nmap.NmapRun, error){
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("could not open file for parsing: %v", err)
	}
	nmapRun, err := nmap.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("error parsing nmap from file: %v", err)
	}
	return nmapRun, nil
}

func (r *Revelio) ImportToLair(filename string) error{
	os.Setenv("LAIR_API_SERVER", r.Opts.LairAPI)
	args := []string{"-k", r.Opts.LairPID, filename}
	command := "/usr/local/bin/drone-nmap"
	err := commandWorker(command, args)
	if err != nil {
		return fmt.Errorf("error importing into lair: %v", err)
	}
	return nil
}