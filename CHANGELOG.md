# Changelog

This document outlines major changes between releases.

## [Unreleased]

New features:

Behaviour changes:
 * add generic Hash/Address parameters to `DBFT` service (#94)
 * remove custom payloads implementation from default `DBFT` service configuration
   (#94)
 * rename `InitializeConsensus` dBFT method to `Reset` (#95)
 * drop outdated dBFT `Service` interface (#95)
 * move all default implementations to `internal` package (#97)
 * remove unused APIs of dBFT and payload interfaces (#104)

Improvements:
 * add MIT License (#78, #79)
 * documentation updates (#80, #86, #95)
 * dependencies upgrades (#82, #85)
 * minimum required Go version upgrade to Go 1.19 (#83)
 * log messages adjustment (#88)
 * untie `dbft` module from `github.com/nspcc-dev/neo-go` dependency (#94)
 * minimum required Go version upgrade to Go 1.20 (#100)

Bugs fixed:

## [0.1.0] (15 May 2023)

Stable dbft 2.0 implementation.

[Unreleased]: https://github.com/nspcc-dev/dbft/compare/v0.1.0...master
[0.1.0]: https://github.com/nspcc-dev/dbft/releases/v0.1.0