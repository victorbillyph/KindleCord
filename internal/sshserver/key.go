package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"golang.org/x/crypto/ssh"
)

func generateKey() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(priv)
}
