package config

import (
	"errors"
	"testing"
)

func TestConfigValidateForRegister(t *testing.T) {
	t.Run("missing_cluster_ip_returns_err_cluster_not_initialized", func(t *testing.T) {
		cfg := &Config{}

		err := cfg.ValidateForRegister()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrClusterNotInitialized) {
			t.Fatalf("expected ErrClusterNotInitialized, got %v", err)
		}
	})

	t.Run("cluster_ip_present_passes", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "203.0.113.10",
		}

		err := cfg.ValidateForRegister()
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func TestConfigValidateForServer(t *testing.T) {
	t.Run("missing_cluster_ip_returns_err_cluster_not_initialized", func(t *testing.T) {
		cfg := &Config{
			Username: "alice",
			Token:    "my-token",
		}

		err := cfg.ValidateForServer()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, ErrClusterNotInitialized) {
			t.Fatalf("expected ErrClusterNotInitialized, got %v", err)
		}
	})

	t.Run("missing_username_fails", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "203.0.113.10",
			Token:     "my-token",
		}

		err := cfg.ValidateForServer()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expected := "validating auth creds: username is required"
		if err.Error() != expected {
			t.Fatalf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("missing_token_fails", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "203.0.113.10",
			Username:  "alice",
		}

		err := cfg.ValidateForServer()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expected := "validating auth creds: token is missing"
		if err.Error() != expected {
			t.Fatalf("error = %q, want %q", err.Error(), expected)
		}
	})

	t.Run("all_required_fields_present_passes", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "203.0.113.10",
			Username:  "alice",
			Token:     "my-token",
		}

		err := cfg.ValidateForServer()
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

func TestConfigEndpointDerivations(t *testing.T) {
	t.Run("ipv4_derivation", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "203.0.113.10",
		}

		got, err := cfg.APIEndpoint()
		if err != nil {
			t.Fatalf("APIEndpoint() error = %v", err)
		}
		want := "https://203.0.113.10:6443"
		if got != want {
			t.Errorf("APIEndpoint() = %q, want %q", got, want)
		}

		got, err = cfg.RegistrationEndpoint()
		if err != nil {
			t.Fatalf("RegistrationEndpoint() error = %v", err)
		}
		want = "http://203.0.113.10:30099"
		if got != want {
			t.Errorf("RegistrationEndpoint() = %q, want %q", got, want)
		}
	})

	t.Run("ipv6_derivation", func(t *testing.T) {
		cfg := &Config{
			ClusterIP: "2001:db8::1",
		}

		got, err := cfg.APIEndpoint()
		if err != nil {
			t.Fatalf("APIEndpoint() error = %v", err)
		}
		want := "https://[2001:db8::1]:6443"
		if got != want {
			t.Errorf("APIEndpoint() = %q, want %q", got, want)
		}

		got, err = cfg.RegistrationEndpoint()
		if err != nil {
			t.Fatalf("RegistrationEndpoint() error = %v", err)
		}
		want = "http://[2001:db8::1]:30099"
		if got != want {
			t.Errorf("RegistrationEndpoint() = %q, want %q", got, want)
		}
	})
}
