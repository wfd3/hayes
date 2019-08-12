package main

// i2c LCD adapter
// Adapted from github.com/davecheney/i2c, which was:
//   Adapted from http://think-bowl.com/raspberry-pi/installing-the-think-bowl-i2c-libraries-for-python/
//
// See https://orientdisplay.com/wp-content/uploads/2018/08/AMC1602AI2C-Full-1.pdf
// Also see: https://www.sunfounder.com/learn/sensor-kit-v2-0-for-raspberry-pi-b-plus/lesson-30-i2c-lcd1602-sensor-kit-v2-0-for-b-plus.html
// TODO: Need to find a real spec sheet.
//

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"sync"
	"time"
)

const (
	// I2C
	i2c_ADDR                 = 0x27
	i2c_SLAVE                = 0x0703

	// LCD Commands
	CMD_Clear_Display        = 0x01
	CMD_Return_Home          = 0x02
	CMD_Entry_Mode           = 0x04
	CMD_Display_Control      = 0x08
	CMD_Cursor_Display_Shift = 0x10
	CMD_Function_Set         = 0x20
	CMD_DDRAM_Set            = 0x80

	// LCD Options
	OPT_Increment = 0x02 // CMD_Entry_Mode
	// OPT_Display_Shift  = 0x01 // CMD_Entry_Mode
	OPT_Enable_Display = 0x04 // CMD_Display_Control
	OPT_Enable_Cursor  = 0x02 // CMD_Display_Control
	OPT_Enable_Blink   = 0x01 // CMD_Display_Control
	OPT_Display_Shift  = 0x08 // CMD_Cursor_Display_Shift
	OPT_Shift_Right    = 0x04 // CMD_Cursor_Display_Shift 0 = Left
	OPT_2_Lines        = 0x08 // CMD_Function_Set 0 = 1 line
	OPT_5x10_Dots      = 0x04 // CMD_Function_Set 0 = 5x7 dots

	// LCD instruction offsets
	INT_RS = 0
	INT_EN = 2
	INT_D4 = 4
	INT_D5 = 5
	INT_D6 = 6
	INT_D7 = 7
	INT_BACKLIGHT = 3
)

type Lcd struct {
	i2c *os.File
	backlight_state bool
	cols, rows int
	m sync.Mutex
}

func ioctl(fd, cmd, arg uintptr) (err error) {
        _, _, e1 := syscall.Syscall6(syscall.SYS_IOCTL, fd, cmd, arg, 0, 0, 0)
        if e1 != 0 {
                err = e1
        }
        return
}

func (lcd *Lcd) writeI2C(b byte) (int, error) {
	var buf [1]byte

	buf[0] = b
	return lcd.i2c.Write(buf[:])
}

func pinInterpret(pin, data byte, value bool) byte {
	if value {
		// Construct mask using pin
		var mask byte = 0x01 << (pin)
		data = data | mask
	} else {
		// Construct mask using pin
		var mask byte = 0x01<<(pin) ^ 0xFF
		data = data & mask
	}
	return data
}

func (lcd *Lcd) enable(data byte) {
	// Determine if black light is on and insure it does not turn off or on
	data = pinInterpret(INT_BACKLIGHT, data, lcd.backlight_state)
	lcd.writeI2C(data)
	lcd.writeI2C(pinInterpret(INT_EN, data, true))
	lcd.writeI2C(data)
}

func (lcd *Lcd) write(data byte, command bool) {
	var i2c_data byte

	// Add data for high nibble
	hi_nibble := data >> 4
	i2c_data = pinInterpret(INT_D4, i2c_data, (hi_nibble&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D5, i2c_data, ((hi_nibble>>1)&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D6, i2c_data, ((hi_nibble>>2)&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D7, i2c_data, ((hi_nibble>>3)&0x01 == 0x01))

	// # Set the register selector to 1 if this is data
	if !command {
		i2c_data = pinInterpret(INT_RS, i2c_data, true)
	}

	//  Toggle Enable
	lcd.enable(i2c_data)

	i2c_data = 0x00

	// Add data for high nibble
	low_nibble := data & 0x0F
	i2c_data = pinInterpret(INT_D4, i2c_data, (low_nibble&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D5, i2c_data, ((low_nibble>>1)&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D6, i2c_data, ((low_nibble>>2)&0x01 == 0x01))
	i2c_data = pinInterpret(INT_D7, i2c_data, ((low_nibble>>3)&0x01 == 0x01))

	// Set the register selector to 1 if this is data
	if !command {
		i2c_data = pinInterpret(INT_RS, i2c_data, true)
	}

	lcd.enable(i2c_data)
}

func (lcd *Lcd) command(data byte) {
	lcd.write(data, true)
}

func (lcd *Lcd) writeBuf(buf []byte) (int, error) {
	for _, c := range buf {
		lcd.write(c, false)
	}
	return len(buf), nil
}

func (lcd *Lcd) getLCDaddress(line, pos byte) (byte, error) {
	var address byte
	if line > byte(lcd.rows) {
		return 0, fmt.Errorf("invalid line number %d, max %d", line, lcd.rows)
	}
	if pos > byte(lcd.cols) {
		return 0, fmt.Errorf("invalid column number %d, max %d", pos, lcd.cols)
	}

	switch line {
	case 1:	address = pos
	case 2:	address = 0x40 + pos
	case 3:	address = 0x14 + pos
	case 4:	address = 0x54 + pos
	}

	return address, nil
}


func capstring(s string, l int) string {
	if len(s) > l {
		s = s[:l]
	}
	return s
}

func pad(s string, l int) string {
	if len(s) < l {
		for i := len(s); i < l; i++ {
			s += " "
		}
	}
	return s
}

func shift(s string, l int) string {
	for i := 0; i < l; i++ {
		s = " " + s
	}
	return s
}

func sfmt(format string, a ...interface{}) string {
	out := fmt.Sprintf(format, a...)
	out = strings.Replace(out, "\n", "", -1)
	return out
}

func (lcd *Lcd) print(line byte, out string) (int, error) {
	out = capstring(out, lcd.cols)
	address, err := lcd.getLCDaddress(line, 0)
	if err != nil {
		return 0, err
	}

	lcd.m.Lock()
	defer lcd.m.Unlock()

	lcd.command(CMD_DDRAM_Set + address) // Do this here to prevent race between print() and SetPosition()
	return lcd.writeBuf([]byte(out))
}

/////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

func NewLcd(c int, r int) (*Lcd, error) {
	i2c, err := os.OpenFile("/dev/i2c-1", os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := ioctl(i2c.Fd(), i2c_SLAVE, uintptr(i2c_ADDR)); err != nil {
		return nil, err
	}
	
	lcd := Lcd{
		i2c:       i2c,
		cols:      c,
		rows:      r,
	}
	// Activate LCD
	var data byte
	data = pinInterpret(INT_D4, data, true)
	data = pinInterpret(INT_D5, data, true)
	lcd.enable(data)
	time.Sleep(200 * time.Millisecond)
	lcd.enable(data)
	time.Sleep(100 * time.Millisecond)
	lcd.enable(data)
	time.Sleep(100 * time.Millisecond)

	// Initialize 4-bit mode
	data = pinInterpret(INT_D4, data, false)
	lcd.enable(data)
	time.Sleep(10 * time.Millisecond)

	lcd.command(CMD_Function_Set | OPT_2_Lines)
	lcd.command(CMD_Display_Control | OPT_Enable_Display | OPT_Enable_Cursor)
	lcd.command(CMD_Clear_Display)
	lcd.command(CMD_Entry_Mode | OPT_Increment | OPT_Display_Shift)

	return &lcd, nil
}

func (lcd *Lcd) BacklightOn() {
	lcd.m.Lock()
	defer lcd.m.Unlock()
	lcd.writeI2C(pinInterpret(INT_BACKLIGHT, 0x00, true))
	lcd.backlight_state = true
}

func (lcd *Lcd) BacklightOff() {
	lcd.m.Lock()
	defer lcd.m.Unlock()
	lcd.writeI2C(pinInterpret(INT_BACKLIGHT, 0x00, false))
	lcd.backlight_state = false
}

func (lcd *Lcd) Clear() {
	lcd.m.Lock()
	defer lcd.m.Unlock()
	lcd.command(CMD_Clear_Display)
}

func (lcd *Lcd) Home() {
	lcd.m.Lock()
	defer lcd.m.Unlock()
	lcd.command(CMD_Return_Home)
}

func (lcd *Lcd) SetPosition(line, pos byte) error {
	address, err := lcd.getLCDaddress(line, pos)
	if err != nil {
		return err
	}

	lcd.m.Lock()
	defer lcd.m.Unlock()
	lcd.command(CMD_DDRAM_Set + address)
	return nil
}

func (lcd *Lcd) ClearLine(line byte) {
	s := pad("", lcd.cols)
	lcd.print(line, s)
}

func (lcd *Lcd) Centerf(line byte, format string, a ...interface{}) (int, error) {
	out := sfmt(format, a...)
	out = shift(out, (lcd.cols - len(out))/2)
	out = pad(out, lcd.cols)
	return lcd.print(line, out)
}

func (lcd *Lcd) RightJustifyf(line byte, format string, a ...interface{}) (int, error) {
	out := sfmt(format, a...)
	out = shift(out, lcd.cols - len(out))
	return lcd.print(line, out)
}

func (lcd *Lcd) Printf(line byte, format string, a ...interface{}) (int, error) {
	out := sfmt(format, a...)
	out = pad(out, lcd.cols)
	return lcd.print(line, out)
}

func setupLCD() {
	var err error
	if !flags.lcd {
		return
	}
	lcd, err = NewLcd(16, 2)
	if err != nil {
		logger.Fatal(err)
	}
	lcd.BacklightOn()
	lcd.Clear()
	lcd.SetPosition(1,1)
	lcd.Centerf(1, "RetroHayes 1.0")
}

func shutdownLCD() {
	if !flags.lcd {
		return
	}
	
	lcd.Clear()
	lcd.BacklightOff()
}
