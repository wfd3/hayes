package main

import (
	"flag"
)

const __ADDRESS_BOOK_FILE = "./phonebook.json"
const __ID_RSA_FILE = "./id_rsa"

var flags struct {
	syslog      bool
	logfile     string
	serialPort  string
	serialSpeed int
	phoneBook   string
	telnetPort  uint
	sshdPort    uint
	privateKey  string
	skipTelnet  bool
	skipSSH     bool
}

func initFlags() {
	flag.BoolVar(&flags.syslog, "syslog", false,
		"Log to syslog (default false)")

	flag.StringVar(&flags.logfile, "logfile", "",
		"Default log `file` (default stderr)")

	flag.StringVar(&flags.serialPort, "serial", "",
		"Serial `device` (eg, /dev/ttyS0)")

	flag.IntVar(&flags.serialSpeed, "speed", 115200,
		"Serial Port `speed` (bps) between DTE and DCE")

	flag.StringVar(&flags.phoneBook, "addressbook", __ADDRESS_BOOK_FILE,
		"Address Book `file`")

	flag.UintVar(&flags.telnetPort, "telnetport", 20000,
		"Network `port` number for inbound telnet sessions")

	flag.UintVar(&flags.sshdPort, "sshport", 22000,
		"Network `port` number for inbound sshd sessions")

	flag.StringVar(&flags.privateKey, "keyfile", __ID_RSA_FILE,
		"SSH Private Key `file`")

	flag.BoolVar(&flags.skipTelnet, "notelnet", false,
		"Don't start telnet server (default false)")

	flag.BoolVar(&flags.skipSSH, "nossh", false,
		"Don't start SSH server (default false)")

	flag.Parse()
}
