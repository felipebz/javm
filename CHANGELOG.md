# Changelog

## [0.10.0](https://github.com/felipebz/javm/compare/v0.9.0...v0.10.0) (2026-01-10)


### Features

* **config:** add configuration management with defaults and user overrides ([3791d92](https://github.com/felipebz/javm/commit/3791d9253f8ccfac6eccc975a20ce1ef9404ec3d))


### Bug Fixes

* **deps:** update module github.com/schollz/progressbar/v3 to v3.19.0 ([197925a](https://github.com/felipebz/javm/commit/197925a20e852f0217260819460f48ce27e2375b))
* **deps:** update module github.com/schollz/progressbar/v3 to v3.19.0 ([a302962](https://github.com/felipebz/javm/commit/a302962d112f80c52b3e6d73e1f4253e29701f5b))
* **deps:** update module github.com/spf13/cobra to v1.10.2 ([#46](https://github.com/felipebz/javm/issues/46)) ([2e23a85](https://github.com/felipebz/javm/commit/2e23a85e5d2dc42b7ded48390fa113d1933767ce))
* **install:** split version and URL logging into info and debug levels ([359ffff](https://github.com/felipebz/javm/commit/359ffff8e40a64264308f18b6695075ebb035d2d))
* **unzip:** optimize zip extraction and handle Zip Slip ([ebed190](https://github.com/felipebz/javm/commit/ebed190b88b9cacc620c01ade7d53191df93e0dd))


### Performance Improvements

* **untgz:** optimize extraction to single-pass and handle Zip Slip ([fa994c6](https://github.com/felipebz/javm/commit/fa994c62fb14c3c205b8fb1c8d5f3297bd0705c6))

## [0.9.0](https://github.com/felipebz/javm/compare/v0.8.0...v0.9.0) (2025-10-30)


### Features

* **install:** enforce HTTPS-only for download URLs ([e375a1e](https://github.com/felipebz/javm/commit/e375a1e70bc814aba6a3764b5a0bfa0a6004ac15))
* **install:** remove custom URL support ([afe2f33](https://github.com/felipebz/javm/commit/afe2f33e879985de4eb0e1fc7902fea03d57bcbb))

## [0.8.0](https://github.com/felipebz/javm/compare/v0.7.0...v0.8.0) (2025-10-29)


### Features

* **install:** add checksum validation for artifact integrity ([ec277e0](https://github.com/felipebz/javm/commit/ec277e0757bbf0d3588eebc6a6ff550c6942391e))

## [0.7.0](https://github.com/felipebz/javm/compare/v0.6.0...v0.7.0) (2025-10-28)


### Features

* **cli:** enhance version flag output with commit hash and build date ([4220479](https://github.com/felipebz/javm/commit/422047946b0364c9842ae55e8cd5927a8927363b))
* **discovery:** normalize architecture strings in JDK metadata ([8fba5cb](https://github.com/felipebz/javm/commit/8fba5cb57317304593978fa98673fa86c14bb82b))
* **discovery:** refactor JDK discovery to include root paths ([e93172d](https://github.com/felipebz/javm/commit/e93172d0df9f28eb6a164bb1d31a55c770eaa7f5))
* **discovery:** simplify JDK struct and metadata extraction logic ([1416a1e](https://github.com/felipebz/javm/commit/1416a1ed2c80f5aaf07e83c9ea6c92fe60767b4b))
* **install:** enhance install scripts with checksum verification and provenance validation ([f78180d](https://github.com/felipebz/javm/commit/f78180d64d63b9c088b5b05f92b6288555d618b4))
* **runtimeinfo:** harden musl/glibc detection ([cea0450](https://github.com/felipebz/javm/commit/cea04503dea31a113e71ee7488e329c46b2fb5f8))

## [0.6.0](https://github.com/felipebz/javm/compare/v0.5.0...v0.6.0) (2025-10-23)


### Features

* **tests:** add integration tests for javm installation and usage across Bash, Zsh, and Fish ([395fdb8](https://github.com/felipebz/javm/commit/395fdb851ad282615320a161d2994225612005aa))
* **workflows:** add macOS job to integration tests ([172b3bb](https://github.com/felipebz/javm/commit/172b3bb010e11d7a17b43f8eefd5096f42f4f091))
* **workflows:** trigger integration tests on completion of build workflow and add token for installation step ([afbec4e](https://github.com/felipebz/javm/commit/afbec4e206888a56c197bb9e57cd4eb346f871ee))


### Bug Fixes

* **command:** normalize OS values for ls-remote command ([f43693e](https://github.com/felipebz/javm/commit/f43693e055fbe99ab3b14e4850739afa1775e0f5))
* **install:** replace `jq` with `sed` to parse JSON response for tag retrieval ([5abf0f7](https://github.com/felipebz/javm/commit/5abf0f74f5b2442c74818f17336cfd118baddc08))
* **packages:** fix lib_c_type used to search macOS packages ([8e3a684](https://github.com/felipebz/javm/commit/8e3a6845d4c325d79ff2317235ba11f5dc211443))
* **packages:** replace runtime.GOOS with os argument ([ea61e7c](https://github.com/felipebz/javm/commit/ea61e7cb8114cb0b18fe40d93a6b57a72e017455))
* **workflows:** add ARM support for integration tests ([cf050ee](https://github.com/felipebz/javm/commit/cf050ee7a153e68d58ef1e96bec836bffc693421))
* **workflows:** enable debug flag for Java installation in integration tests ([7481d18](https://github.com/felipebz/javm/commit/7481d18ce8d3fb84dbb0a71ccebe360be0dc4fdc))
* **workflows:** move concurrency settings into job-level configuration ([bcf28dc](https://github.com/felipebz/javm/commit/bcf28dc7d57842dae09d675f7beadb7650d36b52))
* **workflows:** optimize apt-get and use JAVA_VERSION on install ([1d34444](https://github.com/felipebz/javm/commit/1d3444445f2844c440e9d5b688a6d8b27d7d3490))
* **workflows:** remove redundant PATH export from integration test scripts ([fbd3185](https://github.com/felipebz/javm/commit/fbd318535dbba92f69d1f87534e68d051e149696))
* **workflows:** remove x option from shell to limit verbose output in integration tests ([059a187](https://github.com/felipebz/javm/commit/059a187396659659f5a30d383de3ed6ff220618a))
* **workflows:** update Fish shell integration test to use correct initialization syntax ([052b533](https://github.com/felipebz/javm/commit/052b533de9844b3981aa5d9239ffef0570d94127))
* **workflows:** update integration test scripts and prevent `man-db` auto-update issues ([37a218c](https://github.com/felipebz/javm/commit/37a218c1b7194fea4d1f4dba87d16d61227ed32a))
* **workflows:** update javm which command to use --home flag in integration tests ([daf19cf](https://github.com/felipebz/javm/commit/daf19cfa2b5cd18cc3c12775d73bd7fde977814e))

## [0.5.0](https://github.com/felipebz/javm/compare/v0.4.0...v0.5.0) (2025-10-21)


### Features

* **commands:** scope --fd3 flag to use/deactivate commands and update shell wrappers ([c92ef21](https://github.com/felipebz/javm/commit/c92ef211b255902911b67229d44b1c2124c5f7cf))
* **packages:** support musl-based Linux distributions (closes [#9](https://github.com/felipebz/javm/issues/9)) ([1a9b1df](https://github.com/felipebz/javm/commit/1a9b1df47fc547d3307758e2745ccdb3a8a8cae7))
* **shellscripts:** propagate exit code from javm executable across all shell scripts ([ab2d8d0](https://github.com/felipebz/javm/commit/ab2d8d0f14ccf947856bd97a6f0bfaf277716b7f))

## [0.4.0](https://github.com/felipebz/javm/compare/v0.3.0...v0.4.0) (2025-10-19)


### Features

* **core:** harden env propagation via neutral SET/UNSET protocol ([65aee9b](https://github.com/felipebz/javm/commit/65aee9b150ffa2a3777205113fa502fa6716347e))
* **logging:** add --debug and --quiet flags ([40ca68e](https://github.com/felipebz/javm/commit/40ca68ed8ac460069798e21c27d4c01fa33a2b3f))

## [0.3.0](https://github.com/felipebz/javm/compare/v0.2.1...v0.3.0) (2025-10-15)


### Features

* **default:** add command `default` for setting a default Java version ([b09bf92](https://github.com/felipebz/javm/commit/b09bf92d7d6746612345533c2875208fa02c47c3))

## [0.2.1](https://github.com/felipebz/javm/compare/v0.2.0...v0.2.1) (2025-10-10)


### Bug Fixes

* **ci:** correct conditional for Syft installation step in release workflow ([95a022f](https://github.com/felipebz/javm/commit/95a022f805ba1edb8ef2348561c013751cba2965))

## [0.2.0](https://github.com/felipebz/javm/compare/v0.1.0...v0.2.0) (2025-10-10)


### Features

* integrate SBOM generation with GoReleaser ([1c48854](https://github.com/felipebz/javm/commit/1c488546fd1486f02a2147ae8f616c03ced14c7c))


### Bug Fixes

* **shellscripts:** correct argument order for `--fd3` in javm scripts ([a908e64](https://github.com/felipebz/javm/commit/a908e64cf22d2fd66851984975d36ba8d2ed8937))

## [0.1.0](https://github.com/felipebz/javm/compare/v0.0.1...v0.1.0) (2025-10-09)


### Features

* add "all" option for listing all distributions in ls-remote command ([4d7de4c](https://github.com/felipebz/javm/commit/4d7de4cbd85a519cc2b54842cb01fce50ffb70aa))
* add `ls-distributions` command to list available Java distributions (closes [#6](https://github.com/felipebz/javm/issues/6)) ([d9e04ca](https://github.com/felipebz/javm/commit/d9e04cac4b97894c5b95e1b8da6899d6510d36b7))
* add bash and zsh support for the init command (closes [#13](https://github.com/felipebz/javm/issues/13)) ([de2af0d](https://github.com/felipebz/javm/commit/de2af0d936a47095b26193d8d2955de5b220317b))
* add discoapi/client ([#5](https://github.com/felipebz/javm/issues/5)) ([443ed0f](https://github.com/felipebz/javm/commit/443ed0ff69323b291a1e595ab766acca3648fa46))
* add discoapi/distributions ([#5](https://github.com/felipebz/javm/issues/5)) ([67730f3](https://github.com/felipebz/javm/commit/67730f34ec1fc4bdb1911b7f8776078812c4888b))
* add discoapi/packages ([#5](https://github.com/felipebz/javm/issues/5)) ([3b79f16](https://github.com/felipebz/javm/commit/3b79f168a3c2626c27e1817f402f87253875b52b))
* add fish support for the init command (closes [#15](https://github.com/felipebz/javm/issues/15)) ([25b20ec](https://github.com/felipebz/javm/commit/25b20ece0ed7bcf4127e4d624f4e80427c27e865))
* add Gradle source for JDK discovery ([#16](https://github.com/felipebz/javm/issues/16)) ([d8616b2](https://github.com/felipebz/javm/commit/d8616b2be28ebd28200b7160b59c5336c7eda346))
* add initial GitHub Actions workflow for automated release creation and publishing ([94ea209](https://github.com/felipebz/javm/commit/94ea2099b71435854ae513e0257fe0fa787a626b))
* add installation script for Linux/macOS (closes [#18](https://github.com/felipebz/javm/issues/18)) ([fee3c26](https://github.com/felipebz/javm/commit/fee3c268675d86bdc65f5e64f44b8b2542425c07))
* add IntelliJ source for JDK discovery ([#16](https://github.com/felipebz/javm/issues/16)) ([f3bf0a3](https://github.com/felipebz/javm/commit/f3bf0a3ed38aaa511d12114042686092baf87337))
* add Jabba source for JDK discovery ([#16](https://github.com/felipebz/javm/issues/16)) ([e9db518](https://github.com/felipebz/javm/commit/e9db5183505b8b94670a27bceca157fb21c76ccb))
* add javm source for JDK discovery ([#16](https://github.com/felipebz/javm/issues/16)) ([4fb84da](https://github.com/felipebz/javm/commit/4fb84da60a293d303c49e62008c95d377744a948))
* add PowerShell installer script ([#19](https://github.com/felipebz/javm/issues/19)) ([b7b1654](https://github.com/felipebz/javm/commit/b7b16541caa19e4be3ca62c662429a5a9da193ec))
* adjust archive type and libc settings based on OS in package requests ([a5fd9a2](https://github.com/felipebz/javm/commit/a5fd9a2c4f1fb95e21ca335cf148ee6aaab33344))
* complete release workflow with SBOM generation and asset attestation ([72b41ce](https://github.com/felipebz/javm/commit/72b41ced00a34175b20abc99a70080ee7f54c5ee))
* configure release-please with manifest file ([19d3116](https://github.com/felipebz/javm/commit/19d31164ac3e37e0a36000384f08ba7fe1f09310))
* implement initial prototype of the init command ([#12](https://github.com/felipebz/javm/issues/12), [#14](https://github.com/felipebz/javm/issues/14)) ([d5a3581](https://github.com/felipebz/javm/commit/d5a358183b5e66fbc7281bf3959887cb153bc9fe))
* introduce JDK discovery mechanism ([#16](https://github.com/felipebz/javm/issues/16)) ([84b93a2](https://github.com/felipebz/javm/commit/84b93a2d6cb87e240bc15099c69c4671c80fcbdc))


### Bug Fixes

* **deps:** update module github.com/spf13/cobra to v1.10.1 ([89e39f4](https://github.com/felipebz/javm/commit/89e39f483959347cd0b9b028751371ca89ddaa66))
* **deps:** update module github.com/spf13/cobra to v1.10.1 ([3d87144](https://github.com/felipebz/javm/commit/3d87144ec85f198abc0823f5143e5d32a0e11fd3))
* **deps:** update module github.com/spf13/pflag to v1.0.10 ([#25](https://github.com/felipebz/javm/issues/25)) ([a4b72b5](https://github.com/felipebz/javm/commit/a4b72b5d0f1b30cfd9bf11c97cc90fbbb5a5274a))
* **deps:** update module github.com/ulikunitz/xz to v0.5.13 ([#24](https://github.com/felipebz/javm/issues/24)) ([4f6c7d4](https://github.com/felipebz/javm/commit/4f6c7d42c9cc958578fa0fbf89df2d0ca846cc3b))
* **deps:** update module github.com/ulikunitz/xz to v0.5.15 ([#26](https://github.com/felipebz/javm/issues/26)) ([3979f36](https://github.com/felipebz/javm/commit/3979f36c2e6a56597fda7b0cce6ee18052b8db5c))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([7109416](https://github.com/felipebz/javm/commit/71094165f9b32b4c58d6bea7afb4503e11f38e85))
* **deps:** update module gopkg.in/yaml.v2 to v3 ([31cc07c](https://github.com/felipebz/javm/commit/31cc07c431f3cd1ad7af6ac5e7e93fc3bb55f98b))
* Fix flaky tests on Windows ([52b7ade](https://github.com/felipebz/javm/commit/52b7ade1cbcc8e9a26fbb30d2aecbea9154ec5e8))
* update temp file prefix ([4403ad4](https://github.com/felipebz/javm/commit/4403ad4f7e1d6c9b7582ff0742c03c4c240623f2))
