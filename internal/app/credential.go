package app

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const serviceName = "wagent"
const accountName = "api_key"

type CredentialStore struct{}

func NewCredentialStore() *CredentialStore {
	return &CredentialStore{}
}

func (c *CredentialStore) Get() (string, error) {
	if key := os.Getenv("WAGENT_API_KEY"); key != "" {
		return key, nil
	}
	key, err := keychainGet(serviceName, accountName)
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
	return keychainSet(serviceName, accountName, key)
}

func (c *CredentialStore) Status() (bool, error) {
	if os.Getenv("WAGENT_API_KEY") != "" {
		return true, nil
	}
	key, err := keychainGet(serviceName, accountName)
	if err != nil {
		return false, nil
	}
	return key != "", nil
}

func (c *CredentialStore) Clear() error {
	return keychainDelete(serviceName, accountName)
}

func (c *CredentialStore) InteractivePrompt() (string, error) {
	fmt.Print("Enter API Key (input hidden): ")
	var key string
	n, err := fmt.Scan(&key)
	if err != nil || n == 0 {
		return "", fmt.Errorf("failed to read key: %w", err)
	}
	fmt.Println()
	key = strings.TrimSpace(key)
	if key == "" {
		return "", errors.New("empty key")
	}
	return key, nil
}

func keychainGet(service, account string) (string, error) {
	return "", errors.New("keychain not available in this build")
}

func keychainSet(service, account, key string) error {
	return errors.New("keychain not available in this build")
}

func keychainDelete(service, account string) error {
	return errors.New("keychain not available in this build")
}