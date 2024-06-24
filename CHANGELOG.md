# Changelog

This document outlines major changes between releases.

## [Unreleased]

New features:
 * TLA+ model for MEV-resistant dBFT extension (#116)

Behaviour changes:
 * simplify PublicKey interface (#114)
 * remove WithKeyPair callback from dBFT (#114)

Improvements:

Bugs fixed:

## [0.2.0] (01 April 2024)

We're rolling out an update for dBFT that contains a substantial library interface
refactoring. Starting from this version dBFT is shipped as a generic package with
a wide range of generic interfaces, callbacks and parameters. No default payload
implementations are supplied anymore, the library itself works only with payload
interfaces, and thus users are expected to implement the minimum required set of
payload interfaces by themselves. A lot of outdated and unused APIs were removed,
some of the internal APIs were renamed, so that the resulting library interface
is much more clear and lightweight. Also, the minimum required Go version was
upgraded to Go 1.20.

Please note that no consensus-level behaviour changes introduced, this release
focuses only on the library APIs improvement, so it shouldn't be hard for the users
to migrate to the new interface.

Behaviour changes:
 * add generic Hash/Address parameters to `DBFT` service (#94)
 * remove custom payloads implementation from default `DBFT` service configuration
   (#94)
 * rename `InitializeConsensus` dBFT method to `Reset` (#95)
 * drop outdated dBFT `Service` interface (#95)
 * move all default implementations to `internal` package (#97)
 * remove unused APIs of dBFT and payload interfaces (#104)
 * timer interface refactoring (#105)
 * constructor returns some meaningful error on failed dBFT instance creation (#107)

Improvements:
 * add MIT License (#78, #79)
 * documentation updates (#80, #86, #95)
 * dependencies upgrades (#82, #85)
 * minimum required Go version upgrade to Go 1.19 (#83)
 * log messages adjustment (#88)
 * untie `dbft` module from `github.com/nspcc-dev/neo-go` dependency (#94)
 * minimum required Go version upgrade to Go 1.20 (#100)

## [0.1.0] (15 May 2023)

Stable dbft 2.0 implementation.

[Unreleased]: https://github.com/nspcc-dev/dbft/compare/v0.2.0...master
[0.2.0]: https://github.com/nspcc-dev/dbft/releases/v0.2.0
[0.1.0]: https://github.com/nspcc-dev/dbft/releases/v0.1.0