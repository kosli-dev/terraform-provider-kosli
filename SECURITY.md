# Security Policy

## Supported Versions

The provider is pre-1.0, so security fixes ship only in the latest release. Please upgrade to the [latest release](https://github.com/kosli-dev/terraform-provider-kosli/releases/latest) and check that the issue still reproduces before reporting it.

## Reporting a Vulnerability

Please email reports to security@kosli.com rather than opening a public issue or pull request. This repository is public, so anything raised here is disclosed to everyone at the moment it reaches us. Give us reasonable time to ship a fix before you disclose publicly.

Please include:
* A clear description of the vulnerability and the affected provider version
* Steps to reproduce, ideally a minimal Terraform configuration
* The potential impact
* Any supporting evidence, such as logs or a proof of concept

Before you attach configurations, logs or state, remove API tokens and webhook URLs. Terraform state and `TF_LOG` output can contain these values in plain text.

## Scope

In scope:
* The provider's resources, data sources and list resources (`internal/provider`)
* The Go API client (`pkg/client`)
* Release artifacts and their signatures, and the workflows that build them

Report vulnerabilities in the Kosli platform itself (app.kosli.com and its APIs) through the platform programme described below, not against this provider.

## Bug Bounty

Kosli runs a paid bug bounty for its production platform, **app.kosli.com** and its supporting APIs, and for **www.kosli.com**. The scope, testing guidelines, eligibility criteria and bounty rates are published at:

https://www.kosli.com/vulnerability-disclosure

This provider is not part of that programme, so reports against it do not qualify for payment. They are still welcome. We will acknowledge your report, assess it, tell you our determination and reasoning, fix what needs fixing, and credit you in the release notes if you would like.

## Verifying Releases

Each release publishes a `terraform-provider-kosli_<version>_SHA256SUMS` file and a detached GPG signature of it. Terraform verifies both when it installs the provider from the registry. To check a downloaded archive by hand, verify the signature on the checksum file, then verify the archive's checksum against it.

## Our Commitment

Integrity is a core value at Kosli. We work fairly and openly with security researchers and communicate transparently throughout disclosure. We will acknowledge receipt, assess the finding and respond with our determination. If we classify the vulnerability differently than reported, we will explain why.
