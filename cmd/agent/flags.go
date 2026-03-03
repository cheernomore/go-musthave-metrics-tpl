package main

import (
	"flag"
)

var flagAddressPort string
var flagReportInterval int
var flagPollInterval int

func parseFlags() {
	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.IntVar(&flagReportInterval, "r", 10, "interval between metrics sending")
	flag.IntVar(&flagPollInterval, "p", 2, "interval between metrics pooling")
	flag.Parse()
}
