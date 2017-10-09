# hayes
Turn a Raspberry Pi 3, some associated hardware bits, and some Go into a Hayes Smartmodem

A Work in Progress

To Build:
go build modem

Extensions to the Hayes command set:

To dial a host with "telnet": ATDH hostname[:port]

To dial a host with SSH: ATDE hostname[:port]|username|password

The file 'phonebook.json' also allows phone number to <host, port, protocol, ... > mapping so ATD commands work as expected.

Parts needed:

 . Raspberry Pi 3
 
 . 3 MAX3232's
 
 . DB9
 
 . Some jumpers
 
 . Some LEDs and resistor

This really needs some documentation...

  
