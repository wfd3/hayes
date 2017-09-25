package hayes

import (
	"strings"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

type jsonPhonebook []struct {
	Stored   int    `json:"Stored"`
	Phone    string `json:"Phone"`
	Host     string `json:"Host"`
	Protocol string `json:"Protocol"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}
type Phonebook map[int]pb_host
type pb_host struct {
	phone    string 
	host     string 
	protocol string 
	username string 
	password string 
}	

func LoadPhoneBook() (*Phonebook, error) {
	var pb Phonebook
	var jpb jsonPhonebook

	b, err := ioutil.ReadFile(*_flags_phoneBook)
	if err != nil {
		return nil, fmt.Errorf("Phone book file flag not set (%s)", err)
	}

	if err = json.Unmarshal(b, &jpb); err != nil {
		return nil, err
	}

	// Covert the json parsed array into a map
	pb = make(Phonebook)
	for i, _ := range jpb {
		pb[jpb[i].Stored] = pb_host{jpb[i].Phone, jpb[i].Host,
			jpb[i].Protocol, jpb[i].Username, jpb[i].Password}
	}
	return &pb, nil
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

func (p Phonebook) String() string {
	if len(p) == 0 {
		return "Phone Book is empty"
	}
	s := "Phone Book:\n"
	for i := range p {
		s += fmt.Sprintf(" -- %d: %+v\n", i, p[i])
	}
	return s
}

func (p Phonebook) Lookup(number string) (*pb_host, error) {
	if !isValidPhoneNumber(number) {
		return nil, fmt.Errorf("Invalid phone number '%s'", number)
	}
	sanitized_index, err := sanitizeNumber(number)
	if err != nil {
		return nil, err
	}
	for _, h := range p {
		sanitized_n, _ := sanitizeNumber(h.phone)
		if sanitized_index == sanitized_n {
			return &h, nil
		}
	}
	err = fmt.Errorf("Number '%s' not in phone book", number)
	return nil, err
}

func (p Phonebook) LookupStoredNumber(n int) (string, error) {
	if n > len(p) {
		return "", fmt.Errorf("No stored number at entry %d", n)
	}
	if n > 2 {
		return "", fmt.Errorf("ATDS=n, n=0, 1, 2")
	}
	return p[n].phone, nil
}

// Returns phone|host|protocol|username|password
func splitAmperZ(cmd string) (string, string, string, string, string, error) {
	s := strings.Split(cmd, "|")
	if len(s) != 5 {
		return "", "", "", "", "", fmt.Errorf("Malformated AT&Z command")
	}
	return s[0], s[1], s[2], s[3], s[4], nil
}

func (p Phonebook) storeNumber(phone string, pos int) error {

	fmt.Printf("storeNumber: %s at %d\n", phone, pos)
	phone, host, proto, username, pw, err := splitAmperZ(phone)
	if err != nil {
		fmt.Println(err)
		return err
	}

	if !supportedProtocol(proto) {
		return fmt.Errorf("Unsupported protocol '%s'", proto)
	}
	if !isValidPhoneNumber(phone) {
		return fmt.Errorf("Invalid phone number '%s'", phone)
	}

	if pb, err := p.Lookup(phone); err == nil {
		// TODO: check to see if p[pos] is this number
		inbook, _ := sanitizeNumber(pb.phone)
		passed, _ := sanitizeNumber(phone)
		if inbook == passed {
			return fmt.Errorf("Number already exisits at anohter positiong in phonebook")
		}
	}

	p[pos] = pb_host{host, phone, proto, username, pw}
	return nil
}
