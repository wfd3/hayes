package hayes

import (
	"strings"
	"encoding/json"
	"io/ioutil"
	"fmt"
	"sort"
	"log"
)

type jsonPhonebook []jsonPhonebookEntry
type jsonPhonebookEntry struct {
	Stored   int    `json:"Stored"`
	Phone    string `json:"Phone"`
	Host     string `json:"Host"`
	Protocol string `json:"Protocol"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}

type Phonebook struct {
	entries map[int]pb_host
	filename string
	log *log.Logger
}
type pb_host struct {
	phone    string 
	host     string 
	protocol string 
	username string 
	password string 
}

func NewPhonebook(filename string, log *log.Logger) *Phonebook {
	var pb Phonebook
	pb.filename = filename
	pb.log = log
	return &pb
}

func (p *Phonebook) Load() error {
	var jpb jsonPhonebook

	b, err := ioutil.ReadFile(p.filename)
	if err != nil {
		e := fmt.Errorf("Can't read phonebook file %s: %s",
			p.filename, err)
		p.log.Print(e)
		return e
	}

	if err = json.Unmarshal(b, &jpb); err != nil {
		p.log.Print(err)
		return err
	}

	// Covert the json parsed array into a map
	p.entries = make(map[int]pb_host)
	for i, _ := range jpb {
		p.entries[jpb[i].Stored] = pb_host{jpb[i].Phone, jpb[i].Host,
			jpb[i].Protocol, jpb[i].Username, jpb[i].Password}
	}
	return nil
}

func (p *Phonebook) Write() error {
	var j jsonPhonebook
	var jentry jsonPhonebookEntry
	var keys []int	

	for k := range p.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, i := range keys {
		e := p.entries[i]
		jentry = jsonPhonebookEntry{i, e.phone, e.host, e.protocol,
			e.username, e.password}
		j = append(j, jentry)
	}

	b, err := json.MarshalIndent(j, "", "\t")
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
	if len(p.entries) == 0 {
		return "Phone Book is empty\n"
	}

	var s string
	var keys []int	
	for k := range p.entries {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	max := keys[len(keys)-1]

	for i := 0; i <= max; i++ {
		s += fmt.Sprintf(" -- %d: %+v\n", i, p.entries[i])
	}
	return s
}

func isValidPhoneNumber(n string) bool {
	// 0-9, A, B, C, D, #, * are valid Hayes phone number 'digits
	check := func(r rune) rune {
		switch r {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9': return r
		case 'A', 'B', 'C', 'D', '#', '*': return r
		case '(', ')', '-', '+': return r
		default: return rune(-1)
		}
	};
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
		case '(', ')', '-', '+': return rune(-1)
		default: return r
		}
	};
	if !isValidPhoneNumber(n) {
		return "", fmt.Errorf("Invalid phone number '%s'", n)
	}
	return strings.Map(check, n), nil
}

func (p *Phonebook) Lookup(number string) (*pb_host, error) {
	if !isValidPhoneNumber(number) {
		return nil, fmt.Errorf("Invalid phone number '%s'", number)
	}
	sanitized_index, err := sanitizeNumber(number)
	if err != nil {
		return nil, err
	}
	for _, h := range p.entries {
		sanitized_n, _ := sanitizeNumber(h.phone)
		if sanitized_index == sanitized_n {
			return &h, nil
		}
	}
	err = fmt.Errorf("Number '%s' not in phone book", number)
	return nil, err
}

func (p *Phonebook) LookupStoredNumber(n int) (string, error) {
	pb, ok := p.entries[n]
	if !ok {
		return "", fmt.Errorf("No entry at position %d", n)
	}
	return pb.phone, nil
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
	inbook, _ := sanitizeNumber(p.entries[pos].phone)
	if inbook == passed {
		return fmt.Errorf("Number alreasy exists at position %d in ",
			"phonebook", pos)
	}
	
	if pb, err := p.Lookup(phone); err == nil {
		inbook, _ = sanitizeNumber(pb.phone)
		if inbook == passed {
			return fmt.Errorf("Number already exisits at another ",
				"position in phonebook")
		}
	}

	p.entries[pos] = pb_host{host, phone, proto, username, pw}
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