# Changelog

## 0.1.0 (Unreleased)

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
