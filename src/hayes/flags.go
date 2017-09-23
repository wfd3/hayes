package hayes

import (
	"flag"
)

const __ADDRESS_BOOK_FILE = "./phonebook.json"

var _flags_serialPort  = flag.String("p", "/dev/ttyAMA0", "Serial port")
var _flags_addressbook = flag.String("a", __ADDRESS_BOOK_FILE, "Address book file")
var _flags_console     = flag.Bool("c", false,
	"Use the console rather than serial port for DTE")
var _flags_telnetPort  = flag.Uint("t", 20000, "Port for inbound telnet sessions")
var _flags_sshdPort    = flag.Uint("s", 22000, "Port for inbound sshd sessions")
