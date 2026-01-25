package registration

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/baighasan/kubecraft/internal/config"
	"github.com/baighasan/kubecraft/internal/k8s"
)

// RegisterRequest represents the incoming JSON from the CLI
type RegisterRequest struct {
	Username string `json:"username"`
}

// RegisterResponse represents what we send back to the CLI
type RegisterResponse struct {
	Status   string `json:"status"`             // "success" or "error"
	Username string `json:"username,omitempty"` // only in success
	Token    string `json:"token,omitempty"`    // only in success
	Message  string `json:"message,omitempty"`  // only in error
}

func NewRegistrationHandler(k8sClient *k8s.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check HTTP method
		if r.Method != "POST" {
			sendError(w, http.StatusMethodNotAllowed, "Invalid request method")
			return
		}

		// Parse the JSON request body
		var req RegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON format")
			return
		}

		// Validate username
		if err := ValidateUsername(req.Username); err != nil {
			sendError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Check user limits
		count, err := k8sClient.CountUserNamespaces()
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check user count: %v", err))
			return
		}
		if count >= config.MaxUsers {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("max user limit reached (%d/%d)", count, config.MaxUsers))
			return
		}

		// Check if username already taken
		exists, err := k8sClient.NamespaceExists(req.Username)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to check username exists: %v", err))
			return
		}
		if exists {
			sendError(w, http.StatusConflict, fmt.Sprintf("Username already registered"))
			return
		}

		// Create k8s resources

		// Create namespace
		if err := k8sClient.CreateNamespace(req.Username); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create namespace: %v", err))
			return
		}

		// Create ServiceAccount
		if err := k8sClient.CreateServiceAccount(req.Username); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create serviceaccount: %v", err))
			return
		}

		// Create Role
		if err := k8sClient.CreateRole(); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create role: %v", err))
			return
		}

		// Create RoleBinding
		if err := k8sClient.CreateRoleBinding(req.Username); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create rolebinding: %v", err))
			return
		}

		// Create ResourceQuota
		if err := k8sClient.CreateResourceQuota(req.Username); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to create resourcequota: %v", err))
			return
		}

		// Add user to capacity checker ClusterRoleBinding
		if err := k8sClient.AddUserToCapacityChecker(req.Username); err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add user to capacity checker: %v", err))
			return
		}

		// Generate token
		token, err := k8sClient.GenerateToken(req.Username)
		if err != nil {
			sendError(w, http.StatusInternalServerError, fmt.Sprintf("failed to generate token: %v", err))
			return
		}

		// Send success response
		sendJSONResponse(w, http.StatusCreated, RegisterResponse{
			Status:   "success",
			Username: req.Username,
			Token:    token,
		})
	}
}

func sendJSONResponse(w http.ResponseWriter, statusCode int, response RegisterResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("failed to encode JSON response: %v\n", err)
	}
}

func sendError(w http.ResponseWriter, statusCode int, message string) {
	sendJSONResponse(w, statusCode, RegisterResponse{
		Status:  "error",
		Message: message,
	})
}
