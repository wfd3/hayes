package main

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"path"
)

var logger *log.Logger

func setupLogging() *log.Logger {
	var err error

	logflags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile

	if flags.syslog {
		logger, err := syslog.NewLogger(syslog.LOG_CRIT, logflags)
		if err != nil {
			fmt.Fprintf(os.Stderr,"Can't open syslog: %s\n", err)
			os.Exit(1)
		}
		return logger
	}

	logger := os.Stderr // default to StdErr
	if flags.logfile != "" {
		logger, err = os.OpenFile(flags.logfile,
			os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error, can't open logfile: %s\n",
				err)
			os.Exit(1)
		}
	}
	prefix := path.Base(os.Args[0]) + ": "
	return log.New(logger, prefix, logflags)

}
