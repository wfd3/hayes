package hayes

import (
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"io"
	"net"
	"time"
	"fmt"
)

// Implements connection for in-bound ssh
type sshAcceptReadWriteCloser struct {
	direction int
	mode int
	c io.ReadWriteCloser
	remoteAddr net.Addr
}
func (m sshAcceptReadWriteCloser) Read(p []byte) (int, error) {
	return m.c.Read(p)
}
func (m sshAcceptReadWriteCloser) Write(p []byte) (int, error) {
	return m.c.Write(p)
}
func (m sshAcceptReadWriteCloser) Close() error {
	err := m.c.Close()
	return err
}
func (m sshAcceptReadWriteCloser) Mode() int {
	return m.mode
}
func (m sshAcceptReadWriteCloser) RemoteAddr() net.Addr {
	return m.remoteAddr
}
func (m sshAcceptReadWriteCloser) Direction() int {
	return m.direction
}
func (m sshAcceptReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("bad mode")
	}
	m.mode = mode
}

func (m *Modem) acceptSSH(channel chan connection) {

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
	// TODO: cmdline option!
	private_key := "id_rsa"	
	privateBytes, err := ioutil.ReadFile(private_key)
	if err != nil {
		m.log.Fatalf("Fatal Error: failed to load private key (%s)\n",
			private_key)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		m.log.Fatal("Fatal Error: failed to parse private key")
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be accepted.
	address := "0.0.0.0:" + fmt.Sprintf("%d", *_flags_sshdPort)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		m.log.Fatal("Fatal Error: ", err)
	}
	m.log.Printf("Listening: ssh/%s", address)

	// Accept all connections
	var conn ssh.Channel
	var newChannel ssh.NewChannel
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			m.log.Print("Failed to accept incoming connection (%s)",
				err)
			continue
		}
		// Before use, a handshake must be performed on the
		// incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			m.log.Print("Failed to handshake (%s)", err)
			continue
		}
		go ssh.DiscardRequests(reqs)

		m.log.Printf("New SSH connection from %s (%s)\n",
			sshConn.RemoteAddr(), sshConn.ClientVersion())

		for newChannel = range chans {
			if newChannel.ChannelType() != "session" {
				newChannel.Reject(ssh.UnknownChannelType,
					"unknown channel type")
				continue
			} 

			conn, _, err = newChannel.Accept()
			if err != nil {
				m.log.Fatal("Fatal Error: ", err)
			}

			if m.checkBusy() {
				conn.Write([]byte("Busy..."))
				conn.Close()
				continue
			}
			channel <- sshAcceptReadWriteCloser{INBOUND, DATAMODE,
				conn, sshConn.RemoteAddr()}
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
}
func (m sshDialReadWriteCloser) Read(p []byte) (int, error) {
	return m.in.Read(p)
}
func (m sshDialReadWriteCloser) Write(p []byte) (int, error) {
	return m.out.Write(p)
}
func (m sshDialReadWriteCloser) Close() error {
	// Remember, in is an io.Reader so it doesn't Close()
	err := m.out.Close()
	m.session.Close()
	m.client.Close()
	return err
}
func (m sshDialReadWriteCloser) Direction() int {
	return m.direction
}
func (m sshDialReadWriteCloser) RemoteAddr() net.Addr {
	return m.remoteAddr
}
func (m sshDialReadWriteCloser) Mode() int {
	return m.mode
}
func (m sshDialReadWriteCloser) SetMode(mode int) {
	if mode != DATAMODE || mode != COMMANDMODE {
		panic("bad mode")
	}
	m.mode = mode
}

func (m *Modem) dialSSH(remote string, username string, pw string) (sshDialReadWriteCloser, error) {

	if _, _, err := net.SplitHostPort(remote); err != nil {
		remote += ":22"
	}
	
	m.log.Printf("Connecting to %s [user '%s', pw '%s']", remote, username, pw)

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
		m.log.Print("Fatal Error: ssh.Dial(): ", err)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			m.log.Print("ssh.Dial: Timed out")
		}
		return sshDialReadWriteCloser{},
		fmt.Errorf("ssh.Dial() failed: %s", err)
	}

	// Create a session
	session, err := client.NewSession()
	if err != nil {
    		m.log.Print("unable to create session: ", err)
		return sshDialReadWriteCloser{},
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
    		m.log.Print("request for pseudo terminal failed: ", err)
		return sshDialReadWriteCloser{},
		fmt.Errorf("request for pty failed: ", err)
	}

	// Start remote shell
	send, err :=  session.StdinPipe()
	if err != nil {
		m.log.Print("StdinPipe(): ", err)
		return sshDialReadWriteCloser{},
		fmt.Errorf("session.StdinPipe(): ", err)
	}
	recv, err := session.StdoutPipe()
	if err != nil {
		m.log.Print("StdoutPipe(): ", err)
		return sshDialReadWriteCloser{},
		fmt.Errorf("session.StdinOut(): ", err)
	}

        session.Shell()

	m.log.Printf("Connected to remote host '%s', SSH Server version %s",
		client.Conn.RemoteAddr(), client.Conn.ServerVersion())

	return sshDialReadWriteCloser{OUTBOUND, DATAMODE, recv, send, client,
		session, client.Conn.RemoteAddr()}, nil
}
