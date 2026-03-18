# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v1.3.0](https://github.com/strubio-ray/vm-ward/compare/0482ebe1840d405352749ae4d5f808afa694973e..v1.3.0) - 2026-03-18
#### Features
- add copier template update subcommand - ([0482ebe](https://github.com/strubio-ray/vm-ward/commit/0482ebe1840d405352749ae4d5f808afa694973e)) - Steven

- - -

## [v1.2.0](https://github.com/strubio-ray/vm-ward/compare/6ac2ca4887a2792976f60ce89ba7b324569b7a8d..v1.2.0) - 2026-03-13
#### Features
- add interactive TUI dashboard and destroy command - ([997ae73](https://github.com/strubio-ray/vm-ward/commit/997ae73fb14da02f0a5767aa91358a29eb1f51c3)) - Steven
#### Miscellaneous Chores
- (**vm**) update version - ([6ac2ca4](https://github.com/strubio-ray/vm-ward/commit/6ac2ca4887a2792976f60ce89ba7b324569b7a8d)) - Steven
- add .claude/ to gitignore - ([8b729f4](https://github.com/strubio-ray/vm-ward/commit/8b729f4c17c800d57fe3fdfddf794e94cada9ab9)) - Steven

- - -

## [v1.1.1](https://github.com/strubio-ray/vm-ward/compare/8df4a34ed932d38465cf5056c1238d114c8a850f..v1.1.1) - 2026-03-13
#### Bug Fixes
- (**install**) harden launchd install flow for macOS 26 - ([83580c8](https://github.com/strubio-ray/vm-ward/commit/83580c87601089ba2a543f9fa21ffa075405f5fa)) - Steven
- (**status**) guard empty active_lines array for set -u compatibility - ([8df4a34](https://github.com/strubio-ray/vm-ward/commit/8df4a34ed932d38465cf5056c1238d114c8a850f)) - Steven

- - -

## [v1.1.0](https://github.com/strubio-ray/vm-ward/compare/47c36506fef2cd080e7f57c8bf6b934ce0d6a8a6..v1.1.0) - 2026-03-13
#### Features
- (**events**) add structured event log - ([802e50f](https://github.com/strubio-ray/vm-ward/commit/802e50fbcd8ef733017cc5a2365f12b2d539b9fa)) - Steven
- (**lease**) add halted mode tracking - ([08703ab](https://github.com/strubio-ray/vm-ward/commit/08703abb921ac358ed06937b7d22b91ae6acf866)) - Steven
- (**status**) split active and recently halted sections - ([b62c84f](https://github.com/strubio-ray/vm-ward/commit/b62c84fd65213c74ebc2ebbeafcc2e7d89ff59e3)) - Steven
- (**status**) show original duration in time remaining - ([55a6907](https://github.com/strubio-ray/vm-ward/commit/55a690718c7f6b400aa3865752152c507cb47792)) - Steven
- (**status**) add pending lease state - ([a31b7cc](https://github.com/strubio-ray/vm-ward/commit/a31b7cc87a132ebfaa13aee86247dc6d8d2ea464)) - Steven
- (**status**) add last sweep timestamp - ([6c4345c](https://github.com/strubio-ray/vm-ward/commit/6c4345c4b5641520e754403a88cde9ba861a1eae)) - Steven
- (**status**) add daemon health detection - ([0649da6](https://github.com/strubio-ray/vm-ward/commit/0649da68c2bdb106ff6d20b24f384616a59efb14)) - Steven
- (**sweep**) add stale lease cleanup - ([23b0664](https://github.com/strubio-ray/vm-ward/commit/23b06642f2eb6328c457945327f74353bc0bc052)) - Steven
#### Bug Fixes
- (**status**) handle expired leases in JSON output - ([cd60aa0](https://github.com/strubio-ray/vm-ward/commit/cd60aa04a7f83a2023a124c1a991eac7b6c0ec31)) - Steven
- restore .vm/* glob pattern in gitignore - ([47c3650](https://github.com/strubio-ray/vm-ward/commit/47c36506fef2cd080e7f57c8bf6b934ce0d6a8a6)) - Steven
#### Documentation
- update CLAUDE.md for status observability features - ([5f270c2](https://github.com/strubio-ray/vm-ward/commit/5f270c2d540b316870f11d3e36ff6fd727cbf503)) - Steven
#### Miscellaneous Chores
- track vm sandbox configuration files - ([0a1b47b](https://github.com/strubio-ray/vm-ward/commit/0a1b47b5d6b54339549aa663b88b098637ee2845)) - Steven

- - -

## [v1.0.0](https://github.com/strubio-ray/vm-ward/compare/b24bf8fea4c786c1a178d45d16f31c6a9f05f570..v1.0.0) - 2026-03-12
#### Documentation
- update README and CLAUDE.md for host-only refactor - ([d4263d3](https://github.com/strubio-ray/vm-ward/commit/d4263d302e9fd6b2d697ef988e6fd0af657a8e0d)) - Steven
- note brew update requirement in CLAUDE.md - ([b24bf8f](https://github.com/strubio-ray/vm-ward/commit/b24bf8fea4c786c1a178d45d16f31c6a9f05f570)) - Steven
#### Refactoring
- replace SSH activity detection with VBoxManage metrics - ([a7d7d38](https://github.com/strubio-ray/vm-ward/commit/a7d7d38a02072bd413df621b0e682f1a282dfff9)) - Steven
#### Miscellaneous Chores
- add gitignore and commit format_ago helper - ([1100b99](https://github.com/strubio-ray/vm-ward/commit/1100b99c43f207a38a539c5c81bcee21d35e8b94)) - Steven

- - -

## [v0.1.3](https://github.com/strubio-ray/vm-ward/compare/4b8e8b297e0513d670fd967873dfa64f08c5e8ba..v0.1.3) - 2026-03-11
#### Bug Fixes
- correct VM status detection by resolving VBox UUIDs - ([4b8e8b2](https://github.com/strubio-ray/vm-ward/commit/4b8e8b297e0513d670fd967873dfa64f08c5e8ba)) - Steven
#### Documentation
- add project-level CLAUDE.md - ([f47b3ef](https://github.com/strubio-ray/vm-ward/commit/f47b3ef64fde6d6f5b453f2795ec0cf6207365ce)) - Steven

- - -

## [v0.1.2](https://github.com/strubio-ray/vm-ward/compare/v0.1.1..v0.1.2) - 2026-03-11
#### Miscellaneous Chores
- (**version**) v0.1.1 - ([f804f7a](https://github.com/strubio-ray/vm-ward/commit/f804f7abe51d3a7d1011e245555e4a42b5830a96)) - Steven

- - -

## [v0.1.1](https://github.com/strubio-ray/vm-ward/compare/d3df3554c3155d7f3e7d0e4c031854c91e7a6bc5..v0.1.1) - 2026-03-11
#### Features
- add homebrew formula support - ([d3df355](https://github.com/strubio-ray/vm-ward/commit/d3df3554c3155d7f3e7d0e4c031854c91e7a6bc5)) - Steven

- - -

## [v0.1.0](https://github.com/strubio-ray/vm-ward/compare/3c4575602ffd333781b285f3b541f371a25a2bfe..v0.1.0) - 2026-03-11
#### Features
- add launchd plist template - ([b647bd5](https://github.com/strubio-ray/vm-ward/commit/b647bd529c51fee6e4f71742ba923d1bc4c1a49b)) - Steven
- add host daemon with status, sweep, and lease management - ([4386cec](https://github.com/strubio-ray/vm-ward/commit/4386cec7fd038abe9c78d9575c4b9ecb6ebc29d4)) - Steven
- add vmw entry point - ([49937b0](https://github.com/strubio-ray/vm-ward/commit/49937b0120715ead723f5afdbe0a4ddc5e50bb2b)) - Steven
- add guest warning agent - ([57e842d](https://github.com/strubio-ray/vm-ward/commit/57e842d0f6420550bb4cac79b9fc8199f9fbee7e)) - Steven
- add shared utilities - ([dcb0b9d](https://github.com/strubio-ray/vm-ward/commit/dcb0b9d594681cd7dc126cf6efc012fd231785ab)) - Steven
- scaffold vm-ward repository - ([f698b9c](https://github.com/strubio-ray/vm-ward/commit/f698b9cba5a2c17fdde11e14b9983db03dbb3239)) - Steven
#### Documentation
- add README - ([9d3502a](https://github.com/strubio-ray/vm-ward/commit/9d3502a02f896fdd0a2850e1618e5121d6308c3c)) - Steven

- - -

Changelog generated by [cocogitto](https://github.com/cocogitto/cocogitto).