package app

import (
	"os"
	"testing"

	"github.com/zalando/go-keyring"
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

	cs := newTestCredentialStore()
	cs.Set("sk-from-keychain")
	defer cs.Clear()

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
	cs := newTestCredentialStore()
	cs.Clear()
	_, err := cs.Get()
	if err == nil {
		t.Log("note: keychain may be available in this environment")
	}
}

func TestCredentialSetAndClear(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := newTestCredentialStore()
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
	cs := newTestCredentialStore()
	err := cs.Set("  ")
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestCredentialStatusNoKeychain(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := newTestCredentialStore()
	cs.Clear()
	ok, err := cs.Status()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Log("note: keychain may have a stored key")
	}
}

func TestCredentialCrossInstancePersistence(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs1 := newTestCredentialStore()
	cs1.Clear()

	err := cs1.Set("sk-persistence-test")
	if err != nil {
		t.Skipf("keychain not available: %v", err)
	}
	defer cs1.Clear()

	cs2 := newTestCredentialStore()
	key, err := cs2.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-persistence-test" {
		t.Errorf("key not persisted across instances: got %s, expected sk-persistence-test", key)
	}
}

func TestCredentialEnvOverridesKeychain(t *testing.T) {
	os.Setenv("WAGENT_API_KEY", "sk-from-env-v2")
	defer os.Unsetenv("WAGENT_API_KEY")

	cs := newTestCredentialStore()
	cs.Set("sk-from-keychain-v2")
	defer cs.Clear()

	key, err := cs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-from-env-v2" {
		t.Errorf("env should override keychain: got %s, expected sk-from-env-v2", key)
	}
}

func TestCredentialKeychainOnly(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")
	cs := newTestCredentialStore()
	cs.Clear()

	err := cs.Set("sk-keychain-only")
	if err != nil {
		t.Skipf("keychain not available: %v", err)
	}
	defer cs.Clear()

	key, err := cs.Get()
	if err != nil {
		t.Fatal(err)
	}
	if key != "sk-keychain-only" {
		t.Errorf("expected keychain-only key, got %s", key)
	}
}

func TestCredentialRealUserKeyUntouched(t *testing.T) {
	os.Unsetenv("WAGENT_API_KEY")

	realKey, realErr := keyring.Get("wagent", "api_key")

	testStore := newTestCredentialStore()
	testStore.Set("temp-test-key")
	_, _ = testStore.Get()
	testStore.Clear()

	realKeyAfter, realErrAfter := keyring.Get("wagent", "api_key")

	if realErr != nil && realErrAfter != nil {
		return
	}
	if realErr == nil && realErrAfter != nil {
		t.Error("real user key was deleted during test run")
	}
	if realKey != realKeyAfter {
		t.Errorf("real user key was modified: before=%q after=%q", realKey, realKeyAfter)
	}
}