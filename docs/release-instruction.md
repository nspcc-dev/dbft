# Release instructions

This document outlines the dbft release process. It can be used as a todo
list for a new release.

## Check the state

These should run successfully:
 * build
 * unit-tests
 * lint
 * simulation with default settings

## Update CHANGELOG and ROADMAP

Add an entry to the CHANGELOG.md following the style established there. Add a
codename, version and release date in the heading. Write a paragraph
describing the most significant changes done in this release. In case if the dBFT
configuration was changed, some API was marked as deprecated, any experimental
changes were made in the user-facing code and the users' feedback is needed or
if there's any other information that requires user's response, write
another separate paragraph for those who uses dbft package. Then, add sections
with release content describing each change in detail and with a reference to
GitHub issues and/or PRs. Minor issues that doesn't affect the package end-user may
be grouped under a single label.
 * "New features" section should include new abilities that were added to the
   dBFT/API, are directly visible or available to the user and are large
   enough to be treated as a feature. Do not include minor user-facing
   improvements and changes that don't affect the user-facing functionality
   even if they are new.
 * "Behaviour changes" section should include any incompatible changes in default
   settings or in API that are available to the user. Add a note about changes
   user needs to make if he uses the affected code.
 * "Improvements" section should include user-facing changes that are too
   insignificant to be treated as a feature and are not directly visible to the
   package end-user, such as performance optimizations, refactoring and internal
   API changes.
 * "Bugs fixed" section should include a set of bugs fixed since the previous
   release with optional bug cause or consequences description.

Create a PR with CHANGELOG changes, review/merge it.

## Create a GitHub release and a tag

Use "Draft a new release" button in the "Releases" section. Create a new
`vX.Y.Z` tag for it following the semantic versioning standard. Put change log
for this release into the description. Do not attach any binaries.
Set the "Set as the latest release" checkbox if this is the latest stable
release or "Set as a pre-release" if this is an unstable pre-release.
Press the "Publish release" button.

## Close GitHub milestone

Close corresponding X.Y.Z GitHub milestone.

## Announcements

Copy the GitHub release page link to:
 * Riot channel

## Dependant projects update

Create an issue or PR to fetch the updated package version in the dependant
repositories.
