package hayes

import (
	"strings"
	"encoding/json"
	"io/ioutil"
	"fmt"
)

type abList []ab_host
type ab_host struct {
	Stored   int    `json:"Stored"`
	Phone    string `json:"Phone"`
	Host     string `json:"Host"`
	Protocol string `json:"Protocol"`
	Username string `json:"Username"`
	Password string `json:"Password"`
}

func isValidPhoneNumber(n string) error {
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
		return fmt.Errorf("'%s' is not a valid phone number", n)
	}
	return nil
}

func sanitizeNumber(n string) (string, error) {
	// strip (, ), -, + from a phonenumber.
	check := func(r rune) rune {
		switch r {
		case '(', ')', '-', '+': return rune(-1)
		default: return r
		}
	};
	if err := isValidPhoneNumber(n); err != nil {
		return "", err;
	}
	return strings.Map(check, n), nil
}

func (m *Modem) printAddressBook() {

	if len(m.addressbook) == 0 {
		fmt.Println("Address Book: empty")
		return
	}
	fmt.Println("Address Book:")
	for _, h := range m.addressbook {
		fmt.Printf(" -- %+v\n", h)
	}
}

func (m *Modem) loadAddressBook() {
	var ab abList

	b, err := ioutil.ReadFile(*_flags_addressbook)
	if err != nil {
		m.log.Printf("Address book file flag not set (%s)", err)
		return
	}

	if err = json.Unmarshal(b, &ab); err != nil {
		m.log.Print("Unmarshal error: ", err)
		return
	}

	m.addressbook = ab
}

func (m *Modem) lookupAddressbook(number string) (*ab_host, error) {
	err := isValidPhoneNumber(number)
	if err != nil {
		return nil, err
	}
	sanitized_index, err := sanitizeNumber(number)
	if err != nil {
		return nil, err
	}
	for _, h := range m.addressbook {
		sanitized_n, _ := sanitizeNumber(h.Phone)
		if sanitized_index == sanitized_n {
			return &h, nil
		}
	}
	err = fmt.Errorf("Number '%s' not in address book", number)
	m.log.Print(err)
	return nil, err
}

func (m *Modem) storedNumber(n int) (string, error) {
	for _, h := range m.addressbook {
		if h.Stored == n {
			return h.Phone, OK
		}
	}
	m.log.Printf("No stored number at entry %d", n)
	return "", fmt.Errorf("No stored number at entry %d", n)
}

func (m *Modem) storeNumber(phone string, pos int) error {
	// This can't be done in this implemenetation.  Return ERROR always.
	return fmt.Errorf("Storing numbers not yet implemented")
}
