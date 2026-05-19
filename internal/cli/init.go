package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/spf13/cobra"
)

var initIP string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize cluster connection settings",
	Long:  "I'll think of this later",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateIP(initIP); err != nil {
			return err
		}

		return buildInitConfig(initIP)
	},
}

func buildInitConfig(ip string) error {
	var cfg *config.Config

	// Check config exists
	exists, err := config.CheckConfigExists()
	if err != nil {
		return fmt.Errorf("error checking config exists: %w", err)
	}

	if !exists {
		cfg = &config.Config{}
	} else {
		cfg, err = config.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}
	}

	// Build cluster API endpoint and probe endpoint for TLS status
	cfg.ClusterIP = ip

	apiEndpoint, err := cfg.APIEndpoint()
	if err != nil {
		return fmt.Errorf("error building api endpoint: %w", err)
	}

	tlsInsecure, fellBack, err := probeClusterAPIEndpoint(apiEndpoint)
	if err != nil {
		return fmt.Errorf("error probing api endpoint: %w", err)
	}
	cfg.TLSInsecure = tlsInsecure

	if fellBack {
		fmt.Fprintln(os.Stderr, "Warning: TLS certificate verification failed. Falling back to insecure mode.")
		fmt.Fprintln(os.Stderr, "This is persisted in ~/.kubecraft/config. To re-verify, delete config and re-run init")
	}

	// Probe registration endpoint
	registrationEndpoint, err := cfg.RegistrationEndpoint()
	if err != nil {
		return fmt.Errorf("error building registration endpoint: %w", err)
	}

	if err := probeRegistrationEndpoint(registrationEndpoint); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Successfully initialized cluster config. Configuration saved to ~/.kubecraft/config\n\nNext step: kubecraft register --username <name>\n")
	return nil
}

// probeClusterAPIEndpoint checks Kubernetes API reachability.
//
// Returns:
// - tlsInsecure: whether init should persist tlsInsecure=true
// - fellBack: whether strict TLS failed and insecure fallback was used
// - err: non-nil when API is unreachable or probe cannot complete
func probeClusterAPIEndpoint(apiURL string) (tlsInsecure bool, fellBack bool, err error) {
	// 1) Strict TLS probe
	if err := tryAPIRequest(apiURL, false); err == nil {
		return false, false, nil
	} else {
		// 2) Only fallback for certificate-verification failures
		if !isTLSVerificationError(err) {
			return false, false, fmt.Errorf("%s: %w", apiUnreachableMessage(apiURL), err)
		}
	}

	// 3) Insecure fallback probe
	if err := tryAPIRequest(apiURL, true); err != nil {
		return false, false, fmt.Errorf("%s: %w", apiUnreachableMessage(apiURL), err)
	}

	return true, true, nil
}

func probeRegistrationEndpoint(regURL string) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, regURL+"/register", nil)
	if err != nil {
		return fmt.Errorf("failed to build registration probe request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach registration endpoint at %s: %w", regURL, err)
	}
	defer resp.Body.Close()

	return nil
}

func tryAPIRequest(apiURL string, insecure bool) error {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	// /version is a lightweight endpoint on kube-apiserver
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(apiURL, "/")+"/version", nil)
	if err != nil {
		return fmt.Errorf("building api probe request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Any HTTP response means network/TLS path is reachable.
	// We only care about connectivity/probe here, not auth.
	return nil
}

func isTLSVerificationError(err error) bool {
	// unwrap url.Error if present
	var uerr *url.Error
	if errors.As(err, &uerr) && uerr.Err != nil {
		err = uerr.Err
	}

	var unknownAuthErr x509.UnknownAuthorityError
	if errors.As(err, &unknownAuthErr) {
		return true
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return true
	}

	var certInvalidErr x509.CertificateInvalidError
	if errors.As(err, &certInvalidErr) {
		return true
	}

	// Some TLS verification failures are surfaced as string-wrapped errors.
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "certificate") &&
		(strings.Contains(msg, "unknown authority") ||
			strings.Contains(msg, "cannot validate certificate") ||
			strings.Contains(msg, "is not valid for")) {
		return true
	}

	return false
}

func apiUnreachableMessage(apiURL string) string {
	return fmt.Sprintf(
		"cannot reach Kubernetes API at %s\nCheck that port 6443 is open in your security group / firewall",
		apiURL,
	)
}

func validateIP(ip string) error {
	if ip == "" {
		return fmt.Errorf("ip address is required")
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address: %q", ip)
	}

	return nil
}

func init() {
	initCmd.Flags().StringVarP(&initIP, "ip", "i", "", "Public cluster IP address (IPv4 or IPv6)")
	err := initCmd.MarkFlagRequired("ip")
	if err != nil {
		panic(err)
	}
	RootCmd.AddCommand(initCmd)
}
