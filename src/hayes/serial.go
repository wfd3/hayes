package hayes

import (
	"fmt"
	"time"
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

func setupSerialPort(console bool, regs *Registers, log *log.Logger) (*serialPort) {

	var s serialPort
	
	s.console = console
	s.regs = regs
	s.log = log

	if console {
		s.log.Print("Using stdin/stdout as DTE")
		return &s
	}

	// TODO: Command line option for serial port
	c := &serial.Config{Name: "/dev/ttyAMA0", Baud: 115200}
	p, err := serial.OpenPort(c)
        if err != nil {
                s.log.Fatal(err)
        }
	s.log.Print("Using /dev/ttyAMA0")
	s.port = p
	return &s
}

func (s *serialPort) Read(p []byte) (int, error) {
	if s.console {
		c := byte(C.getch())
		p[0] = c;
		return 1, nil
	}
	b := make([]byte, 1)
	i, err := s.port.Read(b)
	return i, err
}

func (s *serialPort) Write(p []byte) (int, error) {
	if s.console {
		// If we're writing to stdout (console == true), some static
		// key mapping is needed 

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

func (s *serialPort) Printf(format string, a ...interface{}) error {
	str := fmt.Sprintf(format, a...)
	_, err := s.Write([]byte(str))
	return err
}

func (s *serialPort) Print(str string) error {
	return s.Printf("%s", str)
}

func (s *serialPort) Println(a ...interface{}) error {
	return s.Printf("%s\n", a...)
}

func (m *Modem) readSerial() {
	
	// Consume bytes from the serial port and process or send to remote
	// as per m.mode
	var c byte
	var s string
	var lastthree [3]byte
	var out []byte
	var in []byte
	var idx int
	var guard_time time.Duration
	var sinceLastChar time.Time
	var regs *Registers

	out = make([]byte, 1)
	in  = make([]byte, 1)
	for {
		regs = m.registers // Reload a copy into r if we reset the modem

		//TODO: make this a select{} with a channel for the
		//serial characters and a timer for the 1 second guard
		//time for the escape sequence.
		
		if _, err := m.serial.Read(in); err != nil {
			m.log.Fatal("Fatal Error: ", err)
		}

		// Echo back to the DTE
		if m.echoInCmdMode && m.mode == COMMANDMODE {
			m.serial.Write(in)
		}

		c = in[0]
		switch m.mode {
		case COMMANDMODE:
			// Accumulate chars in s until we read a CR, then process
			// s as a command.

			// 'A/' command, immediately exec.
			if (s == "A" || s == "a") && c == '/' && m.lastcmd != "" {
				m.serial.Println()
				m.command(m.lastcmd)
				s = ""
			} else if c == regs.Read(REG_LF_CH) && s != "" {
				m.command(s)
				s = ""
			} else if c == regs.Read(REG_LF_CH) {
				// ignore naked CR's
			} else if c == regs.Read(REG_BS_CH)  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else if c == regs.Read(REG_BS_CH) && len(s) == 0 {
				// ignore BS if s is already empty
			} else {
				s += string(c)
			}

		case DATAMODE:
			// Look for the command escape sequence
			// TODO: This is wrong
			// (see http://www.messagestick.net/modem/Hayes_Ch1-4.html)
			lastthree[idx] = c
			idx = (idx + 1) % 3
			guard_time =
				time.Duration(float64(regs.Read(REG_ESC_CODE_GUARD_TIME))				* 0.02) * time.Second
			
			if lastthree[0] == regs.Read(REG_ESC_CH) &&
				lastthree[1] == regs.Read(REG_ESC_CH) &&
				lastthree[2] == regs.Read(REG_ESC_CH) &&
				time.Since(sinceLastChar) >
				time.Duration(guard_time)  {
				m.mode = COMMANDMODE
				m.prstatus(OK) // signal that we're in command mode
				continue
			}
			if c != '+' {
				sinceLastChar = time.Now()
			}

			// Send to remote
			// TODO: make sure the LED says on long enough
			if m.getHook() == OFF_HOOK && m.conn != nil {
				m.led_SD_on()
				out[0] = c
				m.conn.Write(out)
				m.led_SD_off()	
			}
		}
	}
}

