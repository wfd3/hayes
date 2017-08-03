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
}

func setupSerialPort(console bool) (*serialPort) {

	var s serialPort
	
	s.console = console
	if console {
		return &s
	}

	c := &serial.Config{Name: "/dev/ttyAMA0", Baud: 115200}
	p, err := serial.OpenPort(c)
        if err != nil {
                log.Fatal(err)
        }
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
		fmt.Printf("%s", string(p))
		return len(p), nil
	}
	return s.port.Write(p)
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

	out = make([]byte, 1)
	in  = make([]byte, 1)
	for {
		if _, err := m.serial.Read(in); err != nil {
			m.log.Fatal("Fatal Error: ", err)
		}

		// TODO: All this needs to be written now that we're
		// using the same io.ReadWriter interfaces for the
		// serial port
		c = in[0]
		
		// XXX becuse this is not just a modem program yet, some static
		// key mapping is needed 
		if c == 127 {	// ASCII DEL -> ASCII BS
			c = m.readReg(REG_BS_CH)
		}
		// Ignore anything above ASCII 127 or the ASCII escape
		if c > 127 || c == 27 { 
			continue
		}
		// end of key mappings

		if m.echo {
			fmt.Printf("%c", c)
			// XXX: handle backspace
			if c == m.readReg(REG_BS_CH) {
				fmt.Printf(" %c", c)
			}
		}

		switch m.mode {
		case COMMANDMODE:
			if c == m.readReg(REG_LF_CH) && s != "" {
				m.command(s)
				s = ""
			} else if c == m.readReg(REG_LF_CH) {
				// ignore naked CR's
			} else if c == m.readReg(REG_BS_CH)  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else {
				s += string(c)
			}

		case DATAMODE:
			if m.getHook() == OFF_HOOK && m.conn != nil {
				m.led_SD_on()
				out[0] = c
				m.conn.Write(out)
				time.Sleep(10 *time.Millisecond) // HACK!
				m.led_SD_off()	
				// TODO: make sure the LED says on long enough
			}

			// Look for the command escape sequence
			lastthree[idx] = c
			idx = (idx + 1) % 3
			guard_time =
				time.Duration(float64(m.readReg(REG_ESC_CODE_GUARD))				* 0.02) * time.Second
			
			if lastthree[0] == m.readReg(REG_ESC_CH) &&
				lastthree[1] == m.readReg(REG_ESC_CH) &&
				lastthree[2] == m.readReg(REG_ESC_CH) &&
				time.Since(sinceLastChar) >
				time.Duration(guard_time)  {
				m.mode = COMMANDMODE
				m.prstatus(OK) // signal that we're in command mode
				continue
			}
			if c != '+' {
				sinceLastChar = time.Now()
			}
		}
	}
}

