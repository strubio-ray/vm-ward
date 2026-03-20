# Changelog
All notable changes to this project will be documented in this file. See [conventional commits](https://www.conventionalcommits.org/) for commit guidelines.

- - -
## [v1.8.4](https://github.com/strubio-ray/vm-ward/compare/ebfd4e9c98345e25dd76fad22034cde57b181c89..v1.8.4) - 2026-03-20
#### Refactoring
- hardcode CPU activity threshold to 1% and remove configuration - ([ebfd4e9](https://github.com/strubio-ray/vm-ward/commit/ebfd4e9c98345e25dd76fad22034cde57b181c89)) - Steven

- - -

## [v1.8.3](https://github.com/strubio-ray/vm-ward/compare/489775697237a25c8d5210509b29512c8618f25a..v1.8.3) - 2026-03-20
#### Bug Fixes
- (**sweep**) collect CPU metrics for exempt and indefinite VMs - ([4897756](https://github.com/strubio-ray/vm-ward/commit/489775697237a25c8d5210509b29512c8618f25a)) - Steven

- - -

## [v1.8.2](https://github.com/strubio-ray/vm-ward/compare/aa717bed89d6ae2299db12cb2cd9d237c548e053..v1.8.2) - 2026-03-20
#### Bug Fixes
- (**sweep**) detect metrics collection state from query output - ([aa717be](https://github.com/strubio-ray/vm-ward/commit/aa717bed89d6ae2299db12cb2cd9d237c548e053)) - Steven

- - -

## [v1.8.1](https://github.com/strubio-ray/vm-ward/compare/ed4c41a4cd8f854441efcf096986605ee64c2701..v1.8.1) - 2026-03-20
#### Bug Fixes
- (**tui**) show dash instead of stale activity string when CPU data unavailable - ([ed4c41a](https://github.com/strubio-ray/vm-ward/commit/ed4c41a4cd8f854441efcf096986605ee64c2701)) - Steven

- - -

## [v1.8.0](https://github.com/strubio-ray/vm-ward/compare/57c442837a6fe7e64d5fdbda47543ab65a33cb53..v1.8.0) - 2026-03-20
#### Features
- (**tui**) display CPU % in activity column with threshold picker - ([57c4428](https://github.com/strubio-ray/vm-ward/commit/57c442837a6fe7e64d5fdbda47543ab65a33cb53)) - Steven

- - -

## [v1.7.0](https://github.com/strubio-ray/vm-ward/compare/7a61268049654180e80b8de4f2c918984574ee12..v1.7.0) - 2026-03-19
#### Features
- (**tui**) adopt bubbles/help for responsive keybinding display - ([fd6c671](https://github.com/strubio-ray/vm-ward/commit/fd6c671d99d6eed6203537def3166b60551f2b72)) - Steven
#### Miscellaneous Chores
- (**tui**) add charm.land/bubbles/v2 dependency - ([7a61268](https://github.com/strubio-ray/vm-ward/commit/7a61268049654180e80b8de4f2c918984574ee12)) - Steven

- - -

## [v1.6.1](https://github.com/strubio-ray/vm-ward/compare/f4e355e1b6a41e42d7244a125fd4efd8fef6fd40..v1.6.1) - 2026-03-19
#### Features
- (**tui**) auto-collapse sections based on terminal height - ([f4e355e](https://github.com/strubio-ray/vm-ward/commit/f4e355e1b6a41e42d7244a125fd4efd8fef6fd40)) - Steven

- - -

## [v1.6.0](https://github.com/strubio-ray/vm-ward/compare/dd821ae9202933d9f15a7b42edc9a9ceeb4f2290..v1.6.0) - 2026-03-19
#### Features
- (**tui**) responsive column hiding based on terminal width - ([dd821ae](https://github.com/strubio-ray/vm-ward/commit/dd821ae9202933d9f15a7b42edc9a9ceeb4f2290)) - Steven

- - -

## [v1.5.4](https://github.com/strubio-ray/vm-ward/compare/59d9fa04260bfcea61ac354a6300934dfef4ce69..v1.5.4) - 2026-03-19
#### Bug Fixes
- (**peek**) prevent freeze on first-time SSH connections - ([59d9fa0](https://github.com/strubio-ray/vm-ward/commit/59d9fa04260bfcea61ac354a6300934dfef4ce69)) - Steven

- - -

## [v1.5.3](https://github.com/strubio-ray/vm-ward/compare/e03cb396fdf2e98eba8ac63db289f96c36ff8b0f..v1.5.3) - 2026-03-19
#### Bug Fixes
- (**peek**) add timeout, cancellation, and progress indicator - ([e03cb39](https://github.com/strubio-ray/vm-ward/commit/e03cb396fdf2e98eba8ac63db289f96c36ff8b0f)) - Steven

- - -

## [v1.5.2](https://github.com/strubio-ray/vm-ward/compare/b9464dda87b1215d69534a1bb1c6ef18abfbde96..v1.5.2) - 2026-03-19
#### Bug Fixes
- (**tui**) use correct escape key name for bubbletea v2 - ([b9464dd](https://github.com/strubio-ray/vm-ward/commit/b9464dda87b1215d69534a1bb1c6ef18abfbde96)) - Steven

- - -

## [v1.5.1](https://github.com/strubio-ray/vm-ward/compare/6f5166549760c563beb16b1e51a873ee92a04797..v1.5.1) - 2026-03-19
#### Bug Fixes
- (**peek**) replace GNU timeout with portable perl alarm for macOS - ([6f51665](https://github.com/strubio-ray/vm-ward/commit/6f5166549760c563beb16b1e51a873ee92a04797)) - Steven

- - -

## [v1.5.0](https://github.com/strubio-ray/vm-ward/compare/d77836a6974ac4fd3aef7052b0978292cfaf4188..v1.5.0) - 2026-03-19
#### Features
- (**tui**) add in-progress indicator and duplicate action guard - ([48834ae](https://github.com/strubio-ray/vm-ward/commit/48834ae1fe669a2b54e91bbece88326818873119)) - Steven
- (**tui**) add provisioning prompt after template update confirmation - ([d77836a](https://github.com/strubio-ray/vm-ward/commit/d77836a6974ac4fd3aef7052b0978292cfaf4188)) - Steven
- add peek command to view VM terminal output and processes - ([efad7ac](https://github.com/strubio-ray/vm-ward/commit/efad7ac31eb127339ebf47d094d5af0b11c61469)) - Steven
#### Bug Fixes
- use correct charmbracelet/x/vt API for peek rendering - ([6ad0c00](https://github.com/strubio-ray/vm-ward/commit/6ad0c00f5105ca6167b4cad99e73a0ad591c5938)) - Steven
#### Miscellaneous Chores
- add charmbracelet/x/vt dependency - ([a9e92d9](https://github.com/strubio-ray/vm-ward/commit/a9e92d95f0ba40f5f4a6bac0d88496bf8dbfa984)) - Steven

- - -

## [v1.4.1](https://github.com/strubio-ray/vm-ward/compare/81d7b06b4997b66e59968392998dbc7c2a75c2d6..v1.4.1) - 2026-03-18
#### Bug Fixes
- (**tui**) show multi-line error toasts with longer display time - ([81d7b06](https://github.com/strubio-ray/vm-ward/commit/81d7b06b4997b66e59968392998dbc7c2a75c2d6)) - Steven
- pass .vm destination to copier update - ([14f59cb](https://github.com/strubio-ray/vm-ward/commit/14f59cbf4fad1775d5c45fe38dedcb0a2fa4896f)) - Steven

- - -

## [v1.4.0](https://github.com/strubio-ray/vm-ward/compare/763db7ce63238d9e3f74cf79dc5b98cf10b683e1..v1.4.0) - 2026-03-18
#### Features
- consolidate to interactive TUI dashboard - ([763db7c](https://github.com/strubio-ray/vm-ward/commit/763db7ce63238d9e3f74cf79dc5b98cf10b683e1)) - Steven

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