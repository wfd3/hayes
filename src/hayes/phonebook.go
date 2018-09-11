package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
)

type Phonebook struct {
	entries  map[int]pb_host
	filename string
	log      *log.Logger
}
type pb_host struct {
	Phone    string `json:"Phone"`
	Host     string `json:"Host"`
	Protocol string `json:"Protocol"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func NewPhonebook(filename string, log *log.Logger) *Phonebook {
	var pb Phonebook
	pb.filename = filename
	pb.log = log
	pb.entries = make(map[int]pb_host)
	return &pb
}

func (p *Phonebook) Load() error {
	b, err := ioutil.ReadFile(p.filename)
	if err != nil {
		e := fmt.Errorf("Can't read phonebook file %s: %s",
			p.filename, err)
		p.log.Print(e)
		return e
	}

	if err = json.Unmarshal(b, &p.entries); err != nil {
		p.log.Print(err)
		return err
	}

	return nil
}

func (p *Phonebook) Write() error {
	b, err := json.MarshalIndent(p.entries, "", "\t")
	if err != nil {
		p.log.Print(err)
		return err
	}
	err = ioutil.WriteFile(p.filename, b, 0644)
	if err != nil {
		p.log.Print(err)
	}
	return err
}

func (p *Phonebook) String() string {
	var s string

	count := len(p.entries)
	if count  == 0 {
		return "0=\n1=\n2=\n3=\n"
	}
	if count < 3 {
		count = 3
	}

	for i := 0; i <= count; i++ {
		entry, found := p.entries[i]
		if found {
			phone, _ := sanitizeNumber(entry.Phone)
			if phone == "" {
				phone = entry.Phone
			}
			s += fmt.Sprintf("%d=%s (%s, '%s'/'%s')\n", i, phone,
				entry.Host, entry.Username, entry.Password)
		} else {
			s += fmt.Sprintf("%d=\n", i)
		}
	}
	return s
}

func isValidPhoneNumber(n string) bool {
	// 0-9, A, B, C, D, #, * are valid Hayes phone number digits
	check := func(r rune) rune {
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			return r
		case 'A', 'B', 'C', 'D', '#', '*':
			return r
		case '(', ')', '-', '+':
			return r
		default:
			return rune(-1)
		}
	}
	m := strings.Map(check, n)
	if m != n {
		return false
	}
	return true
}

func sanitizeNumber(n string) (string, error) {
	// strip (, ), -, + from a phonenumber.
	check := func(r rune) rune {
		switch r {
		case '(', ')', '-', '+':
			return rune(-1)
		default:
			return r
		}
	}
	if !isValidPhoneNumber(n) {
		return "", fmt.Errorf("Invalid phone number '%s'", n)
	}
	return strings.Map(check, n), nil
}

func (p *Phonebook) Lookup(number string) (string, string, string, string, error) {
	if !isValidPhoneNumber(number) {
		return "", "", "", "",
			fmt.Errorf("Invalid phone number '%s'", number)
	}
	sanitized_index, err := sanitizeNumber(number)
	if err != nil {
		return "", "", "", "", err
	}
	for _, h := range p.entries {
		sanitized_n, _ := sanitizeNumber(h.Phone)
		if sanitized_index == sanitized_n {
			return h.Host, h.Protocol, h.Username, h.Password, nil
		}
	}
	err = fmt.Errorf("Number '%s' not in phone book", number)
	return "", "", "", "", err
}

func (p *Phonebook) LookupStoredNumber(n int) (string, error) {
	pb, ok := p.entries[n]
	if !ok {
		return "", fmt.Errorf("No entry at position %d", n)
	}
	return pb.Phone, nil
}

// Returns phone|host|protocol|username|password
func splitAmperZ(cmd string) (string, string, string, string, string, error) {
	s := strings.Split(cmd, "|")
	if len(s) != 5 {
		return "", "", "", "", "", fmt.Errorf("Malformated AT&Z command")
	}
	return s[0], s[1], s[2], s[3], s[4], nil
}

func (p *Phonebook) Add(pos int, phone string) error {
	phone, host, proto, username, pw, err := splitAmperZ(phone)
	if err != nil {
		return err
	}

	if !supportedProtocol(proto) {
		return fmt.Errorf("Unsupported protocol '%s'", proto)
	}
	if !isValidPhoneNumber(phone) {
		return fmt.Errorf("Invalid phone number '%s'", phone)
	}

	passed, _ := sanitizeNumber(phone)
	inbook, _ := sanitizeNumber(p.entries[pos].Phone)
	if inbook == passed {
		return fmt.Errorf("Number alreasy exists at position %d in phonebook", pos)
	}

	if _, _, _, _, err = p.Lookup(phone); err == nil {
		return fmt.Errorf("Number already exisits at another position in phonebook")
	}

	p.entries[pos] = pb_host{phone, host, proto, username, pw}
	p.Write()
	return nil
}

func (p *Phonebook) Delete(pos int) error {
	if _, ok := p.entries[pos]; ok {
		delete(p.entries, pos)
		return p.Write()
	}
	return nil
}
