package hayes

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"io"
	"net"
	"time"
	"fmt"
	"log"
)

// Implements connection for in-bound ssh
type sshAcceptReadWriteCloser struct {
	direction int
	mode int
	c io.ReadWriteCloser
	remoteAddr net.Addr
	sent uint64
	recv uint64
}
func (m *sshAcceptReadWriteCloser) Read(p []byte) (int, error) {
	i, err :=  m.c.Read(p)
	m.recv += uint64(i)
	return i, err
}
func (m *sshAcceptReadWriteCloser) Write(p []byte) (int, error) {
	i, err := m.c.Write(p)
	m.sent += uint64(i)
	return i, err
}
func (m *sshAcceptReadWriteCloser) Close() error {
	err := m.c.Close()
	return err
}
func (m *sshAcceptReadWriteCloser) Mode() int {
	return m.mode
}
func (m *sshAcceptReadWriteCloser) RemoteAddr() net.Addr {
	return m.remoteAddr
}
func (m *sshAcceptReadWriteCloser) Direction() int {
	return m.direction
}
func (m *sshAcceptReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("bad mode")
	}
	m.mode = mode
}
func(m *sshAcceptReadWriteCloser) Stats() (uint64, uint64) {
	return m.sent, m.recv
}

func acceptSSH(channel chan connection, private_key string, busy busyFunc,
	log *log.Logger, ok chan error) {

	// In the latest version of crypto/ssh (after Go 1.3), the SSH
	// server type has been removed in favour of an SSH connection
	// type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in
	// effect, upgrading the net.Conn into an ssh.ServerConn

	config := &ssh.ServerConfig{
		// You may also explicitly allow anonymous client
		// authentication, though anon bash sessions may not
		// be a wise idea
		NoClientAuth: true,
	}

	// You can generate a keypair with 'ssh-keygen -t rsa'
	log.Printf("Loading SSH private key from %s", private_key)
	privateBytes, err := ioutil.ReadFile(private_key)
	if err != nil {
		log.Printf("Fatal Error: failed to load private key (%s): %s\n",
			private_key, err)
		ok <- err
		return
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Print("Fatal Error: failed to parse private key: ", err)
		ok <- err
		return
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	address := "0.0.0.0:" + fmt.Sprintf("%d", *_flags_sshdPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Print("Fatal Error: ", err)
		ok <- err
		return
	}
	log.Printf("Listening: ssh/%s", address)

	// Accept all connections
	var conn ssh.Channel
	var newChannel ssh.NewChannel
	ok <- nil
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Print("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the
		// incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Print("Failed to handshake (%s)", err)
			continue
		}
		go ssh.DiscardRequests(reqs)

		log.Printf("New SSH connection from %s (%s)\n",
			sshConn.RemoteAddr(), sshConn.ClientVersion())

		for newChannel = range chans {
			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType,
					"unknown channel type")
				continue
			} 

			conn, _, err = newChannel.Accept()
			if err != nil {
				log.Fatal("Fatal Error: ", err)
			}

			if busy() {
				conn.Write([]byte("Busy...\n\r"))
				conn.Close()
				continue
			}
			channel <- &sshAcceptReadWriteCloser{INBOUND, DATAMODE,
				conn, sshConn.RemoteAddr(), 0, 0}
			break
		}
	}
}

// Implements connection, used to convert SSH ssh.Session for outbound SSH 
type sshDialReadWriteCloser struct {
	direction int
	mode int
	in io.Reader
	out io.WriteCloser
	client *ssh.Client
	session *ssh.Session
	remoteAddr net.Addr
	sent uint64
	recv uint64
}
func (m *sshDialReadWriteCloser) Read(p []byte) (int, error) {
	i, err := m.in.Read(p)
	m.recv += uint64(i)
	return i, err
}
func (m *sshDialReadWriteCloser) Write(p []byte) (int, error) {
	i, err := m.out.Write(p)
	m.sent += uint64(i)
	return i, err
}
func (m *sshDialReadWriteCloser) Close() error {
	// Remember, in is an io.Reader so it doesn't Close()
	err := m.out.Close()
	m.session.Close()
	m.client.Close()
	return err
}
func (m *sshDialReadWriteCloser) Direction() int {
	return m.direction
}
func (m *sshDialReadWriteCloser) RemoteAddr() net.Addr {
	return m.remoteAddr
}
func (m *sshDialReadWriteCloser) Mode() int {
	return m.mode
}
func (m *sshDialReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("bad mode")
	}
	m.mode = mode
}
func (m *sshDialReadWriteCloser) Stats() (uint64, uint64) {
	return m.sent, m.recv
}

func dialSSH(remote string, log *log.Logger, username string, pw string) (*sshDialReadWriteCloser, error) {

	if _, _, err := net.SplitHostPort(remote); err != nil {
		remote += ":22"
	}
	
	log.Printf("Connecting to %s [user '%s', pw '%s']", remote, username, pw)

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(pw),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Danger?
		Timeout: time.Duration(__CONNECT_TIMEOUT),
	}

	client, err := ssh.Dial("tcp", remote, config)
	if err != nil {
		log.Print("Fatal Error: ssh.Dial(): ", err)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Print("ssh.Dial: Timed out")
		}
		return &sshDialReadWriteCloser{},
		fmt.Errorf("ssh.Dial() failed: %s", err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
    		log.Print("unable to create session: ", err)
		return &sshDialReadWriteCloser{},
		fmt.Errorf("unable to create session: ", err)
	}

	// Set up terminal modes
	modes := ssh.TerminalModes{
    		ssh.ECHO:          0,     // disable echoing
    		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
    		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}	
	// Request pseudo terminal
	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
    		log.Print("request for pseudo terminal failed: ", err)
		return &sshDialReadWriteCloser{},
		fmt.Errorf("request for pty failed: ", err)
	}

	// Start remote shell
	send, err :=  session.StdinPipe()
	if err != nil {
		log.Print("StdinPipe(): ", err)
		return &sshDialReadWriteCloser{},
		fmt.Errorf("session.StdinPipe(): ", err)
	}
	recv, err := session.StdoutPipe()
	if err != nil {
		log.Print("StdoutPipe(): ", err)
		return &sshDialReadWriteCloser{},
		fmt.Errorf("session.StdinOut(): ", err)
	}

        session.Shell()

	log.Printf("Connected to remote host '%s', SSH Server version %s",
		client.Conn.RemoteAddr(), client.Conn.ServerVersion())

	return &sshDialReadWriteCloser{OUTBOUND, DATAMODE, recv, send, client,
		session, client.Conn.RemoteAddr(), 0, 0}, nil
}
