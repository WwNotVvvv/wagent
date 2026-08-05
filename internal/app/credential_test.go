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
	_, err := cs.Get()
	if err == nil {
		t.Log("note: keychain may be available in this environment")
	}
}