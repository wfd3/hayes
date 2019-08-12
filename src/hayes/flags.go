package main

import (
	"flag"
	"fmt"
	"os"
)

const (
	__ADDRESS_BOOK_FILE = "./addressbook.json"
	__ID_RSA_FILE       = "./id_rsa"
	__SERIAL_SPEED      = 115200
	__TELNET_PORT       = 20000
	__SSHD_PORT         = 22000
)

var flags struct {
	syslog      bool
	logfile     string
	serialPort  string
	serialSpeed int
	phoneBook   string
	telnetPort  uint
	sshdPort    uint
	privateKey  string
	telnet      bool
	ssh         bool
	sound       bool
	lcd         bool
}

func initFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage for %s: \n", os.Args[0])
		flag.PrintDefaults()
	}
	
	flag.BoolVar(&flags.syslog, "syslog", false,
		"Log to syslog (default false)")

	flag.StringVar(&flags.logfile, "logfile", "",
		"Default log `file` (default stderr)")

	flag.StringVar(&flags.serialPort, "serial", "",
		"Serial `device` (eg, /dev/ttyS0)")

	flag.IntVar(&flags.serialSpeed, "speed", __SERIAL_SPEED,
		"Serial Port `speed` (bps) between DTE and DCE")

	flag.StringVar(&flags.phoneBook, "addressbook", __ADDRESS_BOOK_FILE,
		"Address Book `file`")

	flag.UintVar(&flags.telnetPort, "telnetport", __TELNET_PORT,
		"Network `port` number for inbound telnet sessions")

	flag.UintVar(&flags.sshdPort, "sshport", __SSHD_PORT,
		"Network `port` number for inbound sshd sessions")

	flag.StringVar(&flags.privateKey, "keyfile", __ID_RSA_FILE,
		"SSH Private Key `file`")

	flag.BoolVar(&flags.telnet, "telnet", true,
		"Start telnet server (default true)")

	flag.BoolVar(&flags.ssh, "ssh", true,
		"Start SSH server (default true)")

	flag.BoolVar(&flags.sound, "sound", false,
		"Simulate sounds (default false)")

	flag.BoolVar(&flags.lcd, "lcd", false,
		"Use LCD (default false)")


	flag.Parse()
}
