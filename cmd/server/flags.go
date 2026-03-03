package main

import "flag"

var flagAdressPort string

func parseFlags() {
	flag.StringVar(&flagAdressPort, "a", "localhost:8080", "port to run server")
	flag.Parse()
}
