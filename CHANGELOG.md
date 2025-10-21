# Changelog

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
