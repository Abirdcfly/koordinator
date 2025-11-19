# Koordinator Project Guide for AI Agents

## Project Overview

Koordinator is a QoS-based scheduling system for hybrid orchestration workloads on Kubernetes. It aims to improve runtime efficiency and reliability of both latency-sensitive workloads and batch jobs, simplify resource-related configuration tuning, and increase pod deployment density to improve resource utilization.

**Key Technologies:**
- **Language:** Go 1.21
- **Platform:** Kubernetes (v1.28.7)
- **Architecture:** Cloud-native, microservices-based
- **License:** Apache License 2.0

## Project Structure

### Core Components

The project consists of five main binaries located in `/cmd/`:

1. **koordlet** - Node agent that runs on each Kubernetes node
   - Path: `/cmd/koordlet/`
   - Purpose: Resource management, QoS enforcement, metrics collection
   - Dependencies: Requires libpfm for performance monitoring

2. **koord-manager** - Central controller manager
   - Path: `/cmd/koord-manager/`
   - Purpose: CRD management, webhook handling, cluster-level coordination

3. **koord-scheduler** - Enhanced Kubernetes scheduler
   - Path: `/cmd/koord-scheduler/`
   - Purpose: Advanced scheduling with QoS awareness, co-scheduling, device sharing

4. **koord-descheduler** - Workload descheduler
   - Path: `/cmd/koord-descheduler/`
   - Purpose: Pod migration, resource rebalancing, interference mitigation

5. **koord-device-daemon** - Device management daemon
   - Path: `/cmd/koord-device-daemon/`
   - Purpose: GPU and other device resource management

6. **koord-runtime-proxy** - Container runtime proxy
   - Path: `/cmd/koord-runtime-proxy/`
   - Purpose: Runtime hook injection for resource control

### Package Organization

Main packages in `/pkg/`:
- **scheduler/** - Scheduling plugins and frameworks
- **koordlet/** - Node agent implementation
- **descheduler/** - Descheduling logic and strategies
- **slo-controller/** - SLO (Service Level Objectives) management
- **webhook/** - Admission webhooks for resource validation
- **client/** - Generated Kubernetes clients
- **util/** - Common utilities and helpers
- **runtimeproxy/** - Runtime proxy implementation
- **device-daemon/** - Device management logic

### API Definitions

Custom Resource Definitions (CRDs) in `/apis/`:
- **extension/** - Extended scheduling and resource APIs
- **scheduling/** - Scheduling-related CRDs
- **slo/** - SLO and QoS configuration APIs
- **runtime/** - Runtime configuration APIs
- **configuration/** - System configuration APIs

## Build and Development

### Prerequisites
- Go 1.21+
- Docker (for container builds)
- libpfm4 and libpfm4-dev (for performance monitoring)
- kubectl and access to a Kubernetes cluster

### Build Commands

```bash
# Build all components
make build

# Build individual components
make build-koordlet
make build-koord-manager
make build-koord-scheduler
make build-koord-descheduler
make build-koord-device-daemon

# Build Docker images
make docker-build

# Run tests
make test

# Fast test (skip some checks)
make fast-test
```

### Code Generation

```bash
# Generate CRDs and RBAC manifests
make manifests

# Generate DeepCopy methods and clients
make generate

# Update license headers
make lint-license
```

### Development Tools

```bash
# Install development dependencies
make kustomize
make controller-gen
make envtest
make golangci-lint

# Format code
make fmt

# Run linting
make lint

# Run vet
make vet
```

## Testing Strategy

### Unit Tests
- Located alongside source files (`*_test.go`)
- Run with: `make test` or `make fast-test`
- Uses Ginkgo and Gomega for BDD-style testing
- Coverage reports generated in `cover.out`

### Integration Tests
- Located in `/test/e2e/`
- Framework-based testing with Kubernetes clusters
- Multiple Kubernetes version support (1.22, 1.28, 1.32, latest)
- Run with: `go test ./test/e2e/`

### Test Configuration
- Uses envtest for controller testing
- Kubernetes version: 1.28 (configurable via ENVTEST_K8S_VERSION)
- Agent mode: hostMode (configurable via AGENT_MODE)

## Code Style Guidelines

### Go Code Standards
- Follows standard Go formatting (`gofmt`)
- Uses `goimports` with local prefix `github.com/koordinator-sh/koordinator`
- Linting with golangci-lint v1.55.2
- Enabled linters: gofmt, govet, goimports, ineffassign, misspell, vet, unconvert, staticcheck, depguard

### Import Organization
- Standard library imports first
- External imports second
- Internal imports last with local prefix

### Deprecation Rules
- Avoid `io/ioutil` - use `os` or `io` functions instead
- Follow Kubernetes deprecation patterns

## Deployment and Configuration

### Container Images
- Registry: `ghcr.io/koordinator-sh/`
- Image naming: `{component}:{branch}-{commit}`
- Multi-architecture support (linux/amd64, linux/arm64)

### Kubernetes Deployment
- Uses Kustomize for configuration management
- Base configurations in `/config/`
- RBAC configurations in `/config/rbac/`
- CRD definitions in `/config/crd/`

### Deployment Commands
```bash
# Install CRDs
make install

# Deploy to cluster
make deploy

# Undeploy from cluster
make undeploy
```

## Security Considerations

### Security Policy
- Report vulnerabilities to: kubernetes-security@service.aliyun.com
- Follow embargo policy for security issues
- Regular dependency updates via Dependabot

### RBAC Configuration
- Minimal required permissions principle
- Separate roles for each component
- Service accounts with specific permissions

### Runtime Security
- Non-root container execution where possible
- Resource limits and requests defined
- Security contexts configured

## Development Workflow

### Contributing Process
1. Fork the repository
2. Create feature branch from main
3. Make changes following code standards
4. Run tests and ensure they pass
5. Submit pull request with proper description

### CI/CD Pipeline
- GitHub Actions for continuous integration
- Multiple Kubernetes version testing
- Automated code quality checks
- Container image building and pushing

### Release Process
- Semantic versioning
- Release branches for maintenance
- Automated release notes generation
- Container image publishing to GHCR

## Key Dependencies

### Kubernetes Ecosystem
- Kubernetes v1.28.7 (client-go, apimachinery, etc.)
- Controller-runtime v0.16.5
- Kustomize for configuration management

### Monitoring and Metrics
- Prometheus client libraries
- Performance monitoring with libpfm
- Custom metrics exporters

### Container Runtime
- Containerd integration
- CRI (Container Runtime Interface) support
- Runtime hook mechanisms

## Performance Considerations

### Resource Management
- CPU and memory resource controls
- NUMA awareness and optimization
- Device resource allocation (GPU, etc.)

### Scheduling Optimization
- QoS-aware scheduling algorithms
- Co-scheduling for related workloads
- Resource fragmentation minimization

### Monitoring Integration
- Real-time performance metrics
- Resource utilization tracking
- Interference detection and mitigation

## Troubleshooting

### Common Issues
- libpfm dependency issues: Install libpfm4 and libpfm4-dev
- Permission issues: Check RBAC configurations
- Scheduling conflicts: Review priority and QoS settings

### Debug Tools
- Enable pprof endpoints for performance analysis
- Use kubectl logs for component debugging
- Check events and conditions in Kubernetes resources

### Support Channels
- GitHub Issues for bug reports
- Slack channel: #koordinator in Kubernetes workspace
- Community meetings (bi-weekly, APAC timezone)