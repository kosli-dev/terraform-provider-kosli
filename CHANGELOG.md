# Changelog

## 0.3.0 (Unreleased)

FEATURES:

* **New Resource**: `kosli_logical_environment` for managing logical environments that aggregate physical environments
* **New Data Source**: `kosli_logical_environment` for querying existing logical environments

IMPROVEMENTS:

* resource/kosli_logical_environment: Added support for aggregating multiple physical environments
* resource/kosli_logical_environment: Added validation to prevent nesting logical environments within logical environments (per ADR-004)
* resource/kosli_logical_environment: Added nil normalization for `included_environments` to ensure consistent state handling
* resource/kosli_logical_environment: Full drift detection support for `included_environments` field
* data_source/kosli_logical_environment: Added type validation to ensure only logical environments are queried
* data_source/kosli_logical_environment: Returns `name`, `type`, `description`, `included_environments`, and `last_modified_at`
* docs: Added comprehensive examples for logical environment resource and data source
* docs: Added ADR-004 documenting validation strategy for logical environments

BUG FIXES:

* data_source/kosli_logical_environment: Fixed acceptance tests to expect correct `included_environments` count after API fix [GH-103]

NOTES:

* Logical environments can only contain physical environments (K8S, ECS, S3, docker, server, lambda), not other logical environments
* The Kosli API now returns `included_environments` in GET responses, enabling full state management and drift detection

## 0.2.0

FEATURES:

* **New Resource**: `kosli_environment` for managing Kosli environments
* **New Data Source**: `kosli_environment` for querying existing environments
* Support for physical environment types: K8S, ECS, S3, docker, server, lambda

IMPROVEMENTS:

* provider: Added requirement for Service Account with Admin permissions for managing resources
* docs: Added comprehensive examples for environment resource and data source
* docs: Added documentation templates for tfplugindocs generation

BUG FIXES:

* resource/kosli_environment: Fixed import test failure where `type` field was not being mapped from API response to state [GH-71]

NOTES:

* Environment support is currently limited to physical environments only. Logical environment support will be added in a future release.

## 0.1.0

FEATURES:

* **New Resource**: `kosli_custom_attestation_type` for managing custom attestation types with JSON Schema validation and JQ evaluation rules
* **New Data Source**: `kosli_custom_attestation_type` for querying existing custom attestation types

IMPROVEMENTS:

* provider: Initial release of the Kosli Terraform Provider
* provider: Added support for authentication via `api_token` (configurable via `KOSLI_API_TOKEN` environment variable)
* provider: Added support for organization configuration via `org` (configurable via `KOSLI_ORG` environment variable)
* provider: Added support for regional API endpoints via `api_url` with defaults for EU (`https://app.kosli.com`) and US (`https://app.us.kosli.com`) regions (configurable via `KOSLI_API_URL` environment variable)
* provider: Added configurable HTTP client timeout with 30 second default
* resource/kosli_custom_attestation_type: Added support for JSON Schema definitions via `schema` attribute with semantic equality comparison
* resource/kosli_custom_attestation_type: Added support for JQ evaluation rules via `jq_rules` attribute
* resource/kosli_custom_attestation_type: Added optional `description` attribute for documentation
* resource/kosli_custom_attestation_type: Added resource import support by name
* resource/kosli_custom_attestation_type: Updates create new versions of attestation types
* data_source/kosli_custom_attestation_type: Query custom attestation types by `name`
* data_source/kosli_custom_attestation_type: Returns `description`, `schema`, `jq_rules`, and `archived` status

BUG FIXES:

* resource/kosli_custom_attestation_type: Fixed "provider produced invalid plan" error for JSON Schema attribute by implementing semantic equality - schema formatting differences (whitespace, quote style) no longer trigger unnecessary updates [GH-35]
