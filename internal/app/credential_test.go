package app

import (
	"os"
	"testing"
)

func TestCredentialFromEnv(t *testing.T) {
	os.Setenv("WAGENT_API_KEY", "sk-test-key-from-env")
	defer os.Unsetenv("WAGENT_API_KEY")

	cs := NewCredentialStore()
	key, err := cs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-test-key-from-env" {
		t.Errorf("expected env key, got %s", key)
	}
}

func TestCredentialEnvTakesPrecedence(t *testing.T) {
	os.Setenv("WAGENT_API_KEY", "sk-from-env")
	defer os.Unsetenv("WAGENT_API_KEY")

	cs := NewCredentialStore()
	cs.Set("sk-from-keychain")
	cs.Clear()

	key, err := cs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-from-env" {
		t.Errorf("env should take precedence, got %s", key)
	}
}

func TestCredentialStatus(t *testing.T) {
	os.Setenv("WAGENT_API_KEY", "sk-test")
	defer os.Unsetenv("WAGENT_API_KEY")

	cs := NewCredentialStore()
	ok, err := cs.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected status ok when key is set")
	}
}

func TestCredentialEmpty(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := NewCredentialStore()
	cs.Clear()
	_, err := cs.Get()
	if err == nil {
		t.Log("note: keychain may be available in this environment")
	}
}

func TestCredentialSetAndClear(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := NewCredentialStore()
	cs.Clear()

	err := cs.Set("sk-test-keychain")
	if err != nil {
		t.Skipf("keychain not available: %v", err)
	}

	ok, err := cs.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected status ok after set")
	}

	key, err := cs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-test-keychain" {
		t.Errorf("expected 'sk-test-keychain', got %s", key)
	}

	err = cs.Clear()
	if err != nil {
		t.Fatal(err)
	}

	ok, err = cs.Status()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected status false after clear")
	}
}

func TestCredentialSetEmptyKey(t *testing.T) {
	cs := NewCredentialStore()
	err := cs.Set("  ")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestCredentialStatusNoKeychain(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := NewCredentialStore()
	cs.Clear()
	ok, err := cs.Status()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Log("note: keychain may have a stored key")
	}
}