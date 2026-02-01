# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-02-01

### Added

- Vim-style `gx` keybinding to open current repo/build/step in browser

### Fixed

- Corrected curl flags in installation docs

## [0.2.0] - 2026-01-28

### Added

- Start in filter mode by default on repo list screen
- Vim navigation bindings (gg/G) for jumping to top/bottom in all views
- Breadcrumb statusbar showing current repo and build context
- Mouse support for clicking repos/builds and scrolling logs
- Toggle to show/hide inactive repos with 'a' key
- Double-escape to quit from repo list
- Select first build by default when viewing build list

### Changed

- Simplified log tab bar to single-line integrated with statusbar
- Removed spacing between build title and metadata for clearer grouping
- Removed extra newlines between log lines for cleaner display
- Hide inactive repos by default (toggle with 'a' to show all)

## [0.1.0] - 2026-01-28

### Added

- Interactive TUI for browsing Drone CI repos, builds, and logs
- Repo list with fuzzy search
- Build list with fuzzy search and status indicators
- Log viewer with tabbed step navigation and ANSI color support
- Keyboard-driven navigation (enter to drill down, esc to go back)
- Cross-platform binaries (Linux, macOS, Windows)
- MkDocs documentation site
