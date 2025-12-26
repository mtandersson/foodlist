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
