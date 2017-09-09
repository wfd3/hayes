package main

import (
	"hayes"
	"flag"
	"os"
	"log"
	"io"
)

var _flags_logfile     = flag.String("l", "", "Default logfile")

func main() {
	var m hayes.Modem
	var logger io.Writer
	var err error

	flag.Parse()

	// TODO: should this be here or in the main
	logger = os.Stdout
	if *_flags_logfile != "" {
		logger, err = os.OpenFile(*_flags_logfile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			panic("Can't open logfile")
		}
	}
	log := log.New(logger, "modem: ",
		log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	// TODO: end

	m.PowerOn(log)
}
