package hayes

import (
	"flag"
)

const __ADDRESS_BOOK_FILE = "./phonebook.json"

var _flags_serialPort  = flag.String("p", "", "Serial port")
var _flags_phoneBook   = flag.String("a", __ADDRESS_BOOK_FILE, "Phonebook file")
var _flags_telnetPort  = flag.Uint("t", 20000, "Port for inbound telnet sessions")
var _flags_sshdPort    = flag.Uint("s", 22000, "Port for inbound sshd sessions")
var _flags_privateKey  = flag.String("k", "id_rsa", "SSH Private Key File")
