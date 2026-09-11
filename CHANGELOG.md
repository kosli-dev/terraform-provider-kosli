# Changelog

## v0.9.4 (September 11, 2026)

BUG FIXES:

* resource/kosli_service_account_api_key: Fixed `expires_at` producing an inconsistent result after apply by aligning the provider with the server's 365-day cap on API key expiry, including the "never expires" case [GH-258]

NOTES:

* resource/kosli_service_account_api_key: `expires_at` is now capped at 365 days by the server; omitting it yields the maximum 365-day expiry rather than a non-expiring key [GH-258]

## 0.9.3 (August 18, 2026)

FEATURES:

* data_source/kosli_custom_attestation_type: Added `summary` attribute exposing the ordered, labelled jq expressions rendered as rows on the attestation detail page [GH-243]
* resource/kosli_custom_attestation_type: Added optional `summary` attribute for defining ordered, labelled jq expressions rendered as rows on the attestation detail page, with URL values rendered as links [GH-243]

BUG FIXES:

* data_source/kosli_custom_attestation_type: Fixed `schema` being read as an empty string instead of `null` for attestation types defined without a schema, which caused a "not valid JSON" error [GH-243]

## 0.9.2 (July 30, 2026)

BREAKING CHANGES:

* data_source/kosli_environment: Removed the `include_scaling` attribute; the Kosli API no longer honors it and always reports it as `false` [GH-236]
* resource/kosli_environment: Removed the `include_scaling` attribute; the Kosli API no longer honors it, so setting it to `true` caused apply failures [GH-236]

## 0.9.1 (July 28, 2026)

NOTES:

* No user-facing changes in this release.

## 0.9.0 (July 15, 2026)

NOTES:

* No user-facing changes in this release. (internal refactoring and optimisation)

## 0.8.1 (July 10, 2026)

FEATURES:

* **New List Resource**: `kosli_control` for discovering existing Kosli Controls via `terraform query` (requires Terraform >= 1.14)

IMPROVEMENTS:

* list_resource/kosli_control: Added `search` and `archived` filters to the list configuration for scoping query results
* list_resource/kosli_control: Surfaces the Controls beta feature-flag hint on `403` responses, matching the `kosli_control` resource
* docs: Added a `.tfquery.hcl` example and generated documentation for the new `kosli_control` list resource

## 0.8.0 (July 8, 2026)

FEATURES:

* **New Resource**: `kosli_control` for managing Kosli Controls (beta)
* **New Data Source**: `kosli_control` for querying existing Kosli Controls (beta)

IMPROVEMENTS:

* client: Added a `Control` client supporting create/get/list/update/archive of Kosli Controls
* resource/kosli_control: Added `identifier` (immutable), `name`, `description`, and `links` attributes, plus computed `version`, `created_at`, `created_by`, `tags`, and `policies_referencing`; supports import by `identifier`
* resource/kosli_control: Archived controls are treated as deleted for drift detection, with diagnostics hinting at the Controls beta feature flag on `403` responses and at archived identifier conflicts on `409` responses
* data_source/kosli_control: Added lookup of existing controls by `identifier`
* docs: Added examples and generated documentation for the new `kosli_control` resource and data source

## 0.7.0 (July 7, 2026)

FEATURES:

* **New Resource**: `kosli_service_account` for managing Kosli service accounts
* **New Resource**: `kosli_service_account_api_key` for managing service account API keys
* **New Data Source**: `kosli_service_account` for querying existing service accounts

IMPROVEMENTS:

* client: Added `ServiceAccount` and `ServiceAccountAPIKey` clients supporting create/get/list/update/delete of service accounts and create/list/revoke of their API keys
* resource/kosli_service_account: Added `name`, `description`, and `privilege` attributes with import support by `name`
* resource/kosli_service_account_api_key: Added support for minting and revoking API keys; the raw key is a write-once sensitive value preserved across reads, and all arguments force replacement since keys are immutable
* data_source/kosli_service_account: Added lookup of existing service accounts by `name`
* docs: Added examples and generated documentation for the new service account resources and data source

## 0.6.4 (May 11, 2026)

IMPROVEMENTS:

* provider: Bumped Go version to 1.26

## 0.6.3 (April 29, 2026)

BUG FIXES:

* resource/kosli_environment: Fixed race condition where a parallel destroy+create (triggered by a Terraform resource label rename) could cause the post-create read to receive a 404 "has been archived" error. The provider now retries the create PUT and subsequent GET with bounded backoff when a 404 is encountered [GH-121]

NOTES:

* If you are intentionally renaming a Terraform resource label for a Kosli environment (while keeping the same `name`), the recommended approach is `terraform state mv`.

## 0.6.2 (April 29, 2026)

IMPROVEMENTS:

* docs: Updated policy schema URL from `kosli.com/schemas/policy/environment/v1` to `docs.kosli.com/schemas/policy/v1` across all examples and documentation [GH-183]

BUG FIXES:

* resource/kosli_environment: Fixed clearing environment description by switching from PUT to PATCH endpoint - empty string descriptions are now correctly applied [GH-186]
* resource/kosli_logical_environment: Fixed clearing environment description by switching from PUT to PATCH endpoint - empty string descriptions are now correctly applied [GH-186]

## 0.6.1 (April 21, 2026)

IMPROVEMENTS:

* ci: Expanded PR validation to cover all 13 examples, matching the main pipeline and preventing broken examples from slipping through review [GH-177]
* ci: Added Slack notification on release to improve release visibility [GH-181]

BUG FIXES:

* client: Bumped `hc-install` to v0.9.4 to resolve `openpgp: key expired` errors caused by an expired HashiCorp GPG key that was breaking acceptance tests [GH-180]

NOTES:

* Dependency: Updated `kosli-dev/setup-cli-action` from v3 to v4; note that omitting the `version` input now defaults to `latest` [GH-179]

## 0.6.0 (April 15, 2026)

FEATURES:

* resource/kosli_environment: Added `tags` attribute for managing resource tags via dedicated Kosli tags API endpoint
* data_source/kosli_environment: Added `tags` attribute for reading resource tags
* resource/kosli_logical_environment: Added `tags` attribute for managing resource tags
* data_source/kosli_logical_environment: Added `tags` attribute for reading resource tags
* resource/kosli_flow: Added `tags` attribute for managing resource tags
* data_source/kosli_flow: Added `tags` attribute for reading resource tags

IMPROVEMENTS:

* client: Added `Patch()` HTTP method to support tag management via dedicated API endpoint
* client: Added `TagResource()` client method for applying tag diffs (`set_tags`/`remove_tags`) via `PATCH /api/v2/tags/{org}/{resourceType}/{name}`
* resource/kosli_environment: Tags are applied as a diff (only changed tags are sent to the API) to avoid unnecessary updates
* resource/kosli_flow: Generalized `applyTags` helper to accept a `resourceType` parameter, enabling reuse across environment, logical environment, and flow resources
* resource/kosli_logical_environment: Introduced `mapLogicalEnvToState()` to consolidate state mapping and normalize nil tags to an empty map, preventing state drift
* resource/kosli_flow: Nil API tags normalized to empty map in `mapFlowToModel` to prevent drift when `tags = {}` is set in config
* docs: Updated schema documentation and examples for environment, logical environment, and flow resources to include tags usage

## 0.5.0 (April 1, 2026)

FEATURES:

* **New Resource**: `kosli_flow` for managing Kosli flows with full CRUD lifecycle support, including `name`, `description`, and `template` attributes
* **New Data Source**: `kosli_flow` for querying existing Kosli flows by name, exposing `name`, `description`, and `template` attributes

IMPROVEMENTS:

* docs: Added documentation and examples for `kosli_flow` resource and data source

BUG FIXES:

* ci: Fixed CI workflows when adding new Terraform resources to use local provider build instead of released version [GH-166]
* ci: Fixed GitHub Actions bot email to produce verified commits on GHA-created PRs [GH-164]

## 0.4.2 (March 26, 2026)

IMPROVEMENTS:

* client: Simplified `type_schema` handling by changing `TypeSchema` from `string` to `json.RawMessage` to accept proper JSON objects returned by the Kosli API
* client: Removed Python repr normalization workaround (`normalizePythonToJSON()`) after server-side fix

BUG FIXES:

* resource/kosli_custom_attestation_type: Fixed unmarshalling error caused by API returning `type_schema` as a JSON object instead of a Python repr() string [GH-160]
* data_source/kosli_custom_attestation_type: Fixed unmarshalling error caused by API returning `type_schema` as a JSON object instead of a Python repr() string [GH-160]

## 0.4.0

FEATURES:

* **New Resource**: `kosli_action` for managing webhook notification actions triggered by environment compliance events
* **New Data Source**: `kosli_action` for querying existing webhook notification actions
* **New Resource**: `kosli_policy` for managing Kosli policies
* **New Data Source**: `kosli_policy` for querying existing Kosli policies
* **New Resource**: `kosli_policy_attachment` for attaching policies to Kosli resources

IMPROVEMENTS:

* client: Added Actions API client methods for CRUD operations on webhook notification actions
* docs: Updated README with new resources and data sources

BUG FIXES:

* resource/kosli_action: Fixed failing acceptance tests [GH-132]

## 0.3.1

IMPROVEMENTS:

* client: Extracted Python-to-JSON normalization to dedicated `normalizePythonToJSON()` function for better maintainability
* client: Improved regex-based conversion to handle Python literals (`True`/`False`/`None`) in all contexts (object properties, arrays, nested structures)
* client: Optimized performance by moving regex compilation to package-level variables
* docs: Added "Known Issues" section to README documenting Python keyword conversion limitation

BUG FIXES:

* resource/kosli_custom_attestation_type: Fixed handling of Python boolean and null values in schemas returned by the Kosli API [GH-106]
* data_source/kosli_custom_attestation_type: Fixed handling of Python boolean and null values in schemas returned by the Kosli API [GH-106]

NOTES:

* The Kosli API returns `type_schema` in Python `repr()` format instead of valid JSON. The provider automatically normalizes this to JSON format. See README "Known Issues" section for details.

## 0.3.0

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
* resource/kosli_logical_environment: Fixed `type` attribute to show value during plan instead of "(known after apply)" [GH-105]

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
