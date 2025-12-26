## [1.3.0](https://github.com/mtandersson/foodlist/compare/v1.2.0...v1.3.0) (2025-12-26)

### Features

* **frontend:** implement secret path routing and IP whitelisting ([b858c1f](https://github.com/mtandersson/foodlist/commit/b858c1f7ffa8ac7572628a1de7d15ae7ae3c7d2f))

### Chores

* **frontend:** remove peer property from package-lock.json ([2ebebb0](https://github.com/mtandersson/foodlist/commit/2ebebb0615ef089565332c72d2a89f17004da3dc))
* **release:** update release workflow to create full package for di… ([#47](https://github.com/mtandersson/foodlist/issues/47)) ([e26020d](https://github.com/mtandersson/foodlist/commit/e26020d97cb83787a00ae25c94dd4dd996b3d513))
* **release:** update release workflow to create full package for distribution ([a91f0df](https://github.com/mtandersson/foodlist/commit/a91f0df180500dcdb967383710ac92d28838b601))

## [1.2.0](https://github.com/mtandersson/foodlist/compare/v1.1.1...v1.2.0) (2025-12-26)

### Features

* **backend:** log client IP and proxy headers on new websocket connection ([c071786](https://github.com/mtandersson/foodlist/commit/c0717863aec32bdccf101744e17d8a94fc8ee27b))

### Chores

* **backend:** implement configuration management using environment variables ([4ee95e0](https://github.com/mtandersson/foodlist/commit/4ee95e02f873c7d24fe59d2b124ad2c1d1d320e4))
* **backend:** update dependencies in go.mod and go.sum ([4cd3243](https://github.com/mtandersson/foodlist/commit/4cd3243b0fe13931d7ef1f641a24bbcafc0537be))
* **ci:** update daily chore release workflow and improve documentation ([02142fa](https://github.com/mtandersson/foodlist/commit/02142fa75fb69d2c9012e57ce6f16d66906fd1ac))
* **deps:** update actions/checkout action to v6 ([afc2d08](https://github.com/mtandersson/foodlist/commit/afc2d0804c13588a99b3f69ba7ebe493e05eabf5))
* **deps:** update actions/checkout action to v6 ([#31](https://github.com/mtandersson/foodlist/issues/31)) ([2cb28f7](https://github.com/mtandersson/foodlist/commit/2cb28f7a9e3119766a86ed56024c0e4d66bfbe91))
* **release:** update release workflow to include platform-specific … ([#44](https://github.com/mtandersson/foodlist/issues/44)) ([72f5926](https://github.com/mtandersson/foodlist/commit/72f5926db3163deae31bb98f9ff7177cbc8b79bf))
* **release:** update release workflow to include platform-specific binaries ([db3ed62](https://github.com/mtandersson/foodlist/commit/db3ed628078f4298963bb4b3a5dfdd716684eed9))

## [1.1.1](https://github.com/mtandersson/foodlist/compare/v1.1.0...v1.1.1) (2025-12-26)

### Bug Fixes

* **backend:** add mutex to State to prevent data races ([d365b58](https://github.com/mtandersson/foodlist/commit/d365b58330bc783bc64758485d51ab7baccf77fd))

### Tests

* **backend:** remove time.Sleep calls from tests ([06b7836](https://github.com/mtandersson/foodlist/commit/06b7836e08ae6b756e438516885e3732a5bc48ad))

### Chores

* **ci:** update release workflow to create full package and push Do… ([#43](https://github.com/mtandersson/foodlist/issues/43)) ([facf0a3](https://github.com/mtandersson/foodlist/commit/facf0a386045dbde7b5c50a99c7c5711071ba362))
* **ci:** update release workflow to create full package and push Docker image ([7e572dc](https://github.com/mtandersson/foodlist/commit/7e572dc36004c4cf645281fe54fb87451a152ace))
* **deps:** update actions/github-script action to v8 ([#32](https://github.com/mtandersson/foodlist/issues/32)) ([cb35900](https://github.com/mtandersson/foodlist/commit/cb3590019a5f67f17142518aa48deb24578848de))
* **deps:** update actions/setup-go action to v6 ([#33](https://github.com/mtandersson/foodlist/issues/33)) ([f414652](https://github.com/mtandersson/foodlist/commit/f4146527cdf818b50030ee256a92d744052da4ed))
* **deps:** update conventional-changelog-conventionalcommits to v9 ([#37](https://github.com/mtandersson/foodlist/issues/37)) ([a999a63](https://github.com/mtandersson/foodlist/commit/a999a63dd191f91370b1f99ebf3da09b8ce97e84))
* **deps:** update semantic-release monorepo (major) ([#38](https://github.com/mtandersson/foodlist/issues/38)) ([f6f13ce](https://github.com/mtandersson/foodlist/commit/f6f13ce2929753912329b0f76d09e7e2d020ea76))
* **renovate:** simplify package rules formatting ([2a7cef0](https://github.com/mtandersson/foodlist/commit/2a7cef0d04f76b2c61460ad19496ecd8a829b37f))
* **renovate:** simplify package rules formatting ([#42](https://github.com/mtandersson/foodlist/issues/42)) ([98c4a82](https://github.com/mtandersson/foodlist/commit/98c4a82573c808f591c8a10b7031de7f2148c4c5))

## [1.1.0](https://github.com/mtandersson/foodlist/compare/v1.0.2...v1.1.0) (2025-12-26)

### Features

* **ui:** replace span with button for section titles and todo items ([490659b](https://github.com/mtandersson/foodlist/commit/490659be1dc99536bc15043ff7398eb832ea1f2d))
* **ui:** replace span with button for section titles and todo items ([#41](https://github.com/mtandersson/foodlist/issues/41)) ([b37339e](https://github.com/mtandersson/foodlist/commit/b37339ee93b78204f04648357f5fbfa2f318cc4c))

## [1.0.2](https://github.com/mtandersson/foodlist/compare/v1.0.1...v1.0.2) (2025-12-26)

### Bug Fixes

* **ci:** add root package.json for semantic-release dependencies ([64da3eb](https://github.com/mtandersson/foodlist/commit/64da3eb9609e6759e98e2142d68a5525a98c3093))
* **ci:** add root package.json for semantic-release dependencies ([#35](https://github.com/mtandersson/foodlist/issues/35)) ([85c2e56](https://github.com/mtandersson/foodlist/commit/85c2e56fd90842a12eba5d2324d2879184aa11c3))

### Continuous Integration

* add all checks passed job to CI workflow ([18ea7e5](https://github.com/mtandersson/foodlist/commit/18ea7e52c1577bc9b5f24b61e4bf34045f104846))
* add all checks passed job to CI workflow ([#40](https://github.com/mtandersson/foodlist/issues/40)) ([896a007](https://github.com/mtandersson/foodlist/commit/896a007c39b3376aa71d88c23f127a72d754dc3d))

### Chores

* allow web dev ([0879be5](https://github.com/mtandersson/foodlist/commit/0879be5d97bd05f7326511d98c62c0e584ab0ee8))
* **ci:** change permissions for generate relasese ([cafd2f9](https://github.com/mtandersson/foodlist/commit/cafd2f90ed878c47cc399463cb0022fcae015e6c))
* **ci:** remove auto-merge workflow and release configuration ([973afbb](https://github.com/mtandersson/foodlist/commit/973afbb77becf0b4eb63f5302033cfa9f30ff8fd))
* **ci:** update GitHub Actions workflows to use v4 of actions ([8070e8a](https://github.com/mtandersson/foodlist/commit/8070e8ae536d5ebccfe97fff9339360468497c1c))
* **deps:** update jsdom to ^27.4.0 ([9b10a90](https://github.com/mtandersson/foodlist/commit/9b10a90f3d61baad7e419e9c43e7c3323559eecc))
* **deps:** update semantic-release monorepo ([#36](https://github.com/mtandersson/foodlist/issues/36)) ([7a34a9d](https://github.com/mtandersson/foodlist/commit/7a34a9d2f22e7f2d0290a84d2e1ef8af8b3d1f2e))
* **package:** update repository URL in package.json ([4df43e1](https://github.com/mtandersson/foodlist/commit/4df43e1eec79cb0e8501f3688f0309f132f2c635))
* **package:** update repository URL in package.json ([#39](https://github.com/mtandersson/foodlist/issues/39)) ([eabc325](https://github.com/mtandersson/foodlist/commit/eabc3254391644ea3db5c2b1ad88b09436b1da19))
* **release:** update release rules for docs, style, and test types ([3698161](https://github.com/mtandersson/foodlist/commit/3698161189e51d4e8be2739ac3c17d44d64ceb7f))

## [1.0.1](https://github.com/mtandersson/foodlist/compare/v1.0.0...v1.0.1) (2025-12-26)


### Chores

* **ci:** fix format problem ([74d4e7b](https://github.com/mtandersson/foodlist/commit/74d4e7b24622a5fc89e978544790c9d11baa67ce))
* **ci:** remove end-to-end testing setup and related documentation ([#25](https://github.com/mtandersson/foodlist/issues/25)) ([ba4d745](https://github.com/mtandersson/foodlist/commit/ba4d74519da2122e7f16454b0ff879e560ca45f1))
* remove end-to-end testing setup and related documentation ([d9606a1](https://github.com/mtandersson/foodlist/commit/d9606a17ba75089bdb8bc2b67579314ee17bf1c4))

## 1.0.0 (2025-12-26)


### Bug Fixes

* **frontend:** correct WebSocket send method to accept Commands instead of Events ([bb024bd](https://github.com/mtandersson/foodlist/commit/bb024bd50e957295af0f18b19566e538f883ab58))


### Code Refactoring

* **tests:** update WebSocket tests to handle client count messages ([4b377c9](https://github.com/mtandersson/foodlist/commit/4b377c9ddb3b5a95ab81c6e486f8951cf2a73b00))


### Continuous Integration

* add GitHub Actions workflows and semantic versioning with Cursor integration ([9180c01](https://github.com/mtandersson/foodlist/commit/9180c017d8991a2a449a5cac279c734827e80fbb))
* **renovate:** add automated dependency updates ([bd18a1b](https://github.com/mtandersson/foodlist/commit/bd18a1b3ce58d56132186b4ba85b7462c9e11feb))
* upgrade setup-go to v6 with check-latest flag ([f5c7326](https://github.com/mtandersson/foodlist/commit/f5c7326203f346cf097bf3b3e7d47c1ec910cfb2))


### Chores

* **deps:** migrate Renovate config ([9bf5ae3](https://github.com/mtandersson/foodlist/commit/9bf5ae3b044d1ec99b49646a9b748355ad2f0632))
* **deps:** update @testing-library/svelte to ^5.3.1 ([5234061](https://github.com/mtandersson/foodlist/commit/52340612ad12e2d96d5fbb0b19b93176e79511ae))
* **deps:** update @types/uuid to v11 ([b0ba912](https://github.com/mtandersson/foodlist/commit/b0ba91260c85fe97d3ba1f7618f06910f4387be2)), closes [#8203](https://github.com/mtandersson/foodlist/issues/8203)
* **deps:** update actions/checkout action to v6 ([5223395](https://github.com/mtandersson/foodlist/commit/522339573f44a378539f183d04eb661b370e4e3c))
* **deps:** update actions/checkout action to v6 ([#15](https://github.com/mtandersson/foodlist/issues/15)) ([de20610](https://github.com/mtandersson/foodlist/commit/de2061001feb8edd14738c0563b705193d811674))
* **deps:** update actions/setup-node action to v6 ([a2df20e](https://github.com/mtandersson/foodlist/commit/a2df20e614c345538c92ad4e0c520e406405a060))
* **deps:** update actions/setup-node action to v6 ([#16](https://github.com/mtandersson/foodlist/issues/16)) ([fd874a5](https://github.com/mtandersson/foodlist/commit/fd874a52b03908828e15c1ee22068c864c518866))
* **deps:** update actions/upload-artifact action to v6 ([e9a6b48](https://github.com/mtandersson/foodlist/commit/e9a6b48047e92372b6c40405f06935f5512958ba))
* **deps:** update actions/upload-artifact action to v6 ([#17](https://github.com/mtandersson/foodlist/issues/17)) ([9dd4cd8](https://github.com/mtandersson/foodlist/commit/9dd4cd85bff17490e2f068d7029f02b90bbba2bf))
* **deps:** update codecov/codecov-action action to v5 ([c87c3fe](https://github.com/mtandersson/foodlist/commit/c87c3fe498841cdfea08ec948ef698bf44d753cd))
* **deps:** update codecov/codecov-action action to v5 ([#18](https://github.com/mtandersson/foodlist/issues/18)) ([ea70a20](https://github.com/mtandersson/foodlist/commit/ea70a20900260284c0cd188d0ee2a95809becbd5))
* **deps:** update cypress to v15 ([0d58db5](https://github.com/mtandersson/foodlist/commit/0d58db5b695997617309c9cfc30426f32fcd0888))
* **deps:** update go to v1.25.5 ([f6ccbbf](https://github.com/mtandersson/foodlist/commit/f6ccbbf5d5ad26440efe573dd6d79323011de033))
* **deps:** update golangci/golangci-lint-action action to v9 ([7f57fe8](https://github.com/mtandersson/foodlist/commit/7f57fe8beb2d06d6a56fa0d02abc74e60fd9529d))
* **deps:** update golangci/golangci-lint-action action to v9 ([#20](https://github.com/mtandersson/foodlist/issues/20)) ([dcbdb5c](https://github.com/mtandersson/foodlist/commit/dcbdb5ca4158a304a32238bbdbbd3c595e73d421))
* **deps:** update node to v24 ([a580868](https://github.com/mtandersson/foodlist/commit/a580868172bb3bd6c2254fdbcfdd16c94df49937))
* **deps:** update renovatebot/github-action action to v40.3.6 ([03f549f](https://github.com/mtandersson/foodlist/commit/03f549fac716765ead37874bc6f422d4a0f9c3ef))
* **deps:** update renovatebot/github-action action to v44 ([5f6becb](https://github.com/mtandersson/foodlist/commit/5f6becb79819fab47181dcf1a8f3d98a11807d89))
* **deps:** update svelte to ^5.46.1 ([47afe5c](https://github.com/mtandersson/foodlist/commit/47afe5c9586aa94f1739af2dc8ebb5b58f2371a8))
* **deps:** update svelte-check to ^4.3.5 ([cab4006](https://github.com/mtandersson/foodlist/commit/cab4006acc8035f148bf2720923e58a695be530d))
* **deps:** update vite to ^7.3.0 ([4c7c63e](https://github.com/mtandersson/foodlist/commit/4c7c63e13179700137d92571b4667bce65b975d0))
* **docs:** reorganize documentation structure and update links ([ce4a162](https://github.com/mtandersson/foodlist/commit/ce4a162bdabc0f5a628ea937742bab8d99d5516a))
* **docs:** reorganize documentation structure and update links ([#5](https://github.com/mtandersson/foodlist/issues/5)) ([3fcbfcb](https://github.com/mtandersson/foodlist/commit/3fcbfcb91db38719b1461ce7b9a9cab2349f1a70))

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of FoodList application
- Real-time synchronization with WebSockets
- Event sourcing architecture
- Go backend with structured logging
- Svelte frontend with TypeScript
- Docker/Podman support
- Comprehensive test coverage
