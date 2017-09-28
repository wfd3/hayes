package hayes

import (
	"fmt"
	"log"
	"github.com/tarm/serial"
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

func getByte() byte {
	return byte(C.getch())
}

type serialPort struct {
	console bool
	port *serial.Port
	regs *Registers
	log *log.Logger
}

func setupSerialPort(port string, regs *Registers, log *log.Logger) (*serialPort) {
	var s serialPort

	s.console = port == ""
	s.regs = regs
	s.log = log

	if s.console {
		s.log.Print("Using stdin/stdout as DTE")
		return &s
	}

	s.log.Printf("Using serial port %s", *_flags_serialPort)
	c := &serial.Config{Name: port, Baud: 115200}
	p, err := serial.OpenPort(c)
        if err != nil {
                s.log.Fatal(err)
        }
	s.port = p
	return &s
}

func (s *serialPort) Read(p []byte) (int, error) {
	if s.console {
		p[0] = byte(C.getch())
		// mappings
		if p[0] == 127 {
			p[0] = s.regs.Read(REG_BS_CH)
		}
		return 1, nil
	}

	b := make([]byte, 1)
	i, err := s.port.Read(b)
	return i, err
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
			p[0] = s.regs.Read(REG_BS_CH)
		}
		// end of key mappings

		// Handle BS
		str := string(p)
		if p[0] == s.regs.Read(REG_BS_CH) {
			str = fmt.Sprintf("%c %c", s.regs.Read(REG_BS_CH),
				s.regs.Read(REG_BS_CH))
		} 
		return fmt.Printf("%s", str) // This should be the
					     // only fmt.Print* in the
					     // codebase
	}

	return s.port.Write(p)
}

func (s *serialPort) WriteByte(p byte) (int, error) {
	out := make([]byte, 1)
	out[0] = p
	return s.Write(out)
}

func (s *serialPort) Printf(format string, a ...interface{}) error {
	str := fmt.Sprintf(format, a...)
	_, err := s.Write([]byte(str))
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

func (s *serialPort) getChars(c chan byte) {

	in := make([]byte, 1)
	for {
		if _, err := s.Read(in); err != nil {
			s.log.Print("Read(): ", err)
		}

		c <- in[0]
	}
}
