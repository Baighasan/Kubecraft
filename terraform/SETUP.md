# Terraform Setup Guide

Step-by-step guide for deploying Kubecraft infrastructure to Oracle Cloud.

---

## Prerequisites

- Oracle Cloud account with Always Free tier
- Terraform installed (`terraform version`)
- SSH key pair generated
- OCI API keys generated and uploaded to console

---

## Step 1: Set Up Oracle Cloud Account
**Status:** Complete

- [x] Create Oracle Cloud account
- [x] Complete account verification
- [x] Locate Tenancy OCID and Compartment OCID

---

## Step 2: Generate API Keys for Terraform
**Status:** Complete

- [x] Create `~/.oci/` directory
- [x] Generate private key (`~/.oci/oci_api_key.pem`)
- [x] Generate public key (`~/.oci/oci_api_key_public.pem`)
- [x] Upload public key to OCI Console
- [x] Note down: tenancy_ocid, user_ocid, fingerprint
- [x] Find compartment_ocid

---

## Step 3: Install Terraform and OCI CLI
**Status:** Complete

- [x] Install Terraform
- [x] Verify: `terraform version`
- [x] (Optional) Install OCI CLI
- [x] (Optional) Test OCI CLI authentication

---

## Step 4: Create Terraform Project Structure
**Status:** Complete

- [x] Create `terraform/` directory
- [x] Create empty .tf files (main, variables, network, security, compute, outputs)
- [x] Create cloud-init.yaml
- [x] Create .gitignore
- [x] Create terraform.tfvars.example
- [x] Create terraform.tfvars with real values

---

## Step 5: Write Provider Configuration (main.tf)
**Status:** Complete

Configure Terraform to communicate with Oracle Cloud.

**Tasks:**
- [x] Define required Terraform version (>= 1.0.0)
- [x] Define required OCI provider (oracle/oci ~> 5.0)
- [x] Configure provider block with authentication variables
- [x] Add data source for availability domain
- [x] Add data source for Ubuntu ARM image

**Key Concepts:**
| Concept | Description |
|---------|-------------|
| Provider | Plugin that talks to cloud API (OCI) |
| Data Source | Read-only query for existing resources |
| Required Providers | Lock provider versions for reproducibility |

---

## Step 6: Define Input Variables (variables.tf)
**Status:** Complete

- [x] Define OCI credential variables (tenancy, user, fingerprint, key path)
- [x] Define region and compartment variables
- [x] Define SSH key and IP whitelist variables
- [x] Define NodePort range variables

---

## Step 7: Create Network Resources (network.tf)
**Status:** Complete

- [x] Create VCN (10.0.0.0/16)
- [x] Create Internet Gateway
- [x] Create Route Table (0.0.0.0/0 → IGW)
- [x] Create Public Subnet (10.0.1.0/24)

---

## Step 8: Create Security Rules (security.tf)
**Status:** Complete

- [x] Create Security List
- [x] Add SSH rule (port 22, your IP only)
- [x] Add Kubernetes API rule (port 6443)
- [x] Add NodePort rule (30000-30099)
- [x] Add egress rule (allow all outbound)

---

## Step 9: Create Compute Instance (compute.tf)
**Status:** Complete

- [x] Configure instance shape (VM.Standard.A2.Flex)
- [x] Configure resources (3 OCPU, 16GB RAM)
- [x] Configure boot volume (100GB, Ubuntu 22.04 ARM)
- [x] Attach to subnet with public IP
- [x] Pass SSH key and cloud-init via metadata

---

## Step 10: Write Cloud-Init Script (cloud-init.yaml)
**Status:** Complete

- [x] Update and upgrade packages
- [x] Install K3s
- [x] Wait for K3s to be ready
- [x] Set up anti-idle cron job

---

## Step 11: Define Outputs (outputs.tf)
**Status:** Complete

- [x] Output instance public IP
- [x] Output SSH command
- [x] Output kubeconfig retrieval command
- [x] Output cluster endpoint

---

## Step 12: Deploy and Verify
**Status:** Not Started

- [ ] Run `terraform init`
- [ ] Run `terraform plan`
- [ ] Run `terraform apply`
- [ ] SSH into instance
- [ ] Verify K3s is running
- [ ] Copy kubeconfig to local machine
- [ ] Test kubectl locally

---

## Step 13: Commit and Document
**Status:** Not Started

- [ ] Verify .gitignore excludes secrets
- [ ] Commit Terraform files
- [ ] Update documentation

---

## Quick Reference

**Terraform Commands:**
```bash
terraform init      # Download providers
terraform plan      # Preview changes
terraform apply     # Create resources
terraform destroy   # Delete everything
terraform output    # Show outputs
```

**Useful OCI Commands:**
```bash
# Test API authentication
oci iam availability-domain list --compartment-id <tenancy_ocid>

# Get your public IP
curl -s ifconfig.me
```

**File Structure:**
```
terraform/
├── main.tf              # Provider, data sources
├── variables.tf         # Input variables
├── network.tf           # VCN, subnet, gateway
├── security.tf          # Firewall rules
├── compute.tf           # Instance configuration
├── outputs.tf           # Output values
├── cloud-init.yaml      # K3s installation
├── terraform.tfvars     # Your values (DO NOT COMMIT)
├── terraform.tfvars.example
└── .gitignore
```
