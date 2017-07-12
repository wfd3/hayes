package main

import (
	"log"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
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
	stderr, err := session.StderrPipe()
	if err != nil {
		log.Fatal("StderrPipe(): ", err)
	}
	log.Print("Starting shell")	
        session.Shell()
	log.Print("Starting Copy functions")
	// my stdout -> send
	// my stdin <- recv
	// my stdin <- stderr
	go io.Copy(os.Stdin, recv)
	go io.Copy(os.Stderr, stderr)
	io.Copy(send, os.Stdout)
	log.Print("Wait()'ing")
	session.Wait()
	log.Print("Done")
}
