package server

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/argon2"
)

// Hashed passwords are stored on disk.
const (
	authPath string = ".gofit/auth"
)

func initAuth() {
	_, err := os.Stat(authPath)
	if err != nil {
		err := os.MkdirAll(authPath, 0700)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func hashAndStore(creds Credentials) error {
	if authExists(creds) {
		return fmt.Errorf("Username %s is taken", creds.Username)
	}
	key := argon2.IDKey([]byte(creds.Password), []byte(creds.Username), 1, 46*1024, 1, 32)
	if err := os.WriteFile(filepath.Join(authPath, creds.Username), key, 0600); err != nil {
		return err
	}
	return nil
}

func auth(creds Credentials) error {
	if !authExists(creds) {
		return fmt.Errorf("Username %s does not exist", creds.Username)
	}
	key := argon2.IDKey([]byte(creds.Password), []byte(creds.Username), 1, 46*1024, 1, 32)
	storedKey, err := os.ReadFile(filepath.Join(authPath, creds.Username))
	if err != nil {
		return err
	}
	if bytes.Equal(key, storedKey) {
		return nil
	}
	return fmt.Errorf("Login failed for %s", creds.Username)
}

// authExists checks if a username is already taken
func authExists(creds Credentials) bool {
	_, err := os.Stat(filepath.Join(authPath, creds.Username))
	return err == nil
}
