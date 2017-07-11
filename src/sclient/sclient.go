package main

import (
	"log"
	"golang.org/x/crypto/ssh"
)

var remote string = "127.0.0.1:22"

func main() {
	config := &ssh.ClientConfig{
		User: "walt",
		Auth: []ssh.AuthMethod{
			ssh.Password("ks120872"),
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

	channel, _, err := client.Conn.OpenChannel("session", nil)
	if err != nil {
		log.Fatal("OpenChannel", err)
	}
	channel.Write([]byte("Hello!"))


/*
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

	send, err :=  session.StdinPipe()
	if err != nil {
		log.Fatal("StdoutPipe(): ", err)
	}
	send.Write([]byte("Hello!"))
*/
/*
	if err := session.Shell(); err != nil {
    		log.Fatal("failed to start shell: ", err)
	}	
	log.Printf("Shell() returned\n")
*/
}
