variable "tenancy_ocid" {
  description = "OCI tenancy OCID"
  type        = string
  sensitive   = true
}

variable "user_ocid" {
  description = "OCI user OCID for API authentication"
  type        = string
  sensitive   = true
}

variable "fingerprint" {
  description = "API key fingerprint"
  type        = string
  sensitive   = true
}

variable "private_key_path" {
  description = "Path to OCI API private key file"
  type        = string
}

variable "compartment_ocid" {
  description = "OCI compartment OCID for resource placement"
  type        = string
}

variable "region" {
  description = "OCI region to deploy in"
  type        = string
  default     = "us-ashburn-1"
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
}

variable "my_ip" {
  description = "IP address for SSH whitelist"
  type        = string
}

variable "node_port_min" {
  description = "Start of NodePort range for Minecraft servers"
  type        = number
  default     = 30000
}

variable "node_port_max" {
  description = "End of NodePort range for Minecraft servers"
  type        = number
  default     = 30099
}
