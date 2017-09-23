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

func setupSerialPort(regs *Registers, log *log.Logger) (*serialPort) {
	var s serialPort

	if *_flags_serialPort == "" {
		s.console = true
	} else {
		s.console = false
	}
	s.regs = regs
	s.log = log

	if s.console {
		s.log.Print("Using stdin/stdout as DTE")
		return &s
	}

	c := &serial.Config{Name: *_flags_serialPort, Baud: 115200}
	p, err := serial.OpenPort(c)
        if err != nil {
                s.log.Fatal(err)
        }
	s.log.Printf("Using %s", *_flags_serialPort)
	s.port = p
	return &s
}

func (s *serialPort) Read(p []byte) (int, error) {
	if s.console {
		p[0] = byte(C.getch())
		// mapping 
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
	return s.Printf("%s", a...)
}

func (s *serialPort) Println(a ...interface{}) error {
	return s.Printf("%s\n", a...)
}

func (m *Modem) getChars() {

	in := make([]byte, 1)
	for {
		if _, err := m.serial.Read(in); err != nil {
			m.log.Print("Read(): ", err)
		}

		charchannel <- in[0]
	}
	
}

var charchannel chan byte

// Consume bytes from the serial port and process or send to
// remote as per m.mode
func (m *Modem) readSerial() {
	var c byte
	var s string
	var lastThree [3]byte
	var idx int
	var regs *Registers
	var countAtTick uint64
	var countAtLastTick uint64
	var waitForOneTick bool

	charchannel = make(chan byte, 1)
	go m.getChars()

	countAtTick = 0
	for {
		select {
		case <- m.timer.C:
			if m.mode == COMMANDMODE { // Skip this if in COMMAND mode
				continue
			}

			// Look for the command escape sequence
			// (see http://www.messagestick.net/modem/Hayes_Ch1-4.html)
			// Basically:
			// 1s of silence, "+++", 1s of silence.
			// So, count the incoming chars between ticks, saving
			// the previous tick's count.  If you see
			// countAtTick == 3 && CountAtLastTick == 0 && the last
			// three characters are "+++", wait one more tick.  If
			// countAtTick == 0, the guard sequence was detected.
			
			if countAtTick == 3 && countAtLastTick == 0 &&
				lastThree == escSequence { 
				waitForOneTick = true
			} else if waitForOneTick && countAtTick == 0 {
				m.mode = COMMANDMODE
				m.prstatus(OK) // signal that we're in command mode
			} else {
				waitForOneTick = false
			}
			countAtLastTick = countAtTick
			countAtTick = 0
			continue

		case c = <- charchannel:
			countAtTick++

		}

		switch m.mode {
		case COMMANDMODE:
			regs = m.registers // Reload regs in case we reset the modem
			if m.echoInCmdMode { // Echo back to the DTE
				m.serial.WriteByte(c)
			}

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
			} else if c == regs.Read(REG_BS_CH)  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else if c == regs.Read(REG_LF_CH)  ||
				c == regs.Read(REG_BS_CH) && len(s) == 0 {
				// ignore naked CR's & BS if s is already empty
			} else {
				s += string(c)
			}

		case DATAMODE:
			// Look for the command escape sequence
			if c != m.registers.Read(REG_ESC_CH) {
				lastThree = [3]byte{' ', ' ', ' '}
				idx = 0
			} else {
				lastThree[idx] = c
				idx = (idx + 1) % 3
			}
			
			// Send to remote
			// TODO: make sure the LED says on long enough
			if m.offHook() && m.conn != nil {
				m.led_SD_on()
				out := make([]byte, 1)
				out[0] = c
				m.conn.Write(out)
				m.led_SD_off()	
			}
		}
	}
}

