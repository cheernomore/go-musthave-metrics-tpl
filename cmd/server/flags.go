package main

import "flag"

var flagPort string

func parseFlags() {
	flag.StringVar(&flagPort, "a", ":8080", "port to run server")
	flag.Parse()
}
