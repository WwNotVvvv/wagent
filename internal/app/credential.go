package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS == "windows" {
		return keychainGetWindows(service, account)
	}
	return keychainGetFile(service, account)
}

func keychainSet(service, account, key string) error {
	if runtime.GOOS == "windows" {
		return keychainSetWindows(service, account, key)
	}
	return keychainSetFile(service, account, key)
}

func keychainDelete(service, account string) error {
	if runtime.GOOS == "windows" {
		return keychainDeleteWindows(service, account)
	}
	return keychainDeleteFile(service, account)
}

func keychainGetWindows(service, account string) (string, error) {
	keyPath := keychainFilePath(service, account)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func keychainSetWindows(service, account, key string) error {
	keyPath := keychainFilePath(service, account)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("create keychain dir: %w", err)
	}
	return os.WriteFile(keyPath, []byte(key), 0600)
}

func keychainDeleteWindows(service, account string) error {
	keyPath := keychainFilePath(service, account)
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func keychainFilePath(service, account string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".wagent", "keys", service+"_"+account)
}

func keychainGetFile(service, account string) (string, error) {
	keyPath := keychainFilePath(service, account)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func keychainSetFile(service, account, key string) error {
	keyPath := keychainFilePath(service, account)
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return fmt.Errorf("create keychain dir: %w", err)
	}
	return os.WriteFile(keyPath, []byte(key), 0600)
}

func keychainDeleteFile(service, account string) error {
	keyPath := keychainFilePath(service, account)
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}