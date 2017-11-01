package main

import (
	"net"
	"fmt"
)


// This is a telnet session, negotiate char-at-a-time
const (
	IAC = 0377
	DO = 0375
	WILL = 0373
	ECHO = 0001
	LINEMODE = 0042
)

const PORT = ":30000"

func main() {
	l, err := net.Listen("tcp", PORT)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	fmt.Printf("Echo server at port %s\n", PORT)

	b := make([]byte, 1)
	for {
		fmt.Println("Waiting for connection")
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("Accept(): %s\n", err)
			continue
		}
		fmt.Printf("Connection accepted from %s\n", conn.RemoteAddr())
		conn.Write([]byte{IAC, DO, LINEMODE, IAC, WILL, ECHO})

		for {
			_, err := conn.Read(b)
			if err != nil {
				fmt.Printf("Read(): %s", err)
				break
			}
			if b[0] == IAC {
				for i :=0; i < 3; i++ {
					if _, err := conn.Read(b); err != nil {
						fmt.Printf("Read(): %s", err)
						break
					}
				}
			}
			
			if b[0]== 13 {
				fmt.Println()
				continue
			}

			fmt.Printf("%s", string(b))
			if _, err :=conn.Write(b); err != nil {
				fmt.Printf("Write(): %s", err)
				break
			}
			
			if b[0] == '*' {
				s := "1234567890"
				i, err :=conn.Write([]byte(s))
				fmt.Printf("sent %d (%s)\n", i, err)
			}

			if b[0] == '!' {
				conn.Write([]byte("\nIAC TEST START\n"))
				conn.Write([]byte{IAC, IAC, 'A', 'B', 'C'})
				conn.Write([]byte("\nIAC TEST END\n"))
			}
		}
			
		conn.Close()
		fmt.Println("\nConnection closed by remote")
	}
}
