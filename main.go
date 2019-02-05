package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

func usage() {
	log.Println("fatal: missing csv file")
	fmt.Println(`
usage: pinoy [options] <file>

options:
  -v, --verbose		output verbose messages
	
args:
  file			file containing comma separated values of destinations and ip addresses
	
example: 
  pinoy -v airline-ips.csv`)
}

func output(s string, verbose bool) {
	if verbose {
		log.Println(s)
	}
}

func ping(v []string, ch chan []string) {
	out, _ := exec.Command("ping", v[2]).Output()
	v[3] = string(out)
	ch <- v
}

func updatePingResults(results *[][]string, ch <-chan []string, wg *sync.WaitGroup, verbose bool) {
	for {
		v := <-ch
		output("finished pinging "+v[1]+" at "+v[2]+" and result was:\n"+v[3]+"\n", verbose)
		id, _ := strconv.Atoi(v[0])
		result := (*results)[id]
		result[3] = v[3]
		(*results)[id] = result
		wg.Done()
	}
}

func tracert(v []string, ch chan []string) {
	out, _ := exec.Command("tracert", v[2]).Output()
	v[4] = string(out)
	ch <- v
}

func updateTracertResults(results *[][]string, ch <-chan []string, wg *sync.WaitGroup, verbose bool) {
	for {
		v := <-ch
		output("finished sending tracert "+v[1]+" at "+v[2]+" and result was:\n"+v[3]+"\n", verbose)
		id, _ := strconv.Atoi(v[0])
		result := (*results)[id]
		result[4] = v[4]
		(*results)[id] = result
		wg.Done()
	}
}

func writeResults(data [][]string) {
	f, err := os.OpenFile("results.csv", os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		log.Fatal("fatal: cannot output file", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.WriteAll(data); err != nil {
		log.Fatal("fatal: cannot write to file", err)
	}
}

func main() {
	var help bool
	var verbose bool
	flag.BoolVar(&help, "h", false, "print help")
	flag.BoolVar(&help, "help", false, "print help")
	flag.BoolVar(&verbose, "v", false, "output verbose messages")
	flag.BoolVar(&verbose, "verbose", false, "output verbose messages")
	flag.Parse()

	if help {
		usage()
		return
	}

	if len(flag.Args()) == 0 {
		usage()
		return
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("fatal: %s\n", err)
	}

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	max := len(records)
	log.Printf("found %d records", max)

	pings := make(chan []string, max)
	defer close(pings)

	tracerts := make(chan []string, max)
	defer close(tracerts)

	results := make([][]string, max)
	var wg sync.WaitGroup
	wg.Add(max * 2)
	for id, elem := range records {
		results[id] = make([]string, 5)
		results[id][0] = strconv.Itoa(id)
		results[id][1] = elem[0]
		results[id][2] = elem[1]
		go func(id int, v []string) {
			output("pinging ["+v[1]+" "+v[2]+"]", verbose)
			ping(v, pings)
		}(id, results[id])

		go func(id int, v []string) {
			output("sending tracert ["+v[0]+" "+v[1]+"]", verbose)
			tracert(v, tracerts)
		}(id, results[id])

		go func() {
			updatePingResults(&results, pings, &wg, verbose)
		}()

		go func() {
			updateTracertResults(&results, tracerts, &wg, verbose)
		}()
	}

	wg.Wait()

	writeResults(results)
}
