package main

import (
	"log"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"flag"
)

var remote string = "127.0.0.1:22"


// Implements io.ReadWriteCloser
type myReadWriter struct {
	in io.Reader
	out io.WriteCloser

}

func (m myReadWriter) Read(p []byte) (int, error) {
	return m.in.Read(p)
}

func (m myReadWriter) Write(p []byte) (int, error) {
	return m.out.Write(p)
}

func (m myReadWriter) Close() error {
	// Remember, in is an io.Reader so it doesn't Close()
	return m.out.Close()
}

func newReadWriteCloser(in io.Reader, out io.WriteCloser) io.ReadWriteCloser {
	var q myReadWriter
	q.in = in
	q.out = out

	return io.ReadWriteCloser(q)
}

func dial() io.ReadWriteCloser {
	config := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*pw),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Danger?
	}

	log.Printf("SSH'inh to %s", remote)
	client, err := ssh.Dial("tcp", remote, config)
	if err != nil {
		log.Fatalf("Dial(): %s", err)
	}
	log.Printf("Made a connection\n")
	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
    		log.Fatal("unable to create session: ", err)
	}
	defer session.Close()

	// Set up terminal modes
	modes := ssh.TerminalModes{
    		ssh.ECHO:          0,     // disable echoing
    		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
    		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}	
	// Request pseudo terminal
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
    		log.Fatal("request for pseudo terminal failed: ", err)
	}
	// Start remote shell

	log.Print("Getting stdin")
	send, err :=  session.StdinPipe()
	if err != nil {
		log.Fatal("StdinPipe(): ", err)
	}
	recv, err := session.StdoutPipe()
	if err != nil {
		log.Fatal("StdoutPipe(): ", err)
	}
	log.Print("Creating io.ReadWriteCloser")
	f := newReadWriteCloser(recv, send)

	log.Print("Starting shell")	
        session.Shell()

	log.Print("Returning")

	return f
}


var user = flag.String("u", "", "username")
var pw = flag.String("p", "", "password")
func main() {
	flag.Parse()
	if *user == "" {
		log.Fatal("No username")
	}
	if *pw == "" {
		log.Fatal("No password")
	}

	f := dial()
	
	log.Print("Starting copies")
	go io.Copy(os.Stdin, f)
	// io.Copy(f, os.Stdout)
	var b []byte
	b = make([]byte, 1)
	for {
		_, err := f.Read(b)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("f.Read(): ", err)
		}
		os.Stdout.Write(b)
	}

	log.Print("Done")


}
