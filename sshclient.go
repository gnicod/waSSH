package main

import (
	"code.google.com/p/go.crypto/ssh"
	"net"
	//"fmt"
	"bytes"
	"io/ioutil"
	"log"
)

const (
	ECHO          = 53
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

type SSHClient struct {
	User    string
	Host    string
	Pwd     string
	Key     string
	Agent   net.Conn
	Session *ssh.Session
	Config  *ssh.ClientConfig
}

func parsekey(file string) ssh.Signer {
	privateBytes, err := ioutil.ReadFile(file)
	if err != nil {
		panic("Failed to load private key")

	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		panic("Failed to parse private key")

	}
	return private

}

func NewSSHClient(host string, user string, keyPath string) (c *SSHClient) {

	pkey := parsekey(keyPath)

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(pkey),
			ssh.Password(""),
		},
	}
	c = &SSHClient{
		User:   user,
		Host:   host,
		Config: config,
	}
	return
}

func (c *SSHClient) Output(s string) (stout []byte, sterr error) {
	client, err := ssh.Dial("tcp", c.Host, c.Config)
	if err != nil {
		panic("Failed to dial: " + err.Error())

	}

	c.Session, err = client.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())

	}
	defer c.Session.Close()
	stout, sterr = c.Session.Output(s)
	return stout, sterr

}

func (c *SSHClient) Run(command string) (out string, err error) {
	client, err := ssh.Dial("tcp", c.Host, c.Config)
	if err != nil {
		panic("Failed to dial: " + err.Error())

	}
	// Create a session
	c.Session, err = client.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)

	}
	defer c.Session.Close()
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ECHO:          0,     // disable echoing
		TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud

	}
	// Request pseudo terminal
	if err := c.Session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)

	}

	var b bytes.Buffer
	c.Session.Stdout = &b
	if err := c.Session.Run(command); err != nil {
		c.Session.Stderr = &b
	}
	return b.String(), err

}

func (c *SSHClient) Close(session *ssh.Session) {
	c.Session.SendRequest("close", false, nil)

}
