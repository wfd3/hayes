package main

import (
	"fmt"
	"log"
	"golang.org/x/crypto/ssh"
	"time"
	"io/ioutil"
)

var remote string = "127.0.0.1:22"

func main() {
//	var hostKey ssh.PublicKey
	config := &ssh.ClientConfig{
		User: "wd",
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		//		HostKeyCallback: ssh.FixedHostKey(hostKey)
		HostKeyCallback: ssh.InsecureHostKey(), // Danger?
	}

	log.Printf("SSH'inh to %s", remote)
	client, err := ssh.Dial("tcp", remote, config)
	if err != nil {
		log.Fatalf("Dial(): %s", err)
	}

	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("NewSession(): %s", err)
	}

	for i := 0; i < 100; i++ {
		fmt.Fprintf(session.Stdout, "Hello!\n")
		time.Sleep(250 * time.Millisecond)
	}
	session.Close()
}
