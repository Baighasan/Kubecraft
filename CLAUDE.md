# Minecraft Server Platform - Project Plan

## Project Overview

Self-service Minecraft server hosting platform where users can create, manage, and connect to their own Minecraft servers through a web dashboard. Built with Kubernetes, Terraform, and modern web technologies to demonstrate DevOps and cloud engineering skills.

## Architecture Summary

**Single-Node Kubernetes Setup:**
- One EC2 instance (t3.large: 2 vCPU, 8GB RAM) running K3s
- All components run as pods on this single node
- Terraform provisions AWS infrastructure
- K3s manages container orchestration

**Core Components:**
1. **Frontend Namespace**: React dashboard for user interaction
2. **Backend Namespace**: Node.js API + PostgreSQL database
3. **Minecraft Namespaces**: Dynamically created, one per server (isolated with ResourceQuotas)
4. **System Namespace**: Monitoring and automation services (optional)

**Tech Stack:**
- Infrastructure: AWS (EC2, VPC), Terraform, K3s
- Backend: Node.js, Express, TypeScript, PostgreSQL
- Frontend: React, TypeScript, Vite
- Container Orchestration: K3s (lightweight Kubernetes)
- CI/CD: GitHub Actions
- Container Registry: Docker Hub

## Project Scope

**Users:** 5 people
**Servers per user:** Up to 3
**Concurrent servers:** 2-3 running simultaneously
**Total servers:** Up to 15 (most stopped to save resources)
**Monthly cost:** ~$60 (single t3.large EC2 instance)

---

## Phase 0: Prerequisites & Setup

### Goals
- Set up development environment
- Learn foundational concepts
- Create accounts and tooling

### What to Learn
- **Git basics**: commit, push, pull, branches
- **Docker fundamentals**: 
  - Containers vs VMs
  - Dockerfile syntax
  - docker build, run, compose
- **Kubernetes concepts (high-level)**:
  - Pods, Deployments, Services, Namespaces
  - Basic architecture understanding
- **AWS basics**:
  - EC2, VPC, Security Groups
  - Free Tier limits

### Deliverables
- Development environment configured (Docker, kubectl, Terraform, Node.js installed)
- AWS, GitHub, Docker Hub accounts created
- Project repository initialized with proper directory structure
- Basic understanding of Docker and K8s concepts

---

## Phase 1: Local Development - Backend API

### Goals
- Build the API server that manages Minecraft servers
- Test locally without cloud infrastructure
- Learn backend development and Kubernetes client libraries

### Core Features to Implement

**Authentication System:**
- User registration endpoint
- User login endpoint
- JWT token generation and verification
- Password hashing

**Database Schema:**
- Users table (id, username, email, password_hash, created_at)
- Minecraft_servers table (id, user_id, server_name, namespace, version, game_mode, status, connection details)

**Kubernetes Management:**
- Function to create Minecraft server (namespace, StatefulSet, PVC, Service, ResourceQuota)
- Function to start server (scale to 1 replica)
- Function to stop server (scale to 0 replicas)
- Function to delete server (cleanup all resources)
- Function to get server status (query pod state)

**API Endpoints:**
- POST /api/auth/register
- POST /api/auth/login
- POST /api/servers (create new server)
- GET /api/servers (list user's servers)
- GET /api/servers/:id (get server details)
- POST /api/servers/:id/start
- POST /api/servers/:id/stop
- DELETE /api/servers/:id

### What to Learn

**Node.js + Express + TypeScript:**
- RESTful API design
- Middleware patterns (authentication, error handling)
- Environment variables management
- TypeScript types and interfaces

**PostgreSQL:**
- SQL basics (CREATE, INSERT, SELECT, UPDATE, DELETE)
- Primary keys, foreign keys, relationships
- Connection pooling

**JWT Authentication:**
- Token-based authentication flow
- Token generation and verification
- Protected route middleware

**Kubernetes Client Library:**
- @kubernetes/client-node usage
- Programmatically creating K8s resources
- Reading pod status and events
- Handling async K8s operations

### Deliverables
- Working API server with all endpoints functional
- PostgreSQL database with complete schema
- Successfully tested against local K3s cluster (k3d/minikube)
- API documentation (Postman collection or README)
- Can create/manage Minecraft servers programmatically

---

## Phase 2: Minecraft Server Docker Image

### Goals
- Create a custom Minecraft server Docker image
- Configure for Kubernetes deployment
- Test locally

### What Image Needs
- Base: OpenJDK 17
- Minecraft server.jar
- Configurable via environment variables (game mode, max players, memory limits)
- RCON enabled for server management
- Persistent volume mount for world data
- Startup script with proper memory settings

### What to Learn

**Dockerfile Best Practices:**
- Multi-stage builds
- Layer caching optimization
- COPY vs ADD
- EXPOSE, VOLUME, CMD directives

**Minecraft Server Configuration:**
- server.properties file options
- RCON (Remote Console) setup
- Game modes, difficulty settings
- Memory allocation for Java

**Environment Variables:**
- Parameterizing Docker containers
- Default values with ${VAR:-default} syntax
- Configuration management patterns

### Deliverables
- Working Minecraft server Docker image
- Image pushed to Docker Hub
- Tested locally (can connect with Minecraft client)
- Documented configuration options

---

## Phase 3: Kubernetes Manifests

### Goals
- Create K8s YAML manifests for all components
- Test full stack in local K3s cluster
- Understand K8s resource definitions

### Manifests to Create

**Namespaces:**
- frontend
- backend
- (minecraft namespaces created dynamically by API)

**Backend Resources:**
- PostgreSQL StatefulSet with PVC
- PostgreSQL Service (ClusterIP)
- API Server Deployment
- API Server Service (ClusterIP)
- Secrets (database password, JWT secret)

**Frontend Resources:**
- Frontend Deployment
- Frontend Service (LoadBalancer)

**Minecraft Server Template:**
- Namespace definition
- ResourceQuota (CPU/memory limits)
- StatefulSet for MC server
- PVC for world data
- Service (LoadBalancer with specific port)
- ConfigMap for server configuration

**Supporting Services (Optional):**
- Idle Monitor CronJob
- Backup Service CronJob
- Prometheus/Grafana for monitoring

### What to Learn

**Kubernetes YAML Syntax:**
- apiVersion, kind, metadata, spec structure
- Labels and selectors
- Resource organization

**Core K8s Concepts:**
- **Deployment**: Stateless applications (frontend, backend API)
- **StatefulSet**: Stateful applications (database, Minecraft servers)
- **Service**: Network access (ClusterIP, LoadBalancer)
- **PersistentVolumeClaim**: Storage for stateful data
- **Secret**: Sensitive data storage
- **ResourceQuota**: Resource limits per namespace
- **Namespace**: Isolation boundaries
- **ConfigMap**: Configuration data

**kubectl Commands:**
- apply, get, describe, logs, exec, delete
- Namespace-specific operations
- Resource inspection and debugging

### Deliverables
- Complete K8s manifests for all components
- Successfully deployed to local K3s cluster
- Can create Minecraft servers via API
- Database persists data across pod restarts
- All services communicating properly

---

## Phase 4: Frontend Dashboard

### Goals
- Build React dashboard for user interaction
- Implement authentication flow
- Server management UI

### Pages to Implement

**Login Page:**
- Login form (username/email, password)
- JWT token storage (localStorage)
- Redirect to dashboard on success

**Register Page:**
- Registration form
- Validation
- Success flow to login

**Dashboard:**
- Display user's servers (list view)
- Server cards showing: name, status, connection info
- Action buttons: Start, Stop, Delete
- "Create New Server" button

**Create Server Page:**
- Form: server name, Minecraft version, game mode, max players
- Submit to API
- Redirect to dashboard

### Components
- ServerCard (individual server display)
- ProtectedRoute (authentication guard)
- Navigation/Layout
- Loading states
- Error handling

### What to Learn

**React + TypeScript:**
- Functional components
- Hooks (useState, useEffect)
- Props and state management
- Conditional rendering
- Event handling

**React Router:**
- Client-side routing
- Protected routes
- Navigation between pages

**API Integration:**
- Axios for HTTP requests
- Async/await patterns
- Error handling
- Authentication headers (JWT)

**Modern Frontend Tooling:**
- Vite build tool
- TypeScript in React
- Environment variables
- Docker multi-stage builds for production

### Deliverables
- Working React dashboard with all pages
- Full authentication flow (register, login, logout)
- Can create, start, stop, delete servers through UI
- Responsive design (basic styling)
- Dockerized and ready to deploy
- Successfully tested against backend API

---

## Phase 5: Terraform Infrastructure

### Goals
- Define AWS infrastructure as code
- Provision EC2 instance with K3s
- Automate infrastructure deployment

### Infrastructure to Define

**Networking:**
- VPC (10.0.0.0/16)
- Public subnet
- Internet Gateway
- Route table

**Security:**
- Security Group with rules:
  - SSH (port 22) - restricted to your IP
  - HTTP (port 80) - public
  - HTTPS (port 443) - public
  - Minecraft ports (25565-25575) - public
- SSH key pair

**Compute:**
- EC2 instance (t3.large: 2 vCPU, 8GB RAM)
- Ubuntu 22.04 LTS
- 50GB EBS volume
- Elastic IP (static public IP)

**Automation:**
- User data script to install K3s on first boot
- Automatic kubeconfig setup

**Optional:**
- S3 bucket for backups
- Route53 DNS records
- IAM roles for EC2

### What to Learn

**Terraform Basics:**
- HCL syntax (resource, variable, output, data)
- Resource dependencies
- State management
- terraform init, plan, apply, destroy

**Terraform AWS Provider:**
- VPC and networking resources
- EC2 instances and AMI selection
- Security groups
- Data sources (AMI lookup)

**AWS Networking:**
- VPC concepts
- Subnets (public vs private)
- Internet gateways
- Route tables
- Security groups

**Cloud-Init / User Data:**
- EC2 bootstrap scripts
- Bash scripting basics
- K3s installation automation

**SSH Key Management:**
- Generating SSH keys
- Public vs private keys
- Using keys to access EC2

### Deliverables
- Complete Terraform configuration (main.tf, variables.tf, outputs.tf)
- Successfully deployed AWS infrastructure
- K3s cluster running on EC2
- Can connect via kubectl from local machine
- Elastic IP assigned (static access)
- Documented deployment commands
- Can tear down and recreate infrastructure reliably

---

## Phase 6: CI/CD Pipeline

### Goals
- Automate building and pushing Docker images
- Automate deployment to K3s cluster
- Implement continuous deployment

### Pipelines to Create

**Backend Deployment:**
- Trigger on push to main (backend/** paths)
- Build Docker image
- Push to Docker Hub
- Update K8s deployment
- Verify rollout success

**Frontend Deployment:**
- Trigger on push to main (frontend/** paths)
- Build Docker image
- Push to Docker Hub
- Update K8s deployment
- Verify rollout success

**Infrastructure Changes:**
- Manual terraform apply workflow (optional)

### GitHub Secrets Needed
- DOCKER_USERNAME
- DOCKER_PASSWORD
- KUBE_CONFIG (base64 encoded kubeconfig)

### What to Learn

**GitHub Actions:**
- Workflow syntax (on, jobs, steps)
- Event triggers (push, pull_request, paths)
- Using actions from marketplace
- Secrets management
- Environment variables

**Docker Registry:**
- Docker Hub usage
- Image tagging strategies (latest, commit SHA)
- Multi-arch builds (optional)

**kubectl Deployment Strategies:**
- Rolling updates
- kubectl set image command
- kubectl rollout commands
- Zero-downtime deployments
- Health checks

**CI/CD Best Practices:**
- Separate workflows per service
- Path-based triggers
- Semantic versioning
- Build artifacts and caching

### Deliverables
- Working CI/CD pipelines for backend and frontend
- Automated deployments on git push to main
- Docker images built and tagged with commit SHA
- Deployment verification in workflows
- Initial deployment script for first-time setup
- Documentation of CI/CD process

---

## Phase 7: Testing & Refinement

### Goals
- End-to-end testing
- Bug fixes and polish
- Performance optimization
- Complete documentation

### Testing Checklist

**Functionality:**
- User registration and login flow
- Create Minecraft server through dashboard
- Verify K8s resources created correctly
- Server starts and is reachable
- Connect with Minecraft client
- World data persists
- Stop server (pod scales to 0, PVC remains)
- Restart server (world data restored)
- Delete server (all resources cleaned up)

**Load Testing:**
- Create 3 servers simultaneously
- Monitor resource usage (kubectl top nodes)
- Verify 8GB RAM is sufficient
- Test with multiple concurrent players

**Common Issues to Fix:**
- Server status updates
- LoadBalancer IP assignment delays
- Persistent volume mounting
- API error handling
- Frontend data refresh

### Optimizations

**Backend:**
- Database connection pooling
- Server status caching
- Rate limiting
- Improved error messages

**Frontend:**
- Loading states
- Error handling UI
- Auto-refresh server status
- UI/UX improvements

**Kubernetes:**
- Resource requests/limits tuning
- Readiness/liveness probes
- Health checks for MC servers

### Documentation to Complete

**README.md:**
- Project overview
- Architecture diagram
- Features list
- Tech stack
- Deployment instructions
- Cost breakdown
- Usage guide

**ARCHITECTURE.md:**
- Detailed architecture explanation
- Component interactions
- Data flow diagrams
- Design decisions

**API Documentation:**
- All endpoints
- Request/response formats
- Authentication flow
- Error codes

### What to Learn

**Debugging Kubernetes:**
- Reading pod logs effectively
- Using kubectl describe
- Executing into pods
- Analyzing events
- Troubleshooting networking

**Performance Monitoring:**
- kubectl top commands
- Resource metrics interpretation
- Identifying bottlenecks
- Optimization strategies

**Technical Writing:**
- Clear documentation structure
- Architecture diagrams
- Code examples
- User guides

### Deliverables
- Fully tested and working platform
- 5 friends successfully using the platform
- All bugs fixed
- Performance optimized
- Complete documentation
- Clean, commented codebase

---

## Phase 8: Portfolio Presentation

### Goals
- Create compelling portfolio materials
- Prepare for interview questions
- Record demos
- Update resume

### Portfolio Materials

**Demo Video (5-10 minutes):**
- Architecture walkthrough
- Live deployment demonstration
- Dashboard functionality
- Minecraft server creation
- K8s resource inspection
- Cost optimization explanation
- Scaling discussion

**Portfolio Website Update:**
- Project hero section with architecture diagram
- Problem statement
- Solution overview
- Technical deep-dive
- Key learnings
- Code snippets (interesting parts)
- Results/metrics
- GitHub link

**GitHub Repository Polish:**
- Clean commit history
- Comprehensive README
- Code comments
- Architecture documentation
- Setup instructions
- Remove secrets
- Add LICENSE

### Interview Preparation

**Technical Questions to Prepare:**
- "Why K3s instead of EKS?"
- "How do you handle state for Minecraft servers?"
- "What happens if the EC2 instance fails?"
- "How would you scale this to 100 users?"
- "Why StatefulSets instead of Deployments?"
- "Explain your CI/CD pipeline"
- "How do you ensure cost optimization?"

**Challenges & Solutions:**
- Hardest technical problem faced
- Performance issues encountered
- How you debugged K8s issues
- Trade-offs you made

**Trade-offs Discussion:**
- Single node vs multi-node cluster
- Cost vs availability
- Complexity vs simplicity
- Managed services vs self-hosted

**Future Improvements:**
- Auto-scaling based on player count
- Multi-region deployment
- Server templates/modpacks
- Discord bot integration
- Monitoring dashboards
- Backup/restore functionality

### Resume Bullet Points

Key achievements to highlight:
- Architected self-service platform on AWS with K8s and Terraform
- Implemented IaC for reproducible deployments
- Built REST API with K8s client for dynamic resource management
- Optimized costs through auto-shutdown and resource quotas
- Established CI/CD pipeline with GitHub Actions
- Served 5 concurrent users with 15 total server instances

### What to Learn

**Interview Skills:**
- STAR method for behavioral questions
- Technical question frameworks
- System design approach
- Articulating trade-offs clearly

**Portfolio Best Practices:**
- Scannable layouts
- Compelling visuals
- Results-focused content
- Technical depth without overwhelming

### Deliverables
- Polished portfolio page on hasanbaig.net
- 5-10 minute demo video
- Interview preparation document
- Updated resume with project
- GitHub repository showcase-ready
- Blog post explaining project in depth

---

## Timeline Summary

| Week | Phase | Focus |
|------|-------|-------|
| 1 | Phase 0 | Prerequisites, environment setup, foundational learning |
| 2-3 | Phase 1 | Backend API development, K8s client library |
| 3 | Phase 2 | Minecraft Docker image |
| 4 | Phase 3 | Kubernetes manifests, local testing |
| 5 | Phase 4 | Frontend React dashboard |
| 6 | Phase 5 | Terraform infrastructure, AWS deployment |
| 7 | Phase 6 | CI/CD pipeline setup |
| 8 | Phase 7 | Testing, bug fixes, optimization |
| 9 | Phase 8 | Portfolio presentation, interview prep |

**Total Duration:** ~9 weeks (2-3 months) at 10-15 hours/week

---

## Success Criteria

**MVP (Minimum Viable Product):**
- Users can register and login
- Users can create 1 Minecraft server
- Server starts and can be connected to
- World data persists
- Deployed on AWS with Terraform
- Basic documentation

**Complete Project:**
- All features working (create, start, stop, delete)
- Up to 3 servers per user
- 5 friends actively using
- CI/CD pipeline functional
- Complete documentation
- Portfolio writeup and demo video

**Stretch Goals:**
- Idle auto-shutdown
- Backup/restore to S3
- Monitoring dashboard (Prometheus/Grafana)
- Multiple Minecraft versions support
- Server templates (vanilla, modded, etc.)

---

## Key Learning Outcomes

**DevOps & Platform Engineering:**
- Infrastructure as Code with Terraform
- Kubernetes orchestration and administration
- CI/CD pipeline implementation
- Container management and optimization
- Cloud architecture on AWS

**Backend Development:**
- RESTful API design
- Authentication and authorization
- Database design and management
- Kubernetes client library usage
- Async operations handling

**Frontend Development:**
- Modern React with TypeScript
- State management
- API integration
- User authentication flows

**System Design:**
- Multi-tenancy patterns
- Resource isolation
- Cost optimization strategies
- Scalability considerations
- Trade-off analysis

**Production Operations:**
- Monitoring and debugging
- Performance optimization
- Documentation
- Testing strategies

---

## Resources

**Official Documentation:**
- Kubernetes: https://kubernetes.io/docs/
- K3s: https://docs.k3s.io/
- Terraform: https://developer.hashicorp.com/terraform
- Docker: https://docs.docker.com/
- Node.js K8s Client: https://github.com/kubernetes-client/javascript

**Learning Paths:**
- Kubernetes Basics Tutorial
- Terraform AWS Provider Guide
- React Official Tutorial
- TypeScript Handbook

**Tools:**
- k3d for local K3s clusters
- kubectl cheatsheet
- Postman for API testing
- VS Code with relevant extensions