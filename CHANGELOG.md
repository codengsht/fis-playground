# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.2] - 2024-12-13

### Added
- Health check endpoints for monitoring API and database connectivity:
  - `GET /health` - Simple API health check (confirms Lambda is running)
  - `GET /health/db` - DynamoDB connectivity check (verifies database connection)
- Integration tests for health check endpoints

### Changed
- Updated Lambda runtime from deprecated `go1.x` to `provided.al2023` with custom runtime
- Fixed HTTP status codes for update/delete operations on non-existent items (now returns 404 instead of 409)
- Build process now creates `bootstrap` binary for custom runtime compatibility

### Fixed
- `ConditionalCheckFailedException` now correctly maps to 404 Not Found for update/delete operations
- Create operation properly returns 409 Conflict when item already exists

### Technical Details
- Lambda handler changed from `main` to `bootstrap` for `provided.al2023` runtime
- Added `HealthCheck` and `HealthCheckDB` handlers in `internal/handlers/handlers.go`
- Updated CloudFormation template with health endpoint API Gateway resources
- Improved error handling in `internal/repository/errors.go` and `internal/repository/dynamodb.go`

## [0.0.1]

### Added
- CloudFormation deployment configuration file (`deployments/cloudformation/deploy-config.json`)
  - Standardized stack naming convention: `fis-playground-dev`
  - IAM capabilities configuration for Lambda and DynamoDB resources
  - Consistent tagging strategy across all resources
  - Rollback configuration for failed deployments
  - 30-minute deployment timeout setting
  - Termination protection disabled for playground environment

### Changed
- Simplified deployment parameter management
  - Removed environment-specific parameter files (staging, production)
  - Leveraging CloudFormation template default values for Environment parameter
  - Streamlined deployment process for dev-only playground environment

### Removed
- `deployments/cloudformation/parameters.json` - Using template defaults instead
- `deployments/cloudformation/parameters-staging.json` - Not needed for playground
- `deployments/cloudformation/parameters-prod.json` - Not needed for playground

### Technical Details
- CloudFormation template already parameterized with Environment parameter (default: "dev")
- Deploy config includes necessary IAM capabilities: `CAPABILITY_IAM`, `CAPABILITY_NAMED_IAM`
- Consistent resource tagging: Environment, Application, Purpose, ManagedBy, Repository
- Rollback enabled with no monitoring triggers for simple playground setup