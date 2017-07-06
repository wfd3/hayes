package main

import (
	"net"
	"fmt"
)

func main() {
	l, err := net.Listen("tcp", ":30000")
	if err != nil {
		panic(err)
	}
	defer l.Close()

	var b[]byte
	var c byte
	b = make([]byte, 1)
	
	for {
		fmt.Println("Waiting for connection")
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("l.Accept(): %s\n", err)
			continue
		}
		fmt.Printf("Connection accepted from %s\n", conn.RemoteAddr())
				
		// This is a telnet session, negotiate char-at-a-time
		const (
			IAC = 0377
			DO = 0375
			WILL = 0373
			ECHO = 0001
			LINEMODE = 0042
		)
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})

		done := false
		for !done {
			_, err = conn.Read(b)
			if err != nil {
				done = true
				continue
			}
			c = b[0]
			if c == IAC {
				_, err = conn.Read(b)
				_, err = conn.Read(b)
				continue
			}

			if c == 13 {
				fmt.Println()
				continue
			}
			fmt.Printf("%s", string(b))

		}
		conn.Close()
		fmt.Println("Connection closed by remote")
	}
}
