package main

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

var s *serial.Port

func write(s *serial.Port) {
	var out []byte
	out = make([]byte, 1)
	for {
		out[0] = byte(C.getch())
		if _, err := s.Write(out); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%c", out[0])
	}
}
		
func main() {
	var err error
	
	c := &serial.Config{Name: "/dev/ttyAMA0", Baud: 115200}
	s, err = serial.OpenPort(c)
        if err != nil {
                log.Fatal(err)
        }

	go write(s)
	
	buf := make([]byte, 128)
        for {
		_, err := s.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%c", buf[0])
	}
	
}
