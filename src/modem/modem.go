package main

import (
	"hayes"
	"flag"
)

func main() {
	var m hayes.Modem
	flag.Parse()
	m.PowerOn()
}
