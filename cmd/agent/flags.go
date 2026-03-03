package main

import (
	"flag"
	"time"
)

var flagAddressPort string
var flagReportInterval time.Duration
var flagPollInterval time.Duration

func parseFlags() {
	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.DurationVar(&flagReportInterval, "r", 10*time.Second, "interval between metrics sending")
	flag.DurationVar(&flagPollInterval, "p", 2*time.Second, "interval between metrics pooling")
	flag.Parse()
}
