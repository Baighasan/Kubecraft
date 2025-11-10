# Kubecraft API Design Specification

**Version:** 1.0
**Last Updated:** 2025-01-15
**Framework:** Node.js + Express + TypeScript

---

## Table of Contents
1. [Architecture Overview](#architecture-overview)
2. [Authentication Endpoints](#authentication-endpoints)
3. [Server Management Endpoints](#server-management-endpoints)
4. [Health & System Endpoints](#health--system-endpoints)
5. [Error Response Format](#error-response-format)
6. [Database Models](#database-models)
7. [Project Structure](#project-structure)
8. [Middleware Chain](#middleware-chain)
9. [Design Decisions](#design-decisions)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     API Architecture                         │
├─────────────────────────────────────────────────────────────┤
│ Layer 1: Authentication (JWT-based)                         │
│ Layer 2: User Management                                     │
│ Layer 3: Server Management (CRUD + Actions)                 │
│ Layer 4: Health/Status Endpoints                            │
│ Layer 5: Error Handling                                      │
└─────────────────────────────────────────────────────────────┘
```

**Base URL:** `http://localhost:3000/api`

**Authentication:** JWT Bearer Token (except `/auth/*` and `/health`)

**Content-Type:** `application/json`

---

## Authentication Endpoints

### POST /api/auth/register
Create a new user account.

**Request Body:**
```json
{
  "username": "string (3-50 chars, alphanumeric + underscore)",
  "email": "string (valid email)",
  "password": "string (min 8 chars, 1 uppercase, 1 number)"
}
```

**Example:**
```json
{
  "username": "player1",
  "email": "player1@example.com",
  "password": "SecurePass123"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": 1,
    "username": "player1",
    "email": "player1@example.com",
    "createdAt": "2025-01-15T10:30:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Validation errors (weak password, invalid email)
- `409 Conflict`: Username or email already exists

**Validation Rules:**
- Username: 3-50 characters, alphanumeric + underscore only
- Email: Valid email format
- Password: Minimum 8 characters, at least 1 uppercase letter, 1 number

---

### POST /api/auth/login
Authenticate user and receive JWT token.

**Request Body:**
```json
{
  "email": "string",
  "password": "string"
}
```

**Example:**
```json
{
  "email": "player1@example.com",
  "password": "SecurePass123"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expiresIn": "24h",
    "user": {
      "id": 1,
      "username": "player1",
      "email": "player1@example.com"
    }
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid credentials
- `400 Bad Request`: Missing fields

**Token Usage:**
All subsequent requests must include:
```
Authorization: Bearer <token>
```

---

### GET /api/auth/me
Get current authenticated user information.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "username": "player1",
    "email": "player1@example.com",
    "serverCount": 2,
    "createdAt": "2025-01-15T10:30:00Z"
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid or expired token

---

## Server Management Endpoints

All server endpoints require authentication via JWT Bearer token.

---

### POST /api/servers
Create a new Minecraft server.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Request Body:**
```json
{
  "serverName": "string (3-30 chars, alphanumeric + hyphen)",
  "minecraftVersion": "string (default: '1.20.1')",
  "gameMode": "survival|creative|adventure|spectator (default: survival)",
  "maxPlayers": "number (1-100, default: 20)",
  "difficulty": "peaceful|easy|normal|hard (default: normal)"
}
```

**Example:**
```json
{
  "serverName": "my-survival-world",
  "minecraftVersion": "1.20.1",
  "gameMode": "survival",
  "maxPlayers": 20,
  "difficulty": "normal"
}
```

**Response (201 Created):**
```json
{
  "success": true,
  "message": "Minecraft server created successfully",
  "data": {
    "id": 1,
    "userId": 1,
    "serverName": "my-survival-world",
    "namespace": "mc-player1-my-survival-world",
    "minecraftVersion": "1.20.1",
    "gameMode": "survival",
    "maxPlayers": 20,
    "difficulty": "normal",
    "status": "stopped",
    "serverIp": null,
    "serverPort": null,
    "createdAt": "2025-01-15T11:00:00Z"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Validation errors, duplicate server name for user
- `403 Forbidden`: User already has 3 servers (quota exceeded)
- `500 Internal Server Error`: Kubernetes resource creation failed

**Business Logic:**
1. Verify user has fewer than 3 servers
2. Validate server name is unique for this user
3. Generate Kubernetes namespace: `mc-{username}-{servername}`
4. Create K8s resources:
   - Namespace
   - ResourceQuota (1 CPU, 2Gi RAM)
   - StatefulSet (Minecraft server)
   - PersistentVolumeClaim (5Gi for world data)
   - LoadBalancer Service (port 25565)
5. Insert record into database with status='stopped'

---

### GET /api/servers
List all servers owned by the authenticated user.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Query Parameters (Optional):**
```
?status=running|stopped|error
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "servers": [
      {
        "id": 1,
        "serverName": "my-survival-world",
        "namespace": "mc-player1-my-survival-world",
        "minecraftVersion": "1.20.1",
        "gameMode": "survival",
        "status": "running",
        "serverIp": "192.168.1.100",
        "serverPort": 25565,
        "createdAt": "2025-01-15T11:00:00Z",
        "lastStarted": "2025-01-15T12:00:00Z"
      },
      {
        "id": 2,
        "serverName": "creative-build",
        "namespace": "mc-player1-creative-build",
        "minecraftVersion": "1.20.1",
        "gameMode": "creative",
        "status": "stopped",
        "serverIp": null,
        "serverPort": null,
        "createdAt": "2025-01-14T09:00:00Z",
        "lastStarted": null
      }
    ],
    "total": 2,
    "quota": {
      "used": 2,
      "max": 3
    }
  }
}
```

---

### GET /api/servers/:id
Get detailed information about a specific server.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id`: Server ID (integer)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "userId": 1,
    "serverName": "my-survival-world",
    "namespace": "mc-player1-my-survival-world",
    "minecraftVersion": "1.20.1",
    "gameMode": "survival",
    "maxPlayers": 20,
    "difficulty": "normal",
    "status": "running",
    "serverIp": "192.168.1.100",
    "serverPort": 25565,
    "createdAt": "2025-01-15T11:00:00Z",
    "lastStarted": "2025-01-15T12:00:00Z",
    "kubernetes": {
      "podStatus": "Running",
      "podName": "minecraft-0",
      "pvcSize": "5Gi",
      "resourceLimits": {
        "cpu": "1000m",
        "memory": "2Gi"
      }
    },
    "connectionInfo": {
      "host": "192.168.1.100:25565",
      "instructions": "Open Minecraft > Multiplayer > Add Server > Enter host address"
    }
  }
}
```

**Error Responses:**
- `404 Not Found`: Server doesn't exist
- `403 Forbidden`: Server belongs to another user

**Business Logic:**
1. Query database for server by ID
2. Verify ownership (server.userId === authenticated user ID)
3. Query Kubernetes for real-time pod status
4. Merge database + Kubernetes data
5. Return enriched server details

---

### POST /api/servers/:id/start
Start a stopped Minecraft server.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id`: Server ID (integer)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Server is starting...",
  "data": {
    "id": 1,
    "status": "starting",
    "estimatedTime": "30-60 seconds",
    "serverIp": "192.168.1.100",
    "serverPort": 25565
  }
}
```

**Error Responses:**
- `400 Bad Request`: Server is already running
- `404 Not Found`: Server doesn't exist
- `403 Forbidden`: Unauthorized (not your server)
- `500 Internal Server Error`: Failed to scale Kubernetes StatefulSet

**Business Logic:**
1. Verify ownership
2. Check current status (return 400 if already running)
3. Scale StatefulSet to 1 replica
4. Wait for pod to reach "Running" state (timeout: 120 seconds)
5. Get LoadBalancer Service IP/port
6. Update database:
   - status='running'
   - serverIp=<IP>
   - serverPort=<port>
   - lastStarted=<timestamp>
7. Return connection details

---

### POST /api/servers/:id/stop
Stop a running Minecraft server (preserves world data).

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id`: Server ID (integer)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Server stopped successfully",
  "data": {
    "id": 1,
    "status": "stopped",
    "note": "World data has been preserved and will be available when you restart"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Server is already stopped
- `404 Not Found`: Server doesn't exist
- `403 Forbidden`: Unauthorized

**Business Logic:**
1. Verify ownership
2. Check current status (return 400 if already stopped)
3. Scale StatefulSet to 0 replicas
4. Update database:
   - status='stopped'
   - serverIp=null
   - serverPort=null
5. PersistentVolumeClaim remains (data persists)

---

### DELETE /api/servers/:id
Permanently delete a Minecraft server and all associated data.

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id`: Server ID (integer)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Server and all data deleted permanently",
  "data": {
    "id": 1,
    "serverName": "my-survival-world",
    "warning": "This action cannot be undone"
  }
}
```

**Error Responses:**
- `404 Not Found`: Server doesn't exist
- `403 Forbidden`: Unauthorized
- `500 Internal Server Error`: Failed to delete Kubernetes namespace

**Business Logic:**
1. Verify ownership
2. Delete entire Kubernetes namespace (cascades deletion of):
   - StatefulSet
   - PersistentVolumeClaim (world data destroyed)
   - Service
   - ResourceQuota
3. Delete database record
4. Return success confirmation

---

### GET /api/servers/:id/status
Get real-time server status (lightweight endpoint for polling).

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Path Parameters:**
- `id`: Server ID (integer)

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "id": 1,
    "status": "running",
    "podStatus": "Running",
    "serverIp": "192.168.1.100",
    "serverPort": 25565,
    "uptime": "2h 15m"
  }
}
```

**Note:** This endpoint is optimized for frequent polling from the frontend dashboard.

---

## Health & System Endpoints

### GET /api/health
Check API health status (no authentication required).

**Response (200 OK):**
```json
{
  "success": true,
  "status": "healthy",
  "timestamp": "2025-01-15T12:00:00Z",
  "services": {
    "database": "connected",
    "kubernetes": "connected"
  }
}
```

**Response (503 Service Unavailable):**
```json
{
  "success": false,
  "status": "unhealthy",
  "timestamp": "2025-01-15T12:00:00Z",
  "services": {
    "database": "disconnected",
    "kubernetes": "connected"
  }
}
```

---

### GET /api/stats
Platform-wide statistics (protected, optional admin endpoint).

**Headers Required:**
```
Authorization: Bearer <jwt_token>
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "totalUsers": 5,
    "totalServers": 12,
    "runningServers": 3,
    "stoppedServers": 9,
    "resourceUsage": {
      "cpu": "2.5 cores",
      "memory": "6Gi / 8Gi"
    }
  }
}
```

---

## Error Response Format

All errors follow a consistent structure for easy client-side handling.

**Standard Error Response:**
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "field": "fieldName (optional, for validation errors)",
    "timestamp": "2025-01-15T12:00:00Z"
  }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `VALIDATION_ERROR` | 400 | Invalid input data |
| `UNAUTHORIZED` | 401 | Missing or invalid JWT token |
| `FORBIDDEN` | 403 | Valid token but insufficient permissions |
| `NOT_FOUND` | 404 | Resource doesn't exist |
| `QUOTA_EXCEEDED` | 403 | User hit server limit (3 max) |
| `DUPLICATE_ERROR` | 409 | Resource name conflict |
| `KUBERNETES_ERROR` | 500 | Kubernetes operation failed |
| `DATABASE_ERROR` | 500 | Database operation failed |
| `INTERNAL_ERROR` | 500 | Unknown server error |

**Example Error Responses:**

**Validation Error (400):**
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Server name must be between 3-30 characters",
    "field": "serverName",
    "timestamp": "2025-01-15T12:00:00Z"
  }
}
```

**Unauthorized (401):**
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token",
    "timestamp": "2025-01-15T12:00:00Z"
  }
}
```

**Quota Exceeded (403):**
```json
{
  "success": false,
  "error": {
    "code": "QUOTA_EXCEEDED",
    "message": "You have reached the maximum of 3 servers. Delete a server to create a new one.",
    "timestamp": "2025-01-15T12:00:00Z"
  }
}
```

---

## Database Models

### Users Table

```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);
```

### Minecraft Servers Table

```sql
CREATE TABLE minecraft_servers (
  id SERIAL PRIMARY KEY,
  user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
  server_name VARCHAR(100) NOT NULL,
  namespace VARCHAR(100) UNIQUE NOT NULL,
  minecraft_version VARCHAR(20) DEFAULT '1.20.1',
  game_mode VARCHAR(20) DEFAULT 'survival',
  max_players INTEGER DEFAULT 20,
  difficulty VARCHAR(20) DEFAULT 'normal',
  status VARCHAR(20) DEFAULT 'stopped',
  server_ip VARCHAR(100),
  server_port INTEGER,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_started TIMESTAMP,

  UNIQUE(user_id, server_name)
);

CREATE INDEX idx_user_servers ON minecraft_servers(user_id);
CREATE INDEX idx_namespace ON minecraft_servers(namespace);
CREATE INDEX idx_status ON minecraft_servers(status);
```

**Constraints:**
- Each user can have max 3 servers (enforced in application logic)
- Server names must be unique per user
- Namespaces are globally unique
- Cascade delete: deleting a user deletes all their servers

---

## Project Structure

```
backend/
├── src/
│   ├── config/
│   │   ├── database.ts          # PostgreSQL connection pool
│   │   ├── kubernetes.ts        # K8s client initialization
│   │   └── env.ts               # Environment variables validation
│   │
│   ├── middleware/
│   │   ├── auth.middleware.ts   # JWT token verification
│   │   ├── error.middleware.ts  # Global error handler
│   │   ├── logger.middleware.ts # Request/response logging
│   │   └── validate.middleware.ts # Input validation
│   │
│   ├── models/
│   │   ├── user.model.ts        # User database operations
│   │   └── server.model.ts      # Server database operations
│   │
│   ├── services/
│   │   ├── auth.service.ts      # JWT generation, password hashing
│   │   └── kubernetes.service.ts # K8s resource CRUD operations
│   │
│   ├── controllers/
│   │   ├── auth.controller.ts   # Authentication endpoint handlers
│   │   └── server.controller.ts # Server management handlers
│   │
│   ├── routes/
│   │   ├── auth.routes.ts       # Auth route definitions
│   │   ├── server.routes.ts     # Server route definitions
│   │   └── index.ts             # Route aggregator
│   │
│   ├── types/
│   │   ├── express.d.ts         # Extend Express Request interface
│   │   ├── user.types.ts        # User-related interfaces
│   │   └── server.types.ts      # Server-related interfaces
│   │
│   ├── utils/
│   │   ├── errors.ts            # Custom error classes
│   │   ├── validators.ts        # Input validation functions
│   │   └── logger.ts            # Winston logger configuration
│   │
│   └── index.ts                 # Express app entry point
│
├── database/
│   └── init.sql                 # Database schema
│
├── tests/                       # Unit and integration tests
│   ├── auth.test.ts
│   ├── servers.test.ts
│   └── kubernetes.test.ts
│
├── .env.example                 # Environment variables template
├── .gitignore
├── package.json
├── tsconfig.json
└── README.md
```

---

## Middleware Chain

Every protected endpoint flows through this pipeline:

```
Incoming Request
    ↓
[1] CORS Middleware
    ↓
[2] JSON Body Parser
    ↓
[3] Request Logger (log method, path, timestamp)
    ↓
[4] JWT Auth Middleware (validate token, extract user, attach to req.user)
    ↓
[5] Route Handler (controller logic)
    ↓
[6] Response
    ↓
[7] Error Handler Middleware (catch all errors, format response)
    ↓
Response to Client
```

**Middleware Details:**

**1. CORS Middleware:**
- Allow frontend origin (localhost:5173 in dev)
- Allow credentials
- Expose necessary headers

**2. JSON Body Parser:**
- Parse incoming JSON payloads
- Limit: 10mb
- Handle malformed JSON errors

**3. Request Logger:**
```typescript
// Log format: [2025-01-15 12:00:00] POST /api/servers - 201 (145ms)
logger.info(`${method} ${path} - ${statusCode} (${duration}ms)`);
```

**4. JWT Auth Middleware:**
```typescript
export const authMiddleware = async (req, res, next) => {
  const token = req.headers.authorization?.split(' ')[1];

  if (!token) {
    return res.status(401).json({
      success: false,
      error: { code: 'UNAUTHORIZED', message: 'No token provided' }
    });
  }

  try {
    const decoded = jwt.verify(token, process.env.JWT_SECRET);
    req.user = decoded; // Attach user to request
    next();
  } catch (error) {
    return res.status(401).json({
      success: false,
      error: { code: 'UNAUTHORIZED', message: 'Invalid or expired token' }
    });
  }
};
```

**5. Error Handler Middleware:**
```typescript
export const errorHandler = (err, req, res, next) => {
  logger.error(`Error: ${err.message}`, { stack: err.stack });

  const statusCode = err.statusCode || 500;
  const errorCode = err.code || 'INTERNAL_ERROR';

  res.status(statusCode).json({
    success: false,
    error: {
      code: errorCode,
      message: err.message || 'An unexpected error occurred',
      timestamp: new Date().toISOString()
    }
  });
};
```

---

## Design Decisions

### 1. RESTful Architecture
**Why:** Standard, predictable, easy to document and consume. HTTP verbs map naturally to CRUD operations.

**Alternatives Considered:**
- GraphQL: Overkill for simple CRUD API
- gRPC: Unnecessarily complex, requires protobuf

---

### 2. JWT Token Authentication
**Why:**
- Stateless (no server-side sessions)
- Scales horizontally
- Works across multiple services
- Standard industry practice

**Implementation:**
- Tokens expire in 24 hours
- Stored in client localStorage
- Sent via Authorization header

**Security:**
- Passwords hashed with bcrypt (10 salt rounds)
- HTTPS only in production
- Token secret stored in environment variables

---

### 3. Separation of Concerns (Layered Architecture)

**Controllers:** Handle HTTP requests/responses, input validation
**Services:** Business logic, Kubernetes operations
**Models:** Database queries and data access
**Middleware:** Cross-cutting concerns (auth, logging, errors)

**Why:**
- Easier to test (mock services in controller tests)
- Reusable business logic
- Clear responsibility boundaries
- Easier to maintain and extend

---

### 4. Namespace Isolation for Minecraft Servers

**Pattern:** `mc-{username}-{servername}`

**Why:**
- Complete resource isolation per server
- Easy cleanup (delete namespace = delete everything)
- ResourceQuotas prevent resource hogging
- Prevents name collisions
- Clear ownership model

---

### 5. Async/Await for All I/O Operations

**Why:**
- Non-blocking operations (database, Kubernetes API)
- Better error handling than callbacks
- Readable code flow
- Prevents thread blocking

---

### 6. StatefulSet for Minecraft Servers

**Why:**
- Stable network identity
- Persistent storage (PVC remains across pod restarts)
- Ordered deployment/scaling
- Perfect for stateful applications like game servers

**Alternative Considered:**
- Deployment: Doesn't guarantee stable storage, better for stateless apps

---

### 7. Scale to 0 Replicas for "Stopped" Servers

**Why:**
- Saves compute resources (CPU, RAM)
- PersistentVolumeClaim remains (world data persists)
- Fast restart (just scale to 1)
- Cost-effective for idle servers

---

### 8. Database as Source of Truth + Kubernetes for Execution

**PostgreSQL stores:**
- User accounts
- Server metadata
- Server status
- Connection details

**Kubernetes stores:**
- Running pods
- Active resources
- Real-time status

**Why hybrid approach:**
- Database survives K8s cluster restarts
- Audit trail and history
- Enables server management without querying K8s constantly
- Can rebuild K8s state from database if needed

---

### 9. Global Error Handler Middleware

**Why:**
- Consistent error responses
- Single place to log errors
- Prevents error details leaking in production
- Clean controller code (no try-catch everywhere)

---

### 10. Environment Variables for Configuration

**Required variables:**
```env
DATABASE_URL=postgresql://user:password@localhost:5432/kubecraft_db
JWT_SECRET=your-super-secret-key-change-in-production
JWT_EXPIRY=24h
PORT=3000
NODE_ENV=development
K8S_CONFIG_PATH=/home/user/.kube/config
```

**Why:**
- 12-factor app principles
- Different configs per environment (dev, staging, prod)
- Security (secrets not in code)
- Easy to change without redeploying

---

## Next Steps

### Implementation Order:
1. **Setup & Database** (2-3 hours)
   - Initialize project structure
   - Set up PostgreSQL
   - Create database schema

2. **Authentication System** (2-3 hours)
   - User model
   - Auth service (register, login)
   - JWT middleware
   - Auth routes

3. **Kubernetes Client Setup** (1-2 hours)
   - Initialize K8s client
   - Test connection
   - Set up local K3s cluster

4. **Kubernetes Service Layer** (3-4 hours)
   - Create server function
   - Start/stop/delete functions
   - Status query function

5. **Server Management API** (2-3 hours)
   - Server model
   - Server controller
   - Server routes

6. **Testing & Documentation** (2-3 hours)
   - Manual testing flow
   - Create Postman collection
   - Write API documentation

**Total Estimated Time:** 14-20 hours

---

## Testing Strategy

### Manual Testing Checklist:
1. ✅ Register a new user
2. ✅ Login and receive JWT token
3. ✅ Create a Minecraft server
4. ✅ Verify Kubernetes resources created (`kubectl get all -n <namespace>`)
5. ✅ Start the server
6. ✅ Check pod logs (`kubectl logs -n <namespace> <pod>`)
7. ✅ Connect with Minecraft client
8. ✅ Stop the server (verify data persists)
9. ✅ Restart the server (verify world data restored)
10. ✅ Delete the server (verify all resources cleaned up)
11. ✅ Test quota enforcement (try creating 4th server)
12. ✅ Test error cases (invalid token, wrong user, etc.)

### Automated Testing (Future):
- Unit tests for models and services
- Integration tests for API endpoints
- Kubernetes resource creation tests

---

## API Usage Examples

### Complete Flow Example

**1. Register:**
```bash
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "player1",
    "email": "player1@example.com",
    "password": "SecurePass123"
  }'
```

**2. Login:**
```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "player1@example.com",
    "password": "SecurePass123"
  }'

# Response includes token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**3. Create Server:**
```bash
curl -X POST http://localhost:3000/api/servers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "serverName": "my-world",
    "gameMode": "survival",
    "maxPlayers": 20
  }'
```

**4. Start Server:**
```bash
curl -X POST http://localhost:3000/api/servers/1/start \
  -H "Authorization: Bearer <your_token>"
```

**5. Check Status:**
```bash
curl -X GET http://localhost:3000/api/servers/1/status \
  -H "Authorization: Bearer <your_token>"
```

**6. Stop Server:**
```bash
curl -X POST http://localhost:3000/api/servers/1/stop \
  -H "Authorization: Bearer <your_token>"
```

**7. Delete Server:**
```bash
curl -X DELETE http://localhost:3000/api/servers/1 \
  -H "Authorization: Bearer <your_token>"
```

---

## Appendix: TypeScript Interfaces

### User Interface
```typescript
interface IUser {
  id: number;
  username: string;
  email: string;
  passwordHash: string;
  createdAt: Date;
  updatedAt: Date;
}

interface IUserResponse {
  id: number;
  username: string;
  email: string;
  createdAt: Date;
}
```

### Server Interface
```typescript
interface IMinecraftServer {
  id: number;
  userId: number;
  serverName: string;
  namespace: string;
  minecraftVersion: string;
  gameMode: 'survival' | 'creative' | 'adventure' | 'spectator';
  maxPlayers: number;
  difficulty: 'peaceful' | 'easy' | 'normal' | 'hard';
  status: 'stopped' | 'starting' | 'running' | 'stopping' | 'error';
  serverIp: string | null;
  serverPort: number | null;
  createdAt: Date;
  lastStarted: Date | null;
}

interface ICreateServerRequest {
  serverName: string;
  minecraftVersion?: string;
  gameMode?: 'survival' | 'creative' | 'adventure' | 'spectator';
  maxPlayers?: number;
  difficulty?: 'peaceful' | 'easy' | 'normal' | 'hard';
}
```

### API Response Interface
```typescript
interface IAPIResponse<T> {
  success: boolean;
  message?: string;
  data?: T;
  error?: {
    code: string;
    message: string;
    field?: string;
    timestamp: string;
  };
}
```

---

**End of API Design Specification**
