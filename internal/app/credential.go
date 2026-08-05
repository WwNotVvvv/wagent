package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const serviceName = "wagent"
const accountName = "api_key"

type CredentialStore struct {
	service string
	account string
}

func NewCredentialStore() *CredentialStore {
	return &CredentialStore{service: serviceName, account: accountName}
}

func newTestCredentialStore() *CredentialStore {
	return &CredentialStore{service: "wagent-test", account: "api_key_test"}
}

func (c *CredentialStore) Get() (string, error) {
	if key := os.Getenv("WAGENT_API_KEY"); key != "" {
		return key, nil
	}
	key, err := keyring.Get(c.service, c.account)
	if err == nil && key != "" {
		return key, nil
	}
	return "", errors.New("API Key not found: set WAGENT_API_KEY or run 'wagent key set'")
}

func (c *CredentialStore) Set(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("cannot set empty API Key")
	}
	return keyring.Set(c.service, c.account, key)
}

func (c *CredentialStore) Status() (bool, error) {
	if os.Getenv("WAGENT_API_KEY") != "" {
		return true, nil
	}
	key, err := keyring.Get(c.service, c.account)
	if err != nil {
		return false, nil
	}
	return key != "", nil
}

func (c *CredentialStore) Clear() error {
	return keyring.Delete(c.service, c.account)
}

func (c *CredentialStore) InteractivePrompt() (string, error) {
	fmt.Print("Enter API Key (input hidden): ")
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	fmt.Println()
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", errors.New("empty key")
	}
	return key, nil
}