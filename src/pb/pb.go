package main

import (
	"fmt"
	"encoding/csv"
	"os"
	"io"
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
		} else if r == '-' || r == '(' || r == ')' || r == ' ' {
			return rune(-1)
		} else {
			return '*'
		}
	};
	m := strings.Map(check, n)
	if strings.Contains(m, "*") {
		return "", false
	}
	return m, true
}

func main() {

	// number host-to-eol

	addressbook = make(map[string] *h)

	f, err := os.Open("./phone.csv")
	if err != nil {
		panic(err)
	}

	r := csv.NewReader(f)
	r.Comma = ' '
	r.Comment = '#'
	r.FieldsPerRecord = 3
	r.TrimLeadingSpace = true

	for { 
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		m, ok := isNumber(record[0])
		if !ok {
			fmt.Printf("# '%s' is not a valid number\n", record[0])
			continue
		}
		addressbook[m] = &h{record[1], record[2]}
	}
	
	for i :=  range addressbook {
		fmt.Printf("%s: %s %s\n", i, addressbook[i].host,
			addressbook[i].protocol)
	}
}
