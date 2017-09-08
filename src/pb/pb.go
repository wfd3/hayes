package main

import (
	"fmt"
	"strings"
)
type h struct {
	host string
	protocol string
}
var addressbook map[string] *h

func isNumber(n string) (string, bool) {

	check := func(r rune) rune {
		if (r >= '0' && r <= '9') {
			return r
		} 
		return rune(-1)
	};
	m := strings.Map(check, n)
	return m, m != ""
}

func main() {
	var b []string = []string{"111-2222","4444444","badbadb"}
	for _, n := range b {
		m, ok := isNumber(n)
		fmt.Printf("n: %s, m: %s, ok: %t\n", n, m, ok)
	}
}
