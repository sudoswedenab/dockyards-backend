# Changelog

## [0.18.5](https://github.com/sudoswedenab/dockyards-backend/compare/v0.18.4...v0.18.5) (2026-10-23)


### Features

* add ConfigManager.GetBool() ([4b86aee](https://github.com/sudoswedenab/dockyards-backend/commit/4b86aee5ca1d39a1fcc1544bf838ef6df49c1a9e))
* add jsonrpc2 package ([2d440ba](https://github.com/sudoswedenab/dockyards-backend/commit/2d440ba9644c21ec98772237ea5e5b7c0d792327))
* add jwt package ([5d1fcb3](https://github.com/sudoswedenab/dockyards-backend/commit/5d1fcb32dba62e7c1a8acf609a8a2ae785b08f03))
* add webshell base ([b372501](https://github.com/sudoswedenab/dockyards-backend/commit/b3725013e9d1db27d373dbb4044067a42b220623))
* allow disabling of reconcilers ([75c51de](https://github.com/sudoswedenab/dockyards-backend/commit/75c51de4bdd6ad56c65ed6e8bc66cb645329fe31))
* create/patch/delete/watch endpoints for v2 api ([57549b8](https://github.com/sudoswedenab/dockyards-backend/commit/57549b8b6f462638a08448b8f2da6d4cb49b8efc))
* use configmap allowedOrigins for websocket connection ([4b31907](https://github.com/sudoswedenab/dockyards-backend/commit/4b3190794b873f5ee557e71be4b3215af938fece))


### Bug Fixes

* add heartbeat to websocket connections ([910b662](https://github.com/sudoswedenab/dockyards-backend/commit/910b662a92401c9fd613bcddab0abab96b4a0fa1))
* add label validating webhook for cluster ([a90683a](https://github.com/sudoswedenab/dockyards-backend/commit/a90683a1c1f54a246af012c92e44055439569308))
* add owner reference to invitation in tests ([0b878c3](https://github.com/sudoswedenab/dockyards-backend/commit/0b878c30635d3e54c591b0f9df7be0d26bd5b753))
* do not fallthrough in StreamID() for ClientMessageKindInvalid ([f37b7a5](https://github.com/sudoswedenab/dockyards-backend/commit/f37b7a52927759794b014582363c70d5d76e7de4))
* ensure get verb is used in v2 api rbac when getting resources ([0108519](https://github.com/sudoswedenab/dockyards-backend/commit/01085194395e2f742fdc16c2840795dc9b5f574f))
* generation of webhooks ([3eb3242](https://github.com/sudoswedenab/dockyards-backend/commit/3eb3242584e8d04acc3a61e0b274b16d64bcb08b))
* return default slog if it is not in context ([45999c2](https://github.com/sudoswedenab/dockyards-backend/commit/45999c28fbb01ec3835b4877cb3a42d6b8156554))
* revert "chore: add validating webhook for workloads" ([9aa2de5](https://github.com/sudoswedenab/dockyards-backend/commit/9aa2de5542fd48b72416c9a9d70f2ade6749bca5))
* revert "chore: add validating webhooks for credentials" ([049aba4](https://github.com/sudoswedenab/dockyards-backend/commit/049aba4f3dd00387b269a2ecd768232c6e704865))
* revert "fix: generation of webhooks" ([d4400e0](https://github.com/sudoswedenab/dockyards-backend/commit/d4400e07e86a942a2fa132f1e7d4ff87f502203d))
* typo virtualmachineinstance =&gt; virtualmachineinstances ([57aec3b](https://github.com/sudoswedenab/dockyards-backend/commit/57aec3b2013480c1db31f918c3958b4c6c4ae05c))

## [0.18.4](https://github.com/sudoswedenab/dockyards-backend/compare/v0.18.3...v0.18.4) (2026-09-29)


### Features

* add dockyards.io/virtual-machine-name label constant ([e503d4b](https://github.com/sudoswedenab/dockyards-backend/commit/e503d4b2df1d2a00953433ad6eeb1f231884ceb3))
* add virtualmachine crd ([54b315d](https://github.com/sudoswedenab/dockyards-backend/commit/54b315dfb63e6f305b4d9953e14ae70c08508600))

## [0.18.3](https://github.com/sudoswedenab/dockyards-backend/compare/v0.18.2...v0.18.3) (2026-09-15)


### Features

* allow specifying NodeClassRef when creating Cluster/NodePool through API ([3b8ee61](https://github.com/sudoswedenab/dockyards-backend/commit/3b8ee61d0eda1de233fb3ad37b9bfdc23e15afe9))


### Bug Fixes

* ensure that nodeclass crd is included in the oci ([502e3df](https://github.com/sudoswedenab/dockyards-backend/commit/502e3dfcb869aa29519047b8476324514dbcb7bc))

## [0.18.2](https://github.com/sudoswedenab/dockyards-backend/compare/v0.18.1...v0.18.2) (2026-09-08)


### Features

* allow Dockyards Nodes to be run on specific K8s nodes via NodeClass ([adfded5](https://github.com/sudoswedenab/dockyards-backend/commit/adfded56540f8fdef5b18a00fa73e514b472d427))


### Bug Fixes

* update NodeClass to use NodeAffinity for simplicity ([8c01211](https://github.com/sudoswedenab/dockyards-backend/commit/8c01211d7abd4b3d5113317a213c78eaeae36c67))

## [0.18.1](https://github.com/sudoswedenab/dockyards-backend/compare/v0.18.0...v0.18.1) (2026-09-08)


### Features

* add release-please action ([0646cd6](https://github.com/sudoswedenab/dockyards-backend/commit/0646cd6754476a251fc13fff715402ccd88d63e9))

## 0.18.0

- Includes cumulative changes since `v0.13.10`.
