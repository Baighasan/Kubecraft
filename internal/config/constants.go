package config

import "time"

// User Limits
const (
	MaxUsers          = 15
	MinUsernameLength = 3
	MaxUsernameLength = 16
)

// Network Configuration
const (
	ClusterAPIPort          = 6443  // Cluster API port
	RegistrationPort        = 8080  // Internal container port
	RegistrationServicePort = 30099 // External NodePort for registration
	McNodePortRangeMin      = 30000 // Start of Minecraft server NodePort range
	McNodePortRangeMax      = 30015 // End of range (supports 16 servers)
)

// Kubernetes Resources
const (
	NamespacePrefix = "mc-"
	SystemNamespace = "kubecraft-system"
)

// Common Label
const (
	CommonLabelKey      = "app"
	CommonLabelValue    = "kubecraft"
	CommonLabelValuePod = "minecraft"
	CommonLabelSelector = CommonLabelKey + "=" + CommonLabelValue
)

// RBAC Resource Names
const (
	UserRoleName                   = "minecraft-manager"
	CapacityCheckerClusterRole     = "kc-capacity-checker"
	CapacityCheckerBinding         = "kc-users-capacity-check"
	RegistrationClusterRole        = "kc-registration-admin"
	RegistrationClusterRoleBinding = "kc-registration-admin-binding"
)

// Resource Limits (per server) - Optimized for Oracle Cloud (16GB RAM, 3 OCPU)
const (
	ServerMemoryRequest = "2Gi"
	ServerMemoryLimit   = "4Gi"
	ServerJavaMemory    = "3G"
	ServerCPURequest    = "1000m"
	ServerCPULimit      = "1500m"
)

// Reserved Names
var ReservedUserNames = []string{
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

// Env variables injected at build time via ldflags
var (
	ClusterEndpoint = "localhost" // K8s API server address (host:port)
	NodeAddress     = "localhost" // Public IP/hostname for Minecraft connections
	TLSInsecure     = "false"
	ServerImage     = "ghcr.io/baighasan/kubecraft-minecraft:latest"
)

// Token Configuration
const (
	secondsPerYear     = 365 * 24 * 60 * 60
	TokenExpirySeconds = 5 * secondsPerYear
)

// Server Configuration - Optimized for Oracle Cloud (16GB RAM, 3 OCPU)
const (
	MinServerNameLength = 3
	MaxServerNameLength = 16

	MinecraftPort      = 25565
	ServerStorageSize  = "10Gi"
	ServerStorageClass = "local-path"
	CapacityThreshold  = 4096  // 4GB in MiB — minimum free RAM to allow creation (matches server limit)
	TotalAvailableRAM  = 14336 // 14GB in MiB — total RAM for workloads (16GB - 2GB system overhead)
)

// Readiness Check
const (
	MaxAttempts  = 30
	PollInterval = 5 * time.Second
)
