package main

import (
	"fmt"
	tarmserial "github.com/tarm/serial"
	"log"
	"strings"
)

/*
#include <stdio.h>
#include <unistd.h>
#include <termios.h>
char getch(){
    char ch = 0;
    struct termios old = {0};
    fflush(stdout);
    if( tcgetattr(0, &old) < 0 ) perror("tcsetattr()");
    old.c_lflag &= ~ICANON;
    old.c_lflag &= ~ECHO;
    old.c_cc[VMIN] = 1;
    old.c_cc[VTIME] = 0;
    if( tcsetattr(0, TCSANOW, &old) < 0 ) perror("tcsetattr ICANON");
    if( read(0, &ch,1) < 0 ) perror("read()");
    old.c_lflag |= ICANON;
    old.c_lflag |= ECHO;
    if(tcsetattr(0, TCSADRAIN, &old) < 0) perror("tcsetattr ~ICANON");
    return ch;
}
*/
import "C"

type serialPort struct {
	console bool
	port    *tarmserial.Port
	log     *log.Logger
	channel chan byte
}

func setupSerialPort(port string, speed int, charchannel chan byte,
	log *log.Logger) *serialPort {
	var s serialPort

	s.console = port == ""
	s.log = log
	s.channel = charchannel

	if s.console {
		s.log.Print("Using stdin/stdout as DTE")
	} else {

		s.log.Printf("Using serial port %s at %d bps", port, speed)
		c := &tarmserial.Config{Name: port, Baud: speed}
		p, err := tarmserial.OpenPort(c)
		if err != nil {
			s.log.Fatal(err)
		}
		s.port = p
	}

	go s.getChars()
	return &s
}

func (s *serialPort) Flush() error {
	if s.console || s.port == nil {
		return nil
	}

	s.log.Print("flushing serial port")
	return s.port.Flush()
}

func (s *serialPort) Read(p []byte) (int, error) {
	if s.console {
		p[0] = byte(C.getch())
		// mappings
		switch p[0] {
		case 127:
			p[0] = registers.Read(REG_BS_CH)
		case '\n':
			p[0] = registers.Read(REG_CR_CH)
		}
		return 1, nil
	}

	return s.port.Read(p)
}

func (s *serialPort) getChars() {

	in := make([]byte, 1)
	for {
		if _, err := s.Read(in); err != nil {
			s.log.Print("Read(): ", err)
		}

		s.channel <- in[0]
	}
}

func (s *serialPort) Write(p []byte) (int, error) {
	if s.console {
		// If we're writing to stdout, some static key mapping
		// is needed

		// Ignore anything above ASCII 127 or the ASCII escape
		if p[0] > 127 || p[0] == 27 {
			return 0, nil
		}
		// ASCII DEL -> ASCII BS
		if p[0] == 127 {
			p[0] = registers.Read(REG_BS_CH)
		}
		// end of key mappings

		// Handle BS
		str := string(p)
		if p[0] == registers.Read(REG_BS_CH) {
			str = fmt.Sprintf("%c %c", registers.Read(REG_BS_CH),
				registers.Read(REG_BS_CH))
		}
		return fmt.Printf("%s", str) // This should be the
		// only fmt.Print* in the
		// codebase
	}

	return s.port.Write(p)
}

func (s *serialPort) WriteByte(p byte) (int, error) {
	var out []byte

	// map '\n' to '\n\r'
	switch p {
	case registers.Read(REG_CR_CH):
		out = make([]byte, 2)
		out[0] = p
		out[1] = registers.Read(REG_LF_CH)
	default:
		out = make([]byte, 1)
		out[0] = p
	}

	return s.Write(out)
}

func (s *serialPort) Printf(format string, a ...interface{}) error {
	out := fmt.Sprintf(format, a...)
	out = strings.Replace(out, "\n", "\n\r", -1)
	_, err := s.Write([]byte(out))
	return err
}

func (s *serialPort) Print(a ...interface{}) error {
	if a == nil {
		return nil
	}
	return s.Printf("%s", a...)
}

func (s *serialPort) Println(a ...interface{}) error {
	if a == nil {
		return s.Printf("\n")
	}
	return s.Printf("%s\n", a...)
}
