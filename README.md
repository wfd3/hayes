# hayes
Turn a Raspberry Pi 3, some associated hardware bits, and some Go into a Hayes Smartmodem
```
Command line options:
  -addressbook file
    	Address Book file (default "./addressbook.json")
  -keyfile file
    	SSH Private Key file (default "./id_rsa")
  -logfile file
    	Default log file (default stderr)
  -nossh
    	Don't start SSH server (default false)
  -notelnet
    	Don't start telnet server (default false)
  -serial device
    	Serial device (eg, /dev/ttyS0)
  -speed speed
    	Serial Port speed (bps) between DTE and DCE (default 115200)
  -sshport port
    	Network port number for inbound sshd sessions (default 22000)
  -syslog
    	Log to syslog (default false)
  -telnetport port
    	Network port number for inbound telnet sessions (default 20000)
```

Modem commands supported:
* ATA - Answer
* ATD - Dial
*	ATE - Command state echo
*	ATH - Hook command 
*	ATI - Internal tests
*	ATL - Speaker volume
*	ATM - Speaker on/off
*	ATO - On-line command
*	ATQ - Result code options
*	ATS - Register commands 
*	ATV - Result code fomat
*	ATW - Negoitation Progress message selection
*	ATX - Call progress options
*	ATZ - Soft reset
*	AT&C - Carrier Data Detect (CDC) options
*	AT&D - Data Terminal Read (DTR) options
*	AT&F - Recall factory profile (factory reset)
*	AT&S - Data Set Ready (DSR) options
*	AT&V - View Configuration Profiles
*	AT&W - Write active profile to memory
*	AT&Y - Select stored profile for hard reset
*	AT&Z - Store telephone number

Modem Command Extensions:
*	AT! - Display network status 
*	AT* - Dump internal state
* ATDH*host:port* - Dial *host:port*
* ATDE*host:port|username|password* - Dial *host:port|username|password* using an SSH tunnel
* AT&Z*n*=D - Delete phone book entry *n*
   * NOTE: The addressbook configuration file allows phone number:<host, port, protocol, ... > mapping to enables traditional number based dialing.

 
"Faked" Modem Commands (perform no action but return OK):
* ATB
* ATC
* ATF
* ATN
* ATP
* ATT
* ATY
* AT&A
* AT&B
* AT&G
* AT&J
* AT&K
* AT&L
* AT&M
* AT&O
* AT&Q
* AT&R
* AT&T
* AT&U
* AT&X

RS232 compliance:
* SD/TX, RD/RX, DSR, DTR, RI, DCD pins are supported.
* RTS/CTS flow control is not (AT&K0 is set), alhough the pins are active.

Parts needed:

* Raspberry Pi 3
* MAX3232 IC's (3)
* DB9
* Some jumpers
* Some LEDs and resistors

The docs/ directory has some basic pin mappings and a crude Fritzing diagram (https://github.com/wfd3/hayes/blob/master/docs/Modem%201.fzz).  

  
