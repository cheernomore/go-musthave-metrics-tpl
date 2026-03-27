package main

import "flag"

var flagAddressPort string

func parseFlags() {
	flag.StringVar(&flagAddressPort, "a", "localhost:8080", "port to run server")
	flag.Parse()
}
