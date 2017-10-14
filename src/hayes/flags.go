package hayes

import (
	"flag"
)


const __ADDRESS_BOOK_FILE = "./phonebook.json"
const __ID_RSA_FILE       = "./id_rsa"

var _flags_logfile     = flag.String("l", "", "Default log `file` (default stderr)")
var _flags_serialPort  = flag.String("p", "", "Serial `port` (eg, /dev/ttyS0")
var _flags_serialSpeed = flag.Int("S", 115200,
	"Serial Port `speed (bps)` between DTE and DCE")
var _flags_phoneBook   = flag.String("a", __ADDRESS_BOOK_FILE, "Phonebook `file`")
var _flags_telnetPort  = flag.Uint("t", 20000,
	"Network `port number` for inbound telnet sessions")
var _flags_sshdPort    = flag.Uint("s", 22000,
	"Network `port number` for inbound sshd sessions")
var _flags_privateKey  = flag.String("k", __ID_RSA_FILE, "SSH Private Key `file`")


