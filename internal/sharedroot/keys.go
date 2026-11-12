package sharedroot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"
)

// Automation key pair file names under credentials/automation/ (ADR-0005
// section 6).
const (
	AutomationDir        = "automation"
	AutomationPrivateKey = "ssh_ed25519"
	AutomationPublicKey  = "ssh_ed25519.pub"
)

// maxKeyInput bounds the private key read from standard input.
const maxKeyInput = 64 << 10

// automationComment is the key comment; it names the environment.
func automationComment(env string) string {
	return "assembleiarr-" + env + "-automation"
}

// KeyPair is an OpenSSH private key and its authorized_keys public line.
type KeyPair struct {
	Private []byte
	Public  []byte
}

// Files returns the pair as the files of credentials/automation/.
func (k KeyPair) Files() []File {
	return []File{
		{Name: AutomationPrivateKey, Content: k.Private, Mode: 0o600},
		{Name: AutomationPublicKey, Content: k.Public, Mode: 0o644},
	}
}

// GenerateAutomationKey creates a new ed25519 key pair without a passphrase,
// in OpenSSH format, whose comment names env.
func GenerateAutomationKey(env string) (KeyPair, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return KeyPair{}, fmt.Errorf("generate automation key: %w", err)
	}
	block, err := ssh.MarshalPrivateKey(priv, automationComment(env))
	if err != nil {
		return KeyPair{}, fmt.Errorf("encode automation key: %w", err)
	}
	public, err := publicLine(pub, env)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{Private: pem.EncodeToMemory(block), Public: public}, nil
}

// ImportAutomationKey reads an OpenSSH ed25519 private key without a
// passphrase from r, keeps it as given and derives its public key. Errors
// never hold key material.
func ImportAutomationKey(r io.Reader, env string) (KeyPair, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxKeyInput+1))
	if err != nil {
		return KeyPair{}, fmt.Errorf("read the private key from standard input: %w", err)
	}
	if len(data) > maxKeyInput {
		return KeyPair{}, Usagef("the private key on standard input is larger than %d bytes", maxKeyInput)
	}
	// pem.Decode skips any text before a block, so the input must start with
	// the block (only whitespace before it) and hold nothing after it.
	starts := bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("-----BEGIN OPENSSH PRIVATE KEY-----"))
	block, rest := pem.Decode(data)
	if !starts || block == nil || block.Type != "OPENSSH PRIVATE KEY" || len(bytes.TrimSpace(rest)) != 0 {
		return KeyPair{}, Usagef("standard input must hold exactly one OpenSSH private key (BEGIN OPENSSH PRIVATE KEY)")
	}
	key, err := ssh.ParseRawPrivateKey(data)
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return KeyPair{}, Usagef("the automation private key must not have a passphrase")
		}
		return KeyPair{}, Usagef("standard input does not hold a valid OpenSSH private key")
	}
	var pub ed25519.PublicKey
	switch k := key.(type) {
	case ed25519.PrivateKey:
		pub = k.Public().(ed25519.PublicKey)
	case *ed25519.PrivateKey:
		pub = k.Public().(ed25519.PublicKey)
	default:
		return KeyPair{}, Usagef("the automation private key must be an ed25519 key")
	}
	public, err := publicLine(pub, env)
	if err != nil {
		return KeyPair{}, err
	}
	// The private key is kept exactly as given, so every controller of the
	// environment holds the same file.
	return KeyPair{Private: data, Public: public}, nil
}

func publicLine(pub ed25519.PublicKey, env string) ([]byte, error) {
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("encode automation public key: %w", err)
	}
	line := bytes.TrimRight(ssh.MarshalAuthorizedKey(sshPub), "\n")
	return append(append(line, ' '), automationComment(env)+"\n"...), nil
}

// checkAutomationPair checks that private is an OpenSSH ed25519 private key
// without a passphrase whose public key is the one in public. It returns the
// SHA256 fingerprint of public (when it parses) and a problem message, or "".
// Neither ever holds key material.
func checkAutomationPair(private, public []byte) (string, string) {
	pub, _, _, rest, err := ssh.ParseAuthorizedKey(public)
	if err != nil || len(bytes.TrimSpace(rest)) != 0 {
		return "", AutomationPublicKey + " is not one OpenSSH public key line"
	}
	fingerprint := ssh.FingerprintSHA256(pub)
	key, err := ssh.ParseRawPrivateKey(private)
	if err != nil {
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return fingerprint, AutomationPrivateKey + " has a passphrase; the automation key must not have one"
		}
		return fingerprint, AutomationPrivateKey + " is not a valid OpenSSH private key"
	}
	var derived ed25519.PublicKey
	switch k := key.(type) {
	case ed25519.PrivateKey:
		derived = k.Public().(ed25519.PublicKey)
	case *ed25519.PrivateKey:
		derived = k.Public().(ed25519.PublicKey)
	default:
		return fingerprint, AutomationPrivateKey + " is not an ed25519 key"
	}
	sshDerived, err := ssh.NewPublicKey(derived)
	if err != nil {
		return fingerprint, AutomationPrivateKey + " has an invalid ed25519 public part"
	}
	if !bytes.Equal(sshDerived.Marshal(), pub.Marshal()) {
		return fingerprint, AutomationPrivateKey + " does not match " + AutomationPublicKey
	}
	return fingerprint, ""
}
