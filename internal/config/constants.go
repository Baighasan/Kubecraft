package config

// User Limits
const (
	MaxUsers          = 15
	MinUsernameLength = 3
	MaxUsernameLength = 16
)

// Network Configuration
const (
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
	CommonLabelSelector = CommonLabelKey + "=" + CommonLabelValue
)

// RBAC Resource Names
const (
	UserRoleName               = "minecraft-manager"
	CapacityCheckerClusterRole = "kubecraft-capacity-checker"
	CapacityCheckerBinding     = "kc-users-capacity-check"
	RegistrationClusterRole    = "kc-registration-admin"
)

// Resource Limits (per server)
const (
	ServerMemoryRequest = "768Mi"
	ServerMemoryLimit   = "1Gi"
	ServerCPURequest    = "500m"
	ServerCPULimit      = "750m"
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

// Cluster endpoint (injected at build time via ldflags)
var ClusterEndpoint = "localhost"

// Token Configuration
const (
	secondsPerYear     = 365 * 24 * 60 * 60
	TokenExpirySeconds = 5 * secondsPerYear
)
