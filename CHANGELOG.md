# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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