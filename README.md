# Issue Tracker Service

A gRPC-based issue tracking application designed for efficient management of projects and issues. Features both **HashiCorp MemDB** for high-performance in-memory storage, **PostgreSQL** for persistent data storage, and **Kafka** for messaging between services.

---

## Project Overview

The Issue Tracker provides a robust backend service for managing software development projects and their associated issues. Built using Go, gRPC, **HashiCorp MemDB**, **PostgreSQL**, **Redis**, and **Kafka**, it delivers blazing-fast storage and retrieval operations with structured data schemas and transaction support, along with the durability of a relational database and reliable messaging.

### Features

- **Project Management**:
  - Create, read, update, delete, list projects with persistent storage.
  - Real-time project updates through streaming.
- **Issue Tracking**:
  - Associate issues with projects, assign users, update statuses.
  - Comprehensive issue lifecycle management (creation, updates, resolution).
  - Custom issue types, priorities, and resolution states.
- **User Management**:
  - User CRUD operations with support for v1 and v2 APIs.
  - Authentication and user profile management.
- **Dual Storage Strategy**:
  - **HashiCorp MemDB**: For fast, in-memory operations and rapid prototyping.
  - **PostgreSQL Database**: For persistent, durable storage of all entities.
  - **Redis Cache**: For rapid data access and caching frequently used entities.
- **Messaging Architecture**:
  - **Kafka**: For reliable message delivery between services and real-time updates.
  - **In-Memory Stream**: Option for simplified development environments.
- **gRPC API with REST Gateway**:
  - Efficient and type-safe communication through gRPC.
  - REST compatibility via HTTP gateway for broader client support.

---

## Prerequisites

- **Go** (version 1.16+)
- **Protocol Buffers compiler (`protoc`)**
- Go plugins for Protocol Buffers:
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`
  - `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`

---

## Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd issue-tracker-service
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

---

## Building the Application

Using the Makefile:
```bash
make build
```

Or directly with Go:
```bash
go build -o bin/server cmd/main.go
```

---

## Running the Application

Using the Makefile:
```bash
make run
```

Or directly:
```bash
./bin/server
```

By default, the server will run on the following ports:
- **gRPC Server**: `50052`
- **HTTP/REST Gateway**: `8080`

---

## Project Structure

```
issue-tracker-service/
├── cmd/
│   └── main.go         # Server entry point
├── pkg/
│   ├── pb/             # Generated Protocol Buffer files
│   │   ├── user/       # User service (v1, v2)
│   │   ├── project/    # Project service
│   │   └── issues/     # Issues service
│   ├── svc/            # Service implementations
│   │   ├── usersvc/    # User service implementation
│   │   ├── projectsvc/ # Project service implementation
│   │   └── issuessvc/  # Issues service implementation
│   ├── seed/           # Seeding functionality for test data
│   └── consts/         # Common constants and errors
├── models/             # Database models for PostgreSQL
├── database/           # PostgreSQL database initialization and connections
├── proto/              # Protocol Buffer definitions and validation
├── google/             # Google API protobuf definitions
├── logger/             # Custom logging utilities
└── mocks/              # Mock implementations for testing
```

---

## Integration with HashiCorp MemDB

### Why MemDB?
HashiCorp MemDB is used as the in-memory storage backend because it offers:
- **Schema-based Design**:
  - Define structured tables for projects, issues, and users.
- **Transactional Support**:
  - Ensure consistent reads and writes.
- **High Performance**:
  - Optimized for transient and lightweight data workloads.
  
### MemDB Schema
The project leverages MemDB tables such as:
- `Users`: Stores user profiles with unique IDs and relevant attributes.
- `Projects`: Tracks development projects with associated metadata.
- `Issues`: Manages issues tied to projects and optionally assigns users.

---

## Generating Protocol Buffer Files

Using the Makefile:
```bash
make proto
```

Or manually:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/user/v1/user.proto proto/project/v1/project.proto proto/issue/v1/issue.proto
```

---

## API Documentation (gRPC Overview)

### User Service

- `CreateUser`: Creates a new user with name and email.
- `ListUsers`: Retrieves all users.
- `GetUser`: Fetches user details by ID.
- Other CRUD operations for user management.

### Project Service

- `CreateProject`: Creates a new project with name and description.
- `ListProjects`: Retrieves all projects.
- `StreamProjectUpdates`: Provides real-time updates on project changes.
- Other CRUD operations for project management.

### Issue Service

- `CreateIssue`: Creates a new issue associated with a project.
- `ListIssues`: Retrieves all issues by project ID or other filters.
- Other CRUD operations for issue tracking.

---

## Seeding Test Data

During non-production environments, the application supports automatic data seeding for testing purposes:
- **User Seeding**:
  - Controlled via the `SEED_USER_COUNT` environment variable (default is 5 users).
- **Project Seeding**:
  - Controlled via the `SEED_PROJECT_COUNT` environment variable (default is 5 projects).
- **Relationships** (_Optional_):
  - Automatically assigns users to projects and creates example issues.

To enable relationship seeding, set:
```bash
SEED_RELATIONSHIPS=true
```

---

## Using with a gRPC Client

You can interact with the service using any gRPC client. Here are some options:

### Tools
1. Command line tools like [grpcurl](https://github.com/fullstorydev/grpcurl).
2. GUI tools like [Postman](https://www.postman.com/) or [BloomRPC](https://github.com/uw-labs/bloomrpc).

### Example Interaction via `grpcurl`
```bash
grpcurl -d '{"name": "Test User", "email": "test@example.com"}' \
        -plaintext localhost:50051 userpb.UserService.CreateUser
```

---

## Configuration Options

The application can be customized via environment variables:

| Variable               | Description                                                              | Default Value      |
|------------------------|--------------------------------------------------------------------------|--------------------|
| `GRPC_PORT`            | Port for the gRPC server                                                | `50052`            |
| `HTTP_PORT`            | Port for the REST gateway                                               | `8080`             |
| `ENVIRONMENT`          | Application environment (`production`, `development`)                  | `development`      |
| `DB_TYPE`              | Database type (`postgres`, `memdb`)                                     | `memdb`            |
| `POSTGRES_HOST`        | PostgreSQL host                                                         | `localhost`        |
| `POSTGRES_PORT`        | PostgreSQL port                                                         | `5432`             |
| `POSTGRES_USER`        | PostgreSQL username                                                     | `postgres`         |
| `POSTGRES_PASSWORD`    | PostgreSQL password                                                     | `postgres`         |
| `POSTGRES_DB`          | PostgreSQL database name                                               | `issue_tracker`    |
| `CACHE_TYPE`           | Cache implementation (`memory`, `redis`)                               | `memory`           |
| `REDIS_ADDR`           | Redis address                                                           | `localhost:6379`   |
| `COMMUNICATION_METHOD` | Messaging implementation (`stream`, `kafka`)                           | `stream`           |
| `KAFKA_BROKERS`        | Comma-separated list of Kafka brokers                                  | `localhost:9092`   |
| `KAFKA_TOPIC_PREFIX`   | Prefix for Kafka topics                                                | `issue-tracker`    |
| `SEED_USER_COUNT`      | Number of users to create during seeding                                | `5`                |
| `SEED_PROJECT_COUNT`   | Number of projects to create during seeding                             | `5`                |
| `SEED_RELATIONSHIPS`   | Enable creation of relationships between seeded entities (`true/false`) | `false`            |

---

## Running with Docker Compose

The easiest way to run the application with all its dependencies is using Docker Compose:

```bash
docker-compose up -d
```

This will start:
- The Issue Tracker service
- PostgreSQL database
- Redis cache
- Kafka and Zookeeper for messaging
- Kafdrop for Kafka monitoring

Access the services at:
- **gRPC Server**: `localhost:50052`
- **HTTP/REST Gateway**: `localhost:8080`
- **Kafdrop (Kafka UI)**: `localhost:9000`

---

## Testing

### Running Unit Tests
Run unit tests for repositories and services with verbose output:
```bash
go test -v ./...
```

For focused testing of specific packages:
```bash
go test -v ./pkg/svc/usersvc
go test -v ./pkg/svc/projectsvc
go test -v ./pkg/svc/issuessvc
```

### Test Coverage
Generate test coverage reports to identify untested code:
```bash
go test -cover ./...
# For detailed HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests
End-to-end tests for gRPC services with MemDB and Redis interactions:
```bash
go test -tags=integration ./tests/
```

### Mock Data Testing
Test the application with automatically seeded data:
```bash
# Basic seeding
ENVIRONMENT=development go run ./cmd

# Custom seeding configuration
ENVIRONMENT=development SEED_USER_COUNT=10 SEED_PROJECT_COUNT=10 SEED_RELATIONSHIPS=true go run ./cmd
```

### Redis Integration Testing
To test Redis-specific functionality:
```bash
# Start Redis if not running
docker run -d --name redis-test -p 6379:6379 redis:alpine

# Run tests with Redis connection
REDIS_ENABLED=true REDIS_URL=localhost:6379 go test -v ./pkg/svc/...
```

---

## Future Improvements

1. **Enhanced Redis Integration**:
   - Implement full Redis caching layer for frequently accessed entities
   - Add Redis Pub/Sub for real-time notifications across multiple instances

2. **WebSocket Support**:
   - Extend `StreamProjectUpdates` to use WebSockets for real-time data updates
   - Implement browser-compatible event streaming

3. **Advanced Metrics & Monitoring**:
   - Integrate Prometheus metrics for service performance tracking
   - Add Grafana dashboards for visualizing MemDB and Redis performance

4. **Scalability Enhancements**:
   - Implement sharding strategies for MemDB with Redis coordination
   - Add distributed tracing with OpenTelemetry
   - Scale Kafka consumers for higher throughput

5. **Developer Experience**:
   - Create interactive API documentation with Swagger UI
   - Add CLI tools for common administrative tasks

---