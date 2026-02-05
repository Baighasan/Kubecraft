package config

import (
	"testing"
)

func TestConstants_UserLimits(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"MaxUsers is 15", MaxUsers, 15},
		{"MinUsernameLength is 3", MinUsernameLength, 3},
		{"MaxUsernameLength is 16", MaxUsernameLength, 16},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("got %d, want %d", tt.value, tt.want)
			}
		})
	}
}

func TestConstants_NetworkConfig(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  int
	}{
		{"RegistrationPort is 8080", RegistrationPort, 8080},
		{"RegistrationServicePort is 30099", RegistrationServicePort, 30099},
		{"McNodePortRangeMin is 30000", McNodePortRangeMin, 30000},
		{"McNodePortRangeMax is 30015", McNodePortRangeMax, 30015},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("got %d, want %d", tt.value, tt.want)
			}
		})
	}
}

func TestConstants_NodePortRange(t *testing.T) {
	// Verify NodePort range can accommodate all users
	rangeSize := McNodePortRangeMax - McNodePortRangeMin + 1

	if rangeSize < MaxUsers {
		t.Errorf("NodePort range (%d ports) cannot accommodate MaxUsers (%d)", rangeSize, MaxUsers)
	}
}

func TestConstants_ResourceNames(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"NamespacePrefix", NamespacePrefix, "mc-"},
		{"SystemNamespace", SystemNamespace, "kubecraft-system"},
		{"UserRoleName", UserRoleName, "minecraft-manager"},
		{"CapacityCheckerClusterRole", CapacityCheckerClusterRole, "kubecraft-capacity-checker"},
		{"CapacityCheckerBinding", CapacityCheckerBinding, "kc-users-capacity-check"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("got %q, want %q", tt.value, tt.want)
			}
		})
	}
}

func TestConstants_CommonLabel(t *testing.T) {
	expectedKey := "app"
	expectedValue := "kubecraft"
	expectedSelector := "app=kubecraft"

	if CommonLabelKey != expectedKey {
		t.Errorf("CommonLabelKey: got %q, want %q", CommonLabelKey, expectedKey)
	}

	if CommonLabelValue != expectedValue {
		t.Errorf("CommonLabelValue: got %q, want %q", CommonLabelValue, expectedValue)
	}

	if CommonLabelSelector != expectedSelector {
		t.Errorf("CommonLabelSelector: got %q, want %q", CommonLabelSelector, expectedSelector)
	}
}

func TestConstants_ReservedNames(t *testing.T) {
	expectedReserved := []string{
		"system",
		"admin",
		"root",
		"default",
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"kubecraft",
		"kubecraft-system",
	}

	if len(ReservedUserNames) == 0 {
		t.Fatal("ReservedUserNames is empty")
	}

	// Check that all expected reserved names are present
	for _, expected := range expectedReserved {
		found := false
		for _, reserved := range ReservedUserNames {
			if reserved == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected reserved name %q not found in ReservedUserNames", expected)
		}
	}
}

func TestConstants_TokenExpiry(t *testing.T) {
	// Verify token expiry is 5 years in seconds
	expectedSeconds := int64(5 * 365 * 24 * 60 * 60) // 157,680,000 seconds

	if TokenExpirySeconds != expectedSeconds {
		t.Errorf("TokenExpirySeconds: got %d, want %d (5 years)", TokenExpirySeconds, expectedSeconds)
	}

	// Verify it's approximately 5 years (allowing for leap years)
	yearsApprox := float64(TokenExpirySeconds) / (365.25 * 24 * 60 * 60)
	if yearsApprox < 4.9 || yearsApprox > 5.1 {
		t.Errorf("TokenExpirySeconds represents approximately %.2f years, expected ~5 years", yearsApprox)
	}
}

func TestConstants_ResourceLimits(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"ServerMemoryRequest", ServerMemoryRequest, "2Gi"},
		{"ServerMemoryLimit", ServerMemoryLimit, "4Gi"},
		{"ServerCPURequest", ServerCPURequest, "1000m"},
		{"ServerCPULimit", ServerCPULimit, "1500m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("got %q, want %q", tt.value, tt.want)
			}
		})
	}
}

func TestConstants_ClusterCapacity(t *testing.T) {
	// Verify cluster capacity is set for Oracle Cloud (14GB available from 16GB total)
	if TotalAvailableRAM != 14336 {
		t.Errorf("TotalAvailableRAM = %d, want 14336 (14GB)", TotalAvailableRAM)
	}

	// Verify capacity threshold matches server memory limit (4GB)
	if CapacityThreshold != 4096 {
		t.Errorf("CapacityThreshold = %d, want 4096 (4GB)", CapacityThreshold)
	}

	// Verify we can fit at least 3 servers concurrently at limit
	maxServers := TotalAvailableRAM / CapacityThreshold
	if maxServers < 3 {
		t.Errorf("Cluster can only fit %d servers at limit, need at least 3", maxServers)
	}
}
