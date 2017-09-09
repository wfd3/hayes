package hayes

import (
	"flag"
)

const __ADDRESS_BOOK_FILE = "./phonebook.json"

var _flags_addressbook = flag.String("a", __ADDRESS_BOOK_FILE, "Address book file")
var _flags_console     = flag.Bool("c", false,
	"Use the console rather than serial port for DTE")
