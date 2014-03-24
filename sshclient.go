package main

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Modify by linuz.ly
// Modify by gnicod

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
)

type clientPassword string

func (p clientPassword) Password(user string) (string, error) {
	return string(p), nil

}

type TerminalModes map[uint8]uint32

const (
	VINTR         = 1
	VQUIT         = 2
	VERASE        = 3
	VKILL         = 4
	VEOF          = 5
	VEOL          = 6
	VEOL2         = 7
	VSTART        = 8
	VSTOP         = 9
	VSUSP         = 10
	VDSUSP        = 11
	VREPRINT      = 12
	VWERASE       = 13
	VLNEXT        = 14
	VFLUSH        = 15
	VSWTCH        = 16
	VSTATUS       = 17
	VDISCARD      = 18
	IGNPAR        = 30
	PARMRK        = 31
	INPCK         = 32
	ISTRIP        = 33
	INLCR         = 34
	IGNCR         = 35
	ICRNL         = 36
	IUCLC         = 37
	IXON          = 38
	IXANY         = 39
	IXOFF         = 40
	IMAXBEL       = 41
	ISIG          = 50
	ICANON        = 51
	XCASE         = 52
	ECHO          = 53
	ECHOE         = 54
	ECHOK         = 55
	ECHONL        = 56
	NOFLSH        = 57
	TOSTOP        = 58
	IEXTEN        = 59
	ECHOCTL       = 60
	ECHOKE        = 61
	PENDIN        = 62
	OPOST         = 70
	OLCUC         = 71
	ONLCR         = 72
	OCRNL         = 73
	ONOCR         = 74
	ONLRET        = 75
	CS7           = 90
	CS8           = 91
	PARENB        = 92
	PARODD        = 93
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

// keyring implements the ClientKeyring interface

type keyring struct {
	keys []ssh.Signer
}

func (k *keyring) Key(i int) (ssh.PublicKey, error) {
	if i < 0 || i >= len(k.keys) {
		return nil, nil

	}
	return k.keys[i].PublicKey(), nil

}

func (k *keyring) Sign(i int, rand io.Reader, data []byte) (sig []byte, err error) {
	return k.keys[i].Sign(rand, data)

}

func (k *keyring) add(key ssh.Signer) {
	k.keys = append(k.keys, key)

}

func (k *keyring) loadPEM(file string) error {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return err

	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return err

	}
	k.add(key)
	return nil

}

func NewSSHClient(host, user string) (c *SSHClient) {
	c = &SSHClient{
		User: user,
		Host: host,
	}
	return
}

type SSHClient struct {
	User    string
	Host    string
	Pwd     string
	PemFile string
	Agent   net.Conn
	Conn    *ssh.ClientConn
}

func (c *SSHClient) Connect() (e error) {
	var auths []ssh.ClientAuth

	if c.PemFile != "" {
		k := new(keyring)
		err := k.loadPEM(c.PemFile)
		if err != nil {
			log.Printf("cannot load pem file: %s", err)
		}
		//via private key
		auths = append(auths, ssh.ClientAuthKeyring(k))
	}
	//via ssh-agent
	if c.Agent, e = net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); e == nil {
		ac := ssh.NewAgentClient(c.Agent)
		auths = append(auths, ssh.ClientAuthAgent(ac))
	}
	//via pwd
	if c.Pwd == "" {
		auths = append(auths, ssh.ClientAuthPassword(clientPassword(c.Pwd)))
	}

	config := &ssh.ClientConfig{
		User: c.User,
		Auth: auths,
	}
	c.Conn, e = ssh.Dial("tcp", c.Host, config)
	return e
}

func (c *SSHClient) Execute(command string) string {
	defer c.Conn.Close()
	// Create a session
	session, err := c.Conn.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)

	}
	defer session.Close()
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ECHO:          0,     // disable echoing
		TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud

	}
	// Request pseudo terminal
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)

	}

	var b bytes.Buffer
	session.Stdout = &b
	if err := session.Run(command); err != nil {
		session.Stderr = &b
		//return "\033[31mError: '" + command + "' failed to run\033[0m \n"
	}
	return ("\033[31m" + b.String() + "\033[0m")

}
