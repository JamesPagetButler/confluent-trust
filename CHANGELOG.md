# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Repository scaffolding: package layout, LICENSE (Apache 2.0), README, .golangci.yml.
- GitHub Actions CI: build, vet, race-test, golangci-lint.
- `model/` package with Anchor, Chain, Confluence, Inventory, Programme, Fork types.
- `internal/validate/` package with embedded JSON Schema 2020-12.
- `schema/inventory.schema.json` — canonical inventory schema.
- `testdata/` fixtures: minimal, qbp_v3_2, qbp_quantum_v0_1, qbp_quantum_v0_2.
- Issue / PR templates, CODEOWNERS, SECURITY.md, CONTRIBUTING.md.

[Unreleased]: https://github.com/JamesPagetButler/confluent-trust/commits/main
