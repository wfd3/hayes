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
	"net"
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
const __MAX_RINGS = 5
const __CONNECT_TIMEOUT = __MAX_RINGS * 6 * time.Second

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
	r [255]byte
	curreg byte
	conn net.Conn
	pins Pins
	d [10]int
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
	m.curreg = 0		// current register selected (from ATSn)
	m.setupRegs()
	m.setupDebug()

	time.Sleep(500 *time.Millisecond) // Make it look good
	
	m.raiseDSR()
	m.raiseCTS()		// Ready for DTE to send us data
	return OK
}

// Watch a subset of pins and registers and toggle the LED as apropriate
// Must be a goroutine
func (m *Modem) handlePINs() {
	for {
		// MR LED
		if m.readCTS() && m.readDSR() {
			m.led_MR_on()
		} else {
			m.led_MR_off()
		}
		
		// AA LED
		if m.r[REG_AUTO_ANSWER] != 0 {
			m.led_AA_on()
		} else {
			m.led_AA_off()
		}

		// TR LED
		if m.readDTR() {
			m.led_TR_on()
		} else {
			m.led_TR_off()
		}

		// CD LED
		if m.readCD() {
			m.led_CD_on()
		} else {
			m.led_CD_off()
		}

		// OH LED
		if !m.onhook {
			m.led_OH_on()
		} else {
			m.led_OH_off()
		}

		// RI LED
		if m.readRI() {
			m.led_RD_on()
		} else {
			m.led_RD_off()
		}

		// CS LED
		if m.readCTS() {
			m.led_CS_on()
		} else {
			m.led_CS_off()
		}

		// DTR PIN
		if m.readDTR() == false && !m.onhook && m.conn != nil {
			// DTE Dropped DTR, hang up the phone if DTR is not
			// reestablished withing S25 * 1/100's of a second
			time.Sleep(time.Duration(m.r[REG_DTR_DELAY]) * 100 *
				time.Millisecond)
			if m.readDTR() == false && !m.onhook && m.conn != nil {
				m.onHook()
			}
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

		time.Sleep(250 * time.Millisecond)
	}
}

func (m *Modem) handleModem() {
	// Handle:
	// - passing bytes from the modem to the serial port (stdout for now)
	// - accepting incoming connections (ie, noticing the phone ringing)
	// - other housekeeping tasks (eg, clearing the ring counter)
	//
	// This must be a goroutine.

	// Clear the ring counter if there's been no rings for at least 8 seconds
	last_ring_time := time.Now()
	go func() {		
		for range time.Tick(8 * time.Second) {
			if time.Since(last_ring_time) >= 8 * time.Second {
				m.r[REG_RING_COUNT] = 0
			}
		}
	}()

	l, err := net.Listen("tcp", ":20000")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			debugf("l.Accept(): %s\n", err)
			continue
		}

		if !m.onhook {	// "Busy" signal. 
			conn.Close()
			continue
		}

		for i := byte(0); i < __MAX_RINGS; i++ {
			last_ring_time = time.Now()
			m.prstatus(RING)
			if !m.onhook { // computer has issued 'ATA' 
				m.conn = conn
				conn = nil
				goto answered
			}

			// Simulate the "2-4" pattern for POTS ring signal (2
			// seconds of high voltage ring signal, 4 seconds
			// of silence)

			// Ring for 2s
			d := 0
			m.raiseRI()
			for m.onhook  && d < 2000 {
				time.Sleep(250 * time.Millisecond)
				d += 250
				if !m.onhook { // computer has issued 'ATA' 
					m.conn = conn
					conn = nil
					goto answered
				}
			}
			m.lowerRI()

			// If Auto Answer if enabled and we've
			// exceeded the configured number of rings to
			// wait before answering, answer the call.  We
			// do this here before the 4s delay as I think
			// it feels more correct.
			if m.r[REG_AUTO_ANSWER] > 0 {
				m.r[REG_RING_COUNT]++
				if m.r[REG_RING_COUNT] >= m.r[REG_AUTO_ANSWER] {
					m.answer()
				}
			}

			// Silence for 4s
			d = 0
			for m.onhook && d < 4000 {
				time.Sleep(250 * time.Millisecond)
				d += 250
				if !m.onhook { // computer has issued 'ATA' 
					m.conn = conn
					conn = nil
					goto answered
				}
			}
		}
		
		// At this point we've not answered and have timed out.
		if m.onhook {	// computer didn't answer
			conn.Close()
			continue
		}

	answered:
		// if we're here, the computer answered, so pass bytes
		// from the remote dialer to the serial port (for now, stdout)
		// as long as we're offhook, we're in DATA MODE and we have
		// valid carrier (m.comm != nil)
		//
		// TODO: this is line based, needed to be character based
		// TODO: Blink the RD LED somewhere in here.
		// TODO: XXX Racy -- if DTE issues ATA , causing m.onhook == true
		// before m.mode == DATAMODE, the for loop exits and hangs up.
		m.r[REG_RING_COUNT] = 0
		m.lowerRI()
		buf := make([]byte, 1)
		for !m.onhook  && m.mode == DATAMODE && m.conn != nil {
			if _, err = m.conn.Read(buf); err != nil {
				break
			}
			m.led_RD_on()
			fmt.Printf("%s", string(buf)) //  Send to DTE (stdout)
			m.led_RD_off()
		}

		// If we're here, we lost "carrier" somehow.
		m.led_RD_off()
		m.prstatus(NO_CARRIER)
		m.onHook()
		if conn != nil {
			conn.Close() // just to be safe?
		}
	}	
}

// Catch ^C, reset the HW pins
func (m *Modem) signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)
	m.clearPins()
	os.Exit(0)
}

// Boot the modem
func (m *Modem) PowerOn() {
	m.setupPins()		// Setup LED and GPIO pins
	m.reset()		// Setup modem inital state (or reset to
	                        // initial state)
	go m.signalHandler()	     

	go m.handlePINs()       // Monitor input pins & internal registers
	go m.handleModem()	// Handle in-bound in a seperate goroutine

	// Signal to DTE that we're ready
	m.raiseDSR()
	m.raiseCTS()
	// Tell user (if they're looking) we're ready
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
		// XXX
		// becuse this is not just a modem  program yet, some static
		// key mapping is needed 
		c = byte(C.getch())
		if c == 127 {	// ASCII DEL -> ASCII BS
			c = m.r[REG_BS_CH]
		}
		// Ignore anything above ASCII 127 or the ASCII escape
		if c > 127 || c == 27 { 
			continue
		}
		// end of key mappings

		if m.echo {
			fmt.Printf("%c", c)
			// XXX: handle backspace
			if c == m.r[REG_BS_CH] {
				fmt.Printf(" %c", c)
			}
		}

		switch m.mode {
		case COMMANDMODE:
			if c == m.r[REG_CR_CH] && s != "" {
				m.command(s)
				s = ""
			}  else if c == m.r[REG_BS_CH]  && len(s) > 0 {
				s = s[0:len(s) - 1]
			} else {
				s += string(c)
			}

		case DATAMODE:
			if m.onhook == false && m.conn != nil {
				m.led_SD_on()
				out = make([]byte, 1)
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
				time.Duration(float64(m.r[REG_ESC_CODE_GUARD]) *
				0.02) * time.Second

			if lastthree[0] == m.r[REG_ESC_CH] &&
				lastthree[1] == m.r[REG_ESC_CH] &&
				lastthree[2] == m.r[REG_ESC_CH] &&
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
	m.lowerDSR()
	m.lowerCTS()
}
