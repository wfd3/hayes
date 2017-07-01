package hayes

//
// Pretend to be a Hayes modem.
//
// References:
// - Hayes command/error documentation:
//    http://www.messagestick.net/modem/hayes_modem.html#Introduction
// - Sounds: https://en.wikipedia.org/wiki/Precise_Tone_Plan
// - RS232: https://en.wikipedia.org/wiki/RS-232
// - Serial Programming: https://en.wikibooks.org/wiki/Serial_Programming
// - Raspberry PI lib: github.com/stianeikeland/go-rpio
//

import (
	"os"
	"os/signal"
	"fmt"
	"time"
	"io"
	"sync"
	"runtime/pprof"
	"syscall"
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

////////////////////////////////////////////////////////////////////////////////////

const (
	COMMANDMODE = iota
	DATAMODE
)

const OFFHOOK = false
const ONHOOK = true
const __MAX_RINGS = 15
const __DELAY_MS = 20
const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

type ab_host struct {
	host string
	protocol string
	stored int 		// if 0-3, useable by AT&Z
}

//Basic modem struct
type Modem struct {
	mode int
	onhook bool
	echo bool
	speakermode int
	volume int
	verbose bool
	quiet bool
	lastcmds []string
	lastdialed string
	rlock sync.RWMutex	// Lock for registers map (r)
	r map[byte]byte
	curreg int
	conn io.ReadWriteCloser
	pins Pins
	leds Pins
	d [10]int
	connect_speed int
	linebusy bool
	linebusylock sync.RWMutex
	addressbook map[string] *ab_host
}

// Is the phone line busy?
func (m *Modem) getLineBusy() bool {
	m.linebusylock.RLock()
	defer m.linebusylock.RUnlock()
	return m.linebusy
}	

func (m *Modem) setLineBusy(b bool) {
	m.linebusylock.Lock()
	defer m.linebusylock.Unlock()
	m.linebusy = b
}

// Setup/reset modem.  Also ATZ, conveniently.
func (m *Modem) reset() (int) {
	m.onHook()
	m.lowerDSR()
	m.lowerCTS()
	m.lowerRI()

	m.echo = true		// Echo local keypresses
	m.quiet = false		// Modem offers return status
	m.verbose = true	// Text return codes
	m.volume = 1		// moderate volume
	m.speakermode = 1	// on until other modem heard
	m.lastcmds = nil
	m.lastdialed = ""
	m.connect_speed = 0
	m.setLineBusy(false)
	m.setupRegs()
	m.setupDebug()

	m.loadAddressBook()

	time.Sleep(250 *time.Millisecond) // Make it look good
	
	m.raiseDSR()
	m.raiseCTS()		// Ready for DTE to send us data
	return OK
}

// Watch a subset of pins and registers and toggle the LED as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {
	for {
		if m.readDTR() {
			m.led_TR_on()
		} else { 
			if m.getHook() == OFF_HOOK && m.conn != nil {
				// DTE Dropped DTR, hang up the phone if DTR is not
				// reestablished withing S25 * 1/100's of a second
				time.Sleep(time.Duration(m.readReg(REG_DTR_DELAY)) *
					100 * time.Millisecond)
				if !m.readDTR() && m.getHook() == OFF_HOOK &&
					m.conn != nil {
					m.onHook()
				}
			}
			m.led_TR_off()
		}

		if m.connect_speed > 19200 {
			m.led_HS_on()
		} else {
			m.led_HS_off()
		}
			

		// debug
		if m.d[1] == 2 {
			m.raiseDSR()
			m.raiseCTS()
			m.d[1] = 0
		}
		if m.d[1] == 1 {
			m.lowerDSR()
			m.lowerCTS()
			m.d[1] = 0
		}

		if m.d[2] != 0 {
			m.ledTest(m.d[2])
			m.d[2] = 0
		}

		time.Sleep(250 * time.Millisecond)
	}
}


// Catch ^C, reset the HW pins
func (m *Modem) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	for {
		// Block until a signal is received.
		s := <-c
		fmt.Println("Got signal:", s)
		if s == syscall.SIGINT {
			m.clearPins()
			os.Exit(0)
		}
		if s == syscall.SIGQUIT {
			pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		}
	}
}

// Boot the modem
func (m *Modem) PowerOn() {
	m.setupPins()	      
	m.reset()	      // Setup modem inital state (or reset initial state)
	
	go m.signalHandler()	// Catch signals in a different thread
	go m.handlePINs()       // Monitor input pins & internal registers
	go m.handleModem()	// Handle in-bound bytes in a seperate goroutine

	// Signal to DTE that we're ready
	m.raiseDSR()
	m.raiseCTS()

	// Tell user we're ready
	m.prstatus(OK)

	// Consume bytes from the serial port and process or send to remote
	// as per m.mode
	var c byte
	var s string
	var lastthree [3]byte
	var out []byte
	var idx int
	var guard_time time.Duration
	var sinceLastChar time.Time

	out = make([]byte, 1)
	for {
		// XXX becuse this is not just a modem program yet, some static
		// key mapping is needed 
		c = byte(C.getch())
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

