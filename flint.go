package main

import (
	"flag"
	"fmt"
)

var latencyTableFile = flag.String("l", "", "latency config file")

func main() {
	var (
		t   *LatencyTable
		err error
	)

	flag.Parse()

	if *latencyTableFile != "" {
		t, err = NewLatencyTableFromFile(*latencyTableFile)
		if err != nil {
			fmt.Println(err)
			return
		}
	} else {
		t, err = NewLatencyTable()
		if err != nil {
			fmt.Println(err)
			fmt.Println("Try calling fint with latency config file via -l option")
			return
		}
	}

	RunUI(t)
}
