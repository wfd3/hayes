package hayes

import (
	"flag"
)

var _flags_logfile     = flag.String("l", "", "Default logfile")
var _flags_user        = flag.String("u", "", "username")
var _flags_pw          = flag.String("p", "", "password")
var _flags_addressbook = flag.String("a", "", "Address book file")
var _flags_console     = flag.Bool("c", false, "Use the console rather than serial port for DTE")
