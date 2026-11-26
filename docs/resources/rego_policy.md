---
page_title: "kosli_rego_policy Resource - terraform-provider-kosli"
subcategory: ""
description: |-
  Manages a Kosli rego policy: a versioned bundle of Rego files that Kosli evaluates trails against. Each change to files or input_schema publishes a new version, and the apply waits until Kosli has validated it.
  ~> Note: Rego policies are part of the Controls beta feature, which must be enabled for your organization; API requests return 403 Forbidden otherwise.
  ~> Note: Publishing requires a service account with Admin permissions.
  -> Deleting this resource archives the policy in Kosli rather than hard-deleting it: the policy leaves the policy list, while its versions stay readable and an evaluation by its identifier still runs. Creating the resource again with the same identifier publishes to the archived policy and unarchives it.
---

# Resource: kosli_rego_policy

Manages a Kosli rego policy: a versioned bundle of Rego files that Kosli evaluates trails against. Each change to `files` or `input_schema` publishes a new version, and the apply waits until Kosli has validated it.

~> **Note:** Rego policies are part of the Controls **beta** feature, which must be enabled for your organization; API requests return `403 Forbidden` otherwise.

~> **Note:** Publishing requires a service account with **Admin** permissions.

-> Deleting this resource **archives** the policy in Kosli rather than hard-deleting it: the policy leaves the policy list, while its versions stay readable and an evaluation by its identifier still runs. Creating the resource again with the same `identifier` publishes to the archived policy and unarchives it.

## Example Usage

```terraform
terraform {
  required_providers {
    kosli = {
      source = "kosli-dev/kosli"
    }
  }
}

# A single-file policy, read from a .rego file kept next to the configuration
resource "kosli_rego_policy" "artifact_provenance" {
  identifier   = "artifact-provenance"
  name         = "Artifact provenance"
  description  = "Requires the trail to hold at least one artifact"
  input_schema = "trail"
  comment      = "Initial version"

  files = {
    "policy.rego" = file("${path.module}/policies/artifact_provenance.rego")
  }
}

# A bundle of every .rego file in a directory, keyed by its path in the bundle
resource "kosli_rego_policy" "release_checks" {
  identifier   = "release-checks"
  name         = "Release checks"
  input_schema = "trail"

  files = {
    for name in fileset("${path.module}/policies", "**/*.rego") :
    name => file("${path.module}/policies/${name}")
  }
}

output "artifact_provenance_digest" {
  description = "Digest of the version an evaluation runs"
  value       = kosli_rego_policy.artifact_provenance.digest
}
```

## Versions and Validation

Kosli stores a policy as a series of immutable versions. Publishing returns a version in status `pending`, and Kosli validates it asynchronously:

| Status | Meaning | What the apply does |
|---|---|---|
| `valid` | Kosli accepted the bundle | Succeeds; the version becomes the policy's current version |
| `rejected` | The policy is at fault, for example a Rego syntax error | Fails with the validator's message |
| `failed` | Kosli could not validate the version | Fails; apply again to publish the files for another validation |
| `pending` | Validation has not finished | Waits up to 5 minutes, then fails; apply again to keep waiting |

Only a change to `files` or `input_schema` creates a new version. A change to `name` or `description` updates the policy in place, and a change to `comment` alone publishes nothing.

The resource tracks the policy's **current version**: the highest-numbered valid version, which is the one an evaluation runs. After an apply that ends with a rejected version, the policy keeps running its previous valid version and the next plan still shows the pending change.

A version published outside Terraform shows up as a change to `files` on the next plan, once it is valid.

## Destroying

Kosli never hard-deletes a rego policy. `terraform destroy`, or removing a `kosli_rego_policy` from the configuration, **archives** the policy:

- It leaves the default policy list in Kosli.
- Its versions, files and past evaluations stay readable.
- An evaluation by its identifier still runs.
- Its identifier stays taken.

Creating a `kosli_rego_policy` with the identifier of an archived policy publishes to that policy and unarchives it, so the version numbers continue from where they stopped.

A policy archived outside Terraform reads as deleted: the next plan proposes to create it, and the apply unarchives it.

~> **Note:** Archiving requires a Kosli server with the rego policy archive API. On an older server the destroy removes the resource from the Terraform state, leaves the policy active, and says so in a warning.

## Import

Rego policies can be imported using their identifier:

```shell
# Import an existing rego policy by identifier
terraform import kosli_rego_policy.artifact_provenance artifact-provenance
```

The import reads the files of the current version into the state. `comment` is not read back from Kosli. An archived policy cannot be imported; unarchive it in Kosli first.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `files` (Map of String) The Rego bundle, as a map of each file's relative path to its source. A single-file policy is one entry. Kosli limits a bundle to 100 files and 1 MiB in total, and rejects paths that leave the bundle.
- `identifier` (String) Unique identifier of the policy within the organization. Must start with a letter or number and contain only letters, numbers, periods (`.`), hyphens (`-`), underscores (`_`), and tildes (`~`). Kosli compares identifiers without regard to case. Changing this to a different policy will force recreation of the resource.
- `input_schema` (String) Shape of the input the policy is written against. Known values: `trail` (the policy evaluates one trail) and `trails` (a set of trails). The Kosli API validates the value.
- `name` (String) Human-readable display name of the policy. Can be changed without publishing a new version.

### Optional

- `comment` (String) Why this version is published. Kosli stores the comment with the version an apply creates; changing only the comment publishes nothing, and the value is not read back from Kosli.
- `description` (String) Free-form description of the policy. Can be changed without publishing a new version.

### Read-Only

- `created_at` (String) RFC3339 UTC timestamp of when the policy was created.
- `created_by` (String) Display name of the user who created the policy.
- `digest` (String) Digest of the current version's files, as `sha256:` and a hex digest. A decision made with the policy cites this value.
- `entrypoint` (String) Policy package Kosli found as the entrypoint of the current version.
- `params_schema` (String) JSON Schema, as a JSON string, that the current version declares for its params. Null when the policy declares none.
- `version` (Number) Number of the policy's current version: the highest-numbered valid version, which is the one an evaluation runs.
