# Changelog

## [1.0.1](https://github.com/felipebz/javm/compare/v1.0.0...v1.0.1) (2026-11-25)


### Bug Fixes

* **init:** use CRLF line endings for cmd wrapper ([27ebaed](https://github.com/felipebz/javm/commit/27ebaeda6ab6653203dafb4dfc501fdd948c097e))
* **install:** retry Windows promotion on locked files ([9855ffe](https://github.com/felipebz/javm/commit/9855ffe9373c36afdbf121a2c3922b24114cac69))
* **install:** use package filename for archive extension, fixes GraalVM installation ([bb14a51](https://github.com/felipebz/javm/commit/bb14a51aa5133778c4ae7221da06d4b02b2bfbdc))
* send javm User-Agent on HTTP requests ([9af7858](https://github.com/felipebz/javm/commit/9af7858e2dcb5bb705eaa5a3799017b5f41acaa8))
* **version:** match plain selectors as numeric prefixes ([861f18a](https://github.com/felipebz/javm/commit/861f18a1cd97d9bfa928c00415c2d64c2e03c621))

## [1.0.0](https://github.com/felipebz/javm/compare/v0.17.0...v1.0.0) (2026-11-22)


### Features

* **cli:** look for .java-version in parent directories ([3653f51](https://github.com/felipebz/javm/commit/3653f51b7aa89e87b25890c207b903668b4ca825))


### Bug Fixes

* **alias:** link aliases to resolved JDK paths ([661af35](https://github.com/felipebz/javm/commit/661af3585126c1e927adca4fcf7c9350c8d9eab1))
* **alias:** reject version selector names ([9463efb](https://github.com/felipebz/javm/commit/9463efb953f2a90d981a6d1c5eb681765f6bde0b))
* **alias:** report unset aliases as not found ([2f5b006](https://github.com/felipebz/javm/commit/2f5b00610c651e33614c83f320c4602fd1d6e213))
* **cache:** warn on discovery cache invalidation failures ([d75c2b9](https://github.com/felipebz/javm/commit/d75c2b97635f0da37d557717298e322e02b7617b))
* **cache:** warn on discovery cache save failures ([0e458bf](https://github.com/felipebz/javm/commit/0e458bfdef8b9df4614a384ae40b7d78a5510906))
* **cli:** fail with exit 2 when no version and no usable .java-version ([485142c](https://github.com/felipebz/javm/commit/485142cc990d1559524d8a21f764f608cf66e869))
* **cli:** point command line errors to help ([8b2a50d](https://github.com/felipebz/javm/commit/8b2a50da7020a98e65f3db4704aeb869ae192b00))
* **cli:** preserve backslashes in quoted paths ([d7fa278](https://github.com/felipebz/javm/commit/d7fa278328e6d4171c668c512a0027667777cc76))
* **command:** report unknown subcommands as usage errors ([c8a53c0](https://github.com/felipebz/javm/commit/c8a53c0cac054bc448dc3e8ad1c1cab01871d6b6))
* **current:** detect JDK run by macOS launcher ([5a6de54](https://github.com/felipebz/javm/commit/5a6de54162292634318295bde0ab68904aa5318c))
* **current:** report discovered JDKs and fail when none is active ([66c5b4f](https://github.com/felipebz/javm/commit/66c5b4f3640c7869b1b77e80d3a63096b103eee5))
* **default:** migrate legacy default alias handling ([3ba72ae](https://github.com/felipebz/javm/commit/3ba72ae0550c486c6d1619a5f4f3ea25b3407c8b))
* **default:** print default when selector is omitted ([08cd6d6](https://github.com/felipebz/javm/commit/08cd6d672aa3ceff39ba0380eecdd38780716027))
* **default:** reject uninstalled defaults and warn when a default stops resolving ([85d026f](https://github.com/felipebz/javm/commit/85d026f46c27d0c310ce6a5a27ac3f44899c0296))
* **deps:** update module golang.org/x/term to v0.48.0 ([bb1abe8](https://github.com/felipebz/javm/commit/bb1abe8d5c07bf77de57a565f768f7e4511f7d00))
* **deps:** update module golang.org/x/term to v0.48.0 ([c99f918](https://github.com/felipebz/javm/commit/c99f91836036aeb07c475a56e719d3d94f39c277))
* **disco:** detect unknown distributions from HTTP errors ([cddc2ad](https://github.com/felipebz/javm/commit/cddc2ad1fd42de66bed0eb3e2a92b79a20c57c23))
* **disco:** reject distribution names with wrong case ([3b24cdc](https://github.com/felipebz/javm/commit/3b24cdcfccb1ccfd56090c96e43594c61b73c2eb))
* **discovery:** keep commands working with a broken managed JDK ([9bcaebd](https://github.com/felipebz/javm/commit/9bcaebd59d5f35b6dec08b068da24e3783a9ce05))
* **distribution:** reject unknown Disco distributions ([33f40e3](https://github.com/felipebz/javm/commit/33f40e33787d5a20c1ddc2ceb1b5cb44fbc2efaf))
* **error:** sanitize terminal text in errors ([d735c75](https://github.com/felipebz/javm/commit/d735c75c9c7afbca635d995deca1a4d004577a6d))
* **exec:** preserve child exit status and signals ([47fe829](https://github.com/felipebz/javm/commit/47fe829cf80a30c7295c5113ce977cbcc538c822))
* **exec:** suggest --jdk for selector-like commands ([3047c8c](https://github.com/felipebz/javm/commit/3047c8cc701c8ffa375f97c616e5957627e04c0c))
* harden selector and archive path parsing found by fuzzing ([f69f655](https://github.com/felipebz/javm/commit/f69f655922777efc012fed4bc7343ed89d0e941e))
* **help:** standardize command usage text ([78dab29](https://github.com/felipebz/javm/commit/78dab29311e3c6d5249d1dbb9e11d76cc5a6e805))
* **init:** reject incomplete cmd environment updates ([69e04da](https://github.com/felipebz/javm/commit/69e04da636197392d5516c17b03fcea031d210a3))
* **init:** reuse PowerShell init script files ([c3e22b7](https://github.com/felipebz/javm/commit/c3e22b702af7c921ec02d8d441917b7875c74747))
* **link:** avoid duplicate paths in symlink errors ([0714253](https://github.com/felipebz/javm/commit/0714253029cc8d597ef596f331bf5d8d94cc2b6d))
* **link:** log old target when removing alias links ([5e8c0f7](https://github.com/felipebz/javm/commit/5e8c0f77316f461562555a6b62ab80246825c8e3))
* **link:** log old target when removing alias links ([c15a382](https://github.com/felipebz/javm/commit/c15a382337721cf79dcbf7a5b19f22299ff0b8ef))
* **link:** register external JDKs by path ([acfe827](https://github.com/felipebz/javm/commit/acfe827923e745ffa762dec3a675426c7ecbe3e0))
* **link:** warn instead of failing when convenience links cannot be created ([8e8da4c](https://github.com/felipebz/javm/commit/8e8da4cb13b3a2186bbf7131b3e76ab4c0608135))
* **log:** prefix non-info messages by level ([6768033](https://github.com/felipebz/javm/commit/676803366c79f08d3e7889964cb2a66ddef8305f))
* **ls-remote:** align output columns with distribution ([ad1710d](https://github.com/felipebz/javm/commit/ad1710db4be4f8eacc9d1d14c16451c8bd6a44fb))
* **ls-remote:** support distribution selectors ([86b802e](https://github.com/felipebz/javm/commit/86b802edf8cda3346d376230cfb8e54e36e49e91))
* **ls:** align columns when rows have empty cells ([73f267d](https://github.com/felipebz/javm/commit/73f267d34dd9bae6d4d6fa8c5cbc1df7528d5c53))
* **ls:** group JDKs found through multiple sources ([4cff5c3](https://github.com/felipebz/javm/commit/4cff5c3105e2ac5e8d9212ce6fd2b1446a1be207))
* **selector:** accept trailing newlines and skip invalid pin files ([709ce61](https://github.com/felipebz/javm/commit/709ce6109be69796b19df7f2687619b366da0745))
* **selector:** hint modern syntax for legacy Java versions ([d7ba0a8](https://github.com/felipebz/javm/commit/d7ba0a84d2d59a8d2b360be931f8b72381f6d45f))
* **shell:** restore JAVA_HOME on deactivate in bash and zsh ([e4b18a0](https://github.com/felipebz/javm/commit/e4b18a0d0a9973eee1506513fb75d05a6c7eaf7f))
* **shell:** undo only the PATH and JAVA_HOME changes javm made ([fc973c5](https://github.com/felipebz/javm/commit/fc973c5fb77722404268bdd474deff4eb6b28c07))
* **state:** keep lock files in hidden locks directory ([1ae8884](https://github.com/felipebz/javm/commit/1ae88847cc8f647e41af57d300185f587df389c4))
* **uninstall:** refresh alias links after removal ([8862917](https://github.com/felipebz/javm/commit/8862917bca22032ca68414a0331e857a3a6dada9))
* **uninstall:** refuse ambiguous version selectors ([2ddbd05](https://github.com/felipebz/javm/commit/2ddbd05bb4b9fe262378a846b878f186ebae9321))
* **uninstall:** relink pins to discovered JDKs after removal ([c1c3629](https://github.com/felipebz/javm/commit/c1c3629b22f00e5a672e64a208b86dac77dfe55e))
* **uninstall:** report pin selection changes ([75349cf](https://github.com/felipebz/javm/commit/75349cfa5938f73f71dd71056cf8d0b215d9e70b))
* **uninstall:** use macOS bundle path in default link test ([e655f70](https://github.com/felipebz/javm/commit/e655f708c0c5ab2e96b4e1b836bd674f5a4c8d00))
* **uninstall:** warn when pins stop resolving ([2e237e2](https://github.com/felipebz/javm/commit/2e237e237fb5c551386719f8ef0f62bcfb7c6ffc))
* **version:** use Go build info for version output ([1e9cf2f](https://github.com/felipebz/javm/commit/1e9cf2f9cbb996325461f17b88d0321c82fb5b3a))


### Miscellaneous Chores

* bump version to 1.0.0 ([7a651da](https://github.com/felipebz/javm/commit/7a651dad481fa3300b20b7c9ecbc98fb09887fc9))

## [0.17.0](https://github.com/felipebz/javm/compare/v0.16.0...v0.17.0) (2026-11-18)


### Features

* **ls:** introduce colorized output and enhanced styling ([88aa49f](https://github.com/felipebz/javm/commit/88aa49fb3cf96902198e5d34c8f377ea4f973b46))
* **update:** add command to update installed JDKs ([018e92d](https://github.com/felipebz/javm/commit/018e92d5ab1620a4bdde799d0ca0e3f2fe13fea6))


### Bug Fixes

* **archive:** reject unsafe symlinks and hardlinks during extraction ([fa4b2bb](https://github.com/felipebz/javm/commit/fa4b2bb4517c8e8f20358a3e0436a2c635e2b019))
* **ci:** read JAVA_HOME using optional env access in nushell ([35e7552](https://github.com/felipebz/javm/commit/35e7552acf365073598f6992c17ef67d7b4a0fcb))
* **deps:** update module github.com/spf13/pflag to v1.1.0 ([b10cccc](https://github.com/felipebz/javm/commit/b10cccc161076a9d2389d3591c3c3bb308055839))
* **deps:** update module github.com/spf13/pflag to v1.1.0 ([6d36d7a](https://github.com/felipebz/javm/commit/6d36d7a5a50f6aea537ea78b8860651fd9526e81))
* **deps:** update module github.com/ulikunitz/xz to v0.5.17 ([#102](https://github.com/felipebz/javm/issues/102)) ([d38d7ea](https://github.com/felipebz/javm/commit/d38d7ea981ec62f341c0ae593886b020728b358f))
* **deps:** update module golang.org/x/sys to v0.49.0 ([80cfc31](https://github.com/felipebz/javm/commit/80cfc3166834d2fc8861348667e41204baf4fa3c))
* **deps:** update module golang.org/x/sys to v0.49.0 ([8ee3908](https://github.com/felipebz/javm/commit/8ee39087b5a8727531275ee8665e02ff62c25136))
* **deps:** update module golang.org/x/sys to v0.50.0 ([3764bdc](https://github.com/felipebz/javm/commit/3764bdc7c4a6823915435f9a20ee9af48b2c8530))
* **deps:** update module golang.org/x/sys to v0.50.0 ([15eeac9](https://github.com/felipebz/javm/commit/15eeac9734d5f820c665e23fcdbe0cef810a434e))
* **deps:** update module golang.org/x/term to v0.47.0 ([26dca5b](https://github.com/felipebz/javm/commit/26dca5b32c63c70f25d47d7efe735185fe26f6cf))
* **deps:** update module golang.org/x/term to v0.47.0 ([92520b4](https://github.com/felipebz/javm/commit/92520b4a09b6dc1f8b944703b2d53c5a3f55df9f))
* **install:** ensure symlink containment checks use physical paths ([f5b54c2](https://github.com/felipebz/javm/commit/f5b54c28027ad11540f5fe31e70c741414d2d788))
* **install:** prefer Zulu rebuilds and non-FX packages ([9d93877](https://github.com/felipebz/javm/commit/9d938773c60470dac7ad45311e2445013baab6fd))
* **link:** keep relinking after pruning a stale interim link ([8eb55e9](https://github.com/felipebz/javm/commit/8eb55e96d412ac24c852b917921a3901a553b4a4))
* **output:** sanitize terminal text to prevent unsafe escape sequences ([c8f9bd0](https://github.com/felipebz/javm/commit/c8f9bd03d9725adf00e853a9abff2da0436f9c21))
* **shell:** validate environment records and managed JDK paths ([c0c20c4](https://github.com/felipebz/javm/commit/c0c20c46a31d7a3bcdb668c1a4a3f4b6cb508517))

## [0.16.0](https://github.com/felipebz/javm/compare/v0.15.3...v0.16.0) (2026-09-10)


### Features

* **ls:** show selectors that resolve to each JDK ([6ed97f4](https://github.com/felipebz/javm/commit/6ed97f47d734f3669639a094cc28f3c4e01e868a))

## [0.15.3](https://github.com/felipebz/javm/compare/v0.15.2...v0.15.3) (2026-09-09)


### Bug Fixes

* **deps:** update module golang.org/x/sys to v0.48.0 ([ab52d32](https://github.com/felipebz/javm/commit/ab52d32b92154ee785f9ebe3c5d8849212644750))
* **deps:** update module golang.org/x/sys to v0.48.0 ([071b136](https://github.com/felipebz/javm/commit/071b136f383aa765294d9c87dd0c9ff3739d6cfb))
* **deps:** update module golang.org/x/term to v0.46.0 ([ce94a34](https://github.com/felipebz/javm/commit/ce94a34a76f750481817063ce7b96fdb32ba9213))
* **deps:** update module golang.org/x/term to v0.46.0 ([9802413](https://github.com/felipebz/javm/commit/9802413cbaeff0c0fb621927c0f3cb8ae612a695))
* **install:** use build-free directories for managed JDKs (25.0.4.1+1 -&gt; 25.0.4.1) ([a142656](https://github.com/felipebz/javm/commit/a1426563f23ceee24a2d439e4f24071183195543))
* **version:** ensure complete Java version preservation and build-independent comparison ([aefdf61](https://github.com/felipebz/javm/commit/aefdf61f9bbfa2d5a0ba5405016f8d1fb2697277))

## [0.15.2](https://github.com/felipebz/javm/compare/v0.15.1...v0.15.2) (2026-09-07)


### Bug Fixes

* **version:** support complete Java version ordering and selection ([65bccdf](https://github.com/felipebz/javm/commit/65bccdf1d9c32d0de74d97af8a50c2c8f7ebf1bc))

## [0.15.1](https://github.com/felipebz/javm/compare/v0.15.0...v0.15.1) (2026-08-27)


### Bug Fixes

* **deps:** update module github.com/sirupsen/logrus to v1.10.2 ([d3bb559](https://github.com/felipebz/javm/commit/d3bb5598eec16b9efab520cd0f488051b98c5049))
* **deps:** update module github.com/sirupsen/logrus to v1.10.2 ([e4a3812](https://github.com/felipebz/javm/commit/e4a381210aefe7d34c2548edb814b83daacabe93))
* **exec:** preserve child arguments after separator ([f4c469c](https://github.com/felipebz/javm/commit/f4c469cff7dd8a7590cf019cdc39627de91a898b))
* **exec:** replace separator with explicit JDK option ([272c795](https://github.com/felipebz/javm/commit/272c795aba7864927a7f86582f3bfe7f39c5d456))
* **pwsh:** preserve single arguments in shell integration ([cff69f9](https://github.com/felipebz/javm/commit/cff69f96137d714270eb02a8d7f7f49e1c5d5ac1))
* **shell:** ensure `--fd3` flag is correctly placed and scoped in environment commands ([e741cf7](https://github.com/felipebz/javm/commit/e741cf7af16c5b53f06cd844ee8416096391869b))

## [0.15.0](https://github.com/felipebz/javm/compare/v0.14.1...v0.15.0) (2026-08-25)


### Features

* **exec:** add `javm exec` command to execute with specific JDK context ([e6b9059](https://github.com/felipebz/javm/commit/e6b9059ea968a55c05913a42c57dca13ad5cacef))

## [0.14.1](https://github.com/felipebz/javm/compare/v0.14.0...v0.14.1) (2026-08-23)


### Bug Fixes

* **discovery:** include linked JDKs in local discovery ([35cbca2](https://github.com/felipebz/javm/commit/35cbca27e218469e4fbfad42633fc3ab0280bd72))

## [0.14.0](https://github.com/felipebz/javm/compare/v0.13.4...v0.14.0) (2026-08-23)


### Features

* **installer:** make attestation verification opt-in ([cf0864a](https://github.com/felipebz/javm/commit/cf0864a7a9b0b45a508dabe15af9753131a6bf01))

## [0.13.4](https://github.com/felipebz/javm/compare/v0.13.3...v0.13.4) (2026-08-21)


### Bug Fixes

* **deps:** update module github.com/sirupsen/logrus to v1.10.1 ([da504cf](https://github.com/felipebz/javm/commit/da504cf201200735157e8b8b1079ef2ffbd90357))
* **deps:** update module github.com/sirupsen/logrus to v1.10.1 ([f2d0b1f](https://github.com/felipebz/javm/commit/f2d0b1f6cb244f6160d46f2861d443e01850f7af))
* **discovery:** add JDK discovery warnings and improve error propagation ([3a51c68](https://github.com/felipebz/javm/commit/3a51c68c524bcef2fe3c381bd2bba589aca47e0f))

## [0.13.3](https://github.com/felipebz/javm/compare/v0.13.2...v0.13.3) (2026-08-17)


### Bug Fixes

* **cli:** apply ls filters and strengthen output tests ([b369f67](https://github.com/felipebz/javm/commit/b369f6735fd60b246d6c4bcd5fb7d899a1f8e1e3))
* **cli:** improve shell integration error handling and add detailed tests ([50b6970](https://github.com/felipebz/javm/commit/50b697086f8be5c4cb1759136622d3b76f7d71ae))
* **cli:** standardize argument validation and error handling across commands ([19b3b71](https://github.com/felipebz/javm/commit/19b3b7128208cd28831a7627b3fe5271db32943f))
* **deps:** update module github.com/sirupsen/logrus to v1.10.0 ([d927432](https://github.com/felipebz/javm/commit/d927432f224fd3e3a8ba4719759a57436ba5059a))
* **deps:** update module github.com/sirupsen/logrus to v1.10.0 ([13de4c1](https://github.com/felipebz/javm/commit/13de4c11647968640224695a127f63b10d8c048e))
* **deps:** update module golang.org/x/sys to v0.47.0 ([5dc292a](https://github.com/felipebz/javm/commit/5dc292a6a483470de943a2021cd102e8c97da5b6))
* **deps:** update module golang.org/x/sys to v0.47.0 ([93e70e2](https://github.com/felipebz/javm/commit/93e70e220354e8787da9d06c8bbb2ab153b35d00))
* **deps:** update module golang.org/x/term to v0.45.0 ([5b480f9](https://github.com/felipebz/javm/commit/5b480f9d460da34c45c41368c571be936db5c399))
* **deps:** update module golang.org/x/term to v0.45.0 ([cf17474](https://github.com/felipebz/javm/commit/cf174749124fc91dd727f8fc6109772da433cd71))
* **discovery:** propagate context through scans and subprocesses ([afbdf46](https://github.com/felipebz/javm/commit/afbdf4646d3358cdf63d3cd7f2002d1ef91379af))
* **install:** handle macOS-specific Java home path for legal file validation ([eb6d186](https://github.com/felipebz/javm/commit/eb6d1862dfd80407fbed71de16744b93d81c71a9))
* **install:** support Windows archive path separators ([7892e34](https://github.com/felipebz/javm/commit/7892e348f7693fe2a4366184b1f707edca986908))
* **state:** make configuration writes atomic and concurrency-safe ([668e2da](https://github.com/felipebz/javm/commit/668e2da247692e0664af3b9be9836d94369e33b7))

## [0.13.2](https://github.com/felipebz/javm/compare/v0.13.1...v0.13.2) (2026-08-08)


### Bug Fixes

* **discovery:** make cache lifecycle consistent and resilient ([274f4e5](https://github.com/felipebz/javm/commit/274f4e58f337842158d0cf9cca0df8c74965148b))
* **install:** handle progress termination and copy errors gracefully ([14aac95](https://github.com/felipebz/javm/commit/14aac956d7ba534caa85edfd3979bf80d185b871))
* **security:** prevent shell injection via default version ([c7ad4da](https://github.com/felipebz/javm/commit/c7ad4da4cb8325b0831245895c57c09e373255cf))

## [0.13.1](https://github.com/felipebz/javm/compare/v0.13.0...v0.13.1) (2026-08-08)


### Bug Fixes

* prevent path traversal in alias names ([dd26ea7](https://github.com/felipebz/javm/commit/dd26ea732626415e37328888b0fb79b1fa43e77c))
* prevent usage message when `install` command is canceled ([13b6d3b](https://github.com/felipebz/javm/commit/13b6d3bc274f13c6bab0f6df930bb20740bb7706))

## [0.13.0](https://github.com/felipebz/javm/compare/v0.12.2...v0.13.0) (2026-08-06)


### Features

* add CMD support for `javm init` (closes [#58](https://github.com/felipebz/javm/issues/58)) ([8e758bb](https://github.com/felipebz/javm/commit/8e758bbe5d1ec1b75007fbc77ff2777ec933f700))


### Bug Fixes

* **deps:** update module github.com/masterminds/semver/v3 to v3.5.0 ([4241a8e](https://github.com/felipebz/javm/commit/4241a8e6d443f0538a349ad1a61ac481d317aa08))
* **deps:** update module github.com/masterminds/semver/v3 to v3.5.0 ([63b7765](https://github.com/felipebz/javm/commit/63b7765f6b75d94b46633114fb52f5e2e4adfbba))
* **deps:** update module github.com/schollz/progressbar/v3 to v3.19.1 ([#75](https://github.com/felipebz/javm/issues/75)) ([e498d3d](https://github.com/felipebz/javm/commit/e498d3d9de4e017f35992973ce59ef3259fb4586))
* **deps:** update module github.com/ulikunitz/xz to v0.5.16 ([22f5d7e](https://github.com/felipebz/javm/commit/22f5d7ed8bd36a5a479821a738026099b7168342))
* **deps:** update module github.com/ulikunitz/xz to v0.5.16 ([09971aa](https://github.com/felipebz/javm/commit/09971aae3d9f9affc6ef5d2ee514f21ca7cae2b3))

## [0.12.2](https://github.com/felipebz/javm/compare/v0.12.1...v0.12.2) (2026-02-04)


### Bug Fixes

* **discovery:** handle non-existent cache file gracefully in DeleteCacheFile ([4fe721d](https://github.com/felipebz/javm/commit/4fe721deb3afe0e3faf8c62da7e4d47bb064bb12))
* **install:** clear discovery cache when no custom destination is set ([cbd6075](https://github.com/felipebz/javm/commit/cbd6075fe68475edfa1ff8aec3069ccc2d5062d9))

## [0.12.1](https://github.com/felipebz/javm/compare/v0.12.0...v0.12.1) (2026-01-20)


### Bug Fixes

* **deps:** update module github.com/sirupsen/logrus to v1.9.4 ([#52](https://github.com/felipebz/javm/issues/52)) ([e8db255](https://github.com/felipebz/javm/commit/e8db2554749fee8f4bf5db48f5759412879ec973))
* **discoapi:** add multi-architecture filtering for GetPackages ([7a4a54a](https://github.com/felipebz/javm/commit/7a4a54aaa44bf94216bd677b338a080a62d3ab1a))

## [0.12.0](https://github.com/felipebz/javm/compare/v0.11.0...v0.12.0) (2026-01-14)


### Features

* **nushell:** add support for Nushell (closes [#17](https://github.com/felipebz/javm/issues/17)) ([0d0067c](https://github.com/felipebz/javm/commit/0d0067c64eea1299c0f4fb49e7c1aac45c7f9391))

## [0.11.0](https://github.com/felipebz/javm/compare/v0.10.0...v0.11.0) (2026-01-12)


### Features

* **cache:** delete autodiscovery cache during install and uninstall operations ([ab5a097](https://github.com/felipebz/javm/commit/ab5a0974112dd41f1d4d581ce542f9c41712bebb))
* **discovery:** add JDK identifier generation and display ([ca10859](https://github.com/felipebz/javm/commit/ca10859f811c353ab8fc2122c9da18bb2a5dd7b5))


### Bug Fixes

* **alias:** correct file permission for alias file ([8bdd0e9](https://github.com/felipebz/javm/commit/8bdd0e97c831d1d1e55b354c0b6944930ecc085c))
* **discover:** fix OS-specific path handling ([b2092ba](https://github.com/felipebz/javm/commit/b2092baa7eab76e3ec828df6870ef3ed809e3944))
* **install:** ensure binary install and doc copy logic ([7c44c71](https://github.com/felipebz/javm/commit/7c44c71744c378b386585099103fd56f225673a4))
* **use:** fallback to full JDK list if no match found in managed-only mode ([1db216e](https://github.com/felipebz/javm/commit/1db216e0fd531022576a12c75a2263e92da27fa4))
* **which:** use JDK path directly ([14956e8](https://github.com/felipebz/javm/commit/14956e8f7508a8218fc669ddb9b8fe02f0418b2c))

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
