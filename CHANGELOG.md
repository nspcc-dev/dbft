# Changelog

This document outlines major changes between releases.

## [Unreleased]

New features:

Behaviour changes:

Improvements:
 * timer adjustment for most of the consensus time, more accurate block
   intervals (#55)
 * timer adjustment for network roundtrip time (#140)

Bugs fixed:
 * inappropriate log on attempt to construct Commit for anti-MEV enabled WatchOnly
   (#139)
 * empty PreCommit/Commit can be relayed (#142)

## [0.3.1] (29 November 2024)

This patch version mostly includes a set of library API extensions made to fit the
needs of developing MEV-resistant blockchain node. Also, this release bumps minimum
required Go version up to 1.22 and contains a set of bug fixes critical for the
library functioning.

Minor user-side code adjustments are required to adapt new ProcessBlock callback
signature, whereas the rest of APIs stay compatible with the old implementation.
This version also includes a simplification of PrivateKey interface which may be
adopted by removing extra wrappers around PrivateKey implementation on the user code
side.

Behaviour changes:
 * adjust behaviour of ProcessPreBlock callback (#129)
 * (*DBFT).Header() and (*DBFT).PreHeader() are moved to (*Context) receiver (#133)
 * support error handling for ProcessBlock callback if anti-MEV extension is enabled
   (#134)
 * remove Sign method from PrivateKey interface (#137)

Improvements:
 * minimum required Go version is 1.22 (#122, #126)
 * log Commit signature verification error (#134)
 * add Commit message verification callback (#134)

Bugs fixed:
 * context-bound PreBlock and PreHeader are not reset properly (#127)   
 * PreHeader is constructed instead of PreBlock to create PreCommit message (#128)
 * enable anti-MEV extension with respect to the current block index (#132)
 * (*Context).PreBlock() method returns PreHeader instead of PreBlock (#133)
 * WatchOnly node may send RecoveryMessage on RecoveryRequest (#135)
 * invalid PreCommit message is not removed from cache (#134)

## [0.3.0] (01 August 2024)

New features:
 * TLA+ model for MEV-resistant dBFT extension (#116)
 * support for additional phase of MEV-resistant dBFT (#118)

Behaviour changes:
 * simplify PublicKey interface (#114)
 * remove WithKeyPair callback from dBFT (#114)

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

[Unreleased]: https://github.com/nspcc-dev/dbft/compare/v0.3.1...master
[0.3.1]: https://github.com/nspcc-dev/dbft/releases/v0.3.1
[0.3.0]: https://github.com/nspcc-dev/dbft/releases/v0.3.0
[0.2.0]: https://github.com/nspcc-dev/dbft/releases/v0.2.0
[0.1.0]: https://github.com/nspcc-dev/dbft/releases/v0.1.0
