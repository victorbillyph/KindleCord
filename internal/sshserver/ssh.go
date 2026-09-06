package sshserver

import (
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

// Start starts a debug SSH server on :2222 with user kindlecord / password kindle
// Returns the listener or error. Shell is /bin/sh.
func Start(port int) (net.Listener, error) {
	if port == 0 {
		port = 2222
	}
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "kindlecord" && string(pass) == "kindle" {
				return nil, nil
			}
			if c.User() == "root" && string(pass) == "kindle" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		NoClientAuth: false,
	}
	// Generate host key in memory
	priv, err := generateKey()
	if err != nil {
		return nil, err
	}
	config.AddHostKey(priv)

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	log.Printf("[SSH] listening on %s (user: kindlecord / kindle)", addr)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("[SSH] accept error: %v", err)
				continue
			}
			go handleConn(conn, config)
		}
	}()
	return ln, nil
}

func handleConn(nconn net.Conn, config *ssh.ServerConfig) {
	defer nconn.Close()
	sshConn, chans, reqs, err := ssh.NewServerConn(nconn, config)
	if err != nil {
		log.Printf("[SSH] handshake failed: %v", err)
		return
	}
	defer sshConn.Close()
	log.Printf("[SSH] new connection from %s user %s", sshConn.RemoteAddr(), sshConn.User())
	go ssh.DiscardRequests(reqs)
	for ch := range chans {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.UnknownChannelType, "unknown")
			continue
		}
		channel, requests, err := ch.Accept()
		if err != nil {
			continue
		}
		go handleChannel(channel, requests)
	}
}

func handleChannel(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	var cmd *exec.Cmd
	var stdin io.WriteCloser
	var stdout, stderr io.ReadCloser
	// Handle requests
	for req := range reqs {
		switch req.Type {
		case "exec":
			var payload = struct{ Command string }{}
			if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
				req.Reply(false, nil)
				continue
			}
			req.Reply(true, nil)
			cmd = exec.Command("/bin/sh", "-c", payload.Command)
			stdin, _ = cmd.StdinPipe()
			stdout, _ = cmd.StdoutPipe()
			stderr, _ = cmd.StderrPipe()
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(ch, "failed: %v\r\n", err)
				return
			}
			go io.Copy(stdin, ch)
			go func() {
				io.Copy(ch, stdout)
				ch.CloseWrite()
			}()
			go io.Copy(ch.Stderr(), stderr)
			waitErr := cmd.Wait()
			exitStatus := 0
			if waitErr != nil {
				if exitErr, ok := waitErr.(*exec.ExitError); ok {
					exitStatus = exitErr.ExitCode()
				} else {
					exitStatus = 1
				}
			}
			ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(exitStatus)}))
			return
		case "shell":
			req.Reply(true, nil)
			cmd = exec.Command("/bin/sh")
			cmd.Env = []string{"TERM=xterm"}
			stdin, _ = cmd.StdinPipe()
			stdout, _ = cmd.StdoutPipe()
			stderr, _ = cmd.StderrPipe()
			if err := cmd.Start(); err != nil {
				fmt.Fprintf(ch, "failed to start shell: %v\r\n", err)
				return
			}
			go io.Copy(stdin, ch)
			go io.Copy(ch, stdout)
			go io.Copy(ch.Stderr(), stderr)
		case "pty-req":
			req.Reply(true, nil)
		case "window-change":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
	if cmd != nil {
		cmd.Wait()
	}
}
