# Changelog

## [1.4.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.2...v1.4.3) (2025-06-09)


### Bug Fixes

* update dependencies ([5acdc17](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/5acdc176a75771b715ec21a9d1762032b35024f6))
* update from upstream ([fffdd8e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/fffdd8e09c8ae62d264d2febcdd9a908a77bf14b))

## [1.4.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.1...v1.4.2) (2025-06-06)


### Bug Fixes

* adjust esi rbac permissions ([#115](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/115)) ([c37204a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c37204ac38882c92d9fb4c706e0dce7e7636c835))
* regenerate schema with latest plugin version ([#117](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/117)) ([146a307](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/146a3077a5bbfcd0f10510852df600c4553d4fa8))

## [1.4.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.0...v1.4.1) (2025-06-05)


### Bug Fixes

* add spiffe repo to make manifests ([#111](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/111)) ([5079fec](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/5079fec335edc3ebbee2a29c084888cd1c55fa17))

## [1.4.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.3.0...v1.4.0) (2025-06-05)


### Features

* add postgres generator ([#95](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/95)) ([3921fed](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3921fed106f8d1c01614ddbc72840936a456c13c))
* add spiffe support ([#103](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/103)) ([7486720](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7486720b2b5e81b29f6a00bf3aad16d2ec26a487))
* **aws:** Enable setting custom endpoints for AWS ECR for ECRAuthorizationToken generator ([#4821](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4821)) ([1947224](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1947224853a3cec0fc572dc59cba45d41cba3886))


### Bug Fixes

* adjust mongodb genenator spec ([#110](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/110)) ([283a756](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/283a756477849b2c6da2f9c476a2ceb8e0eeb662))
* helm push ([#101](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/101)) ([c265e42](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c265e42eca8a5daac859b575657aeaa7db606be9))
* sanitize user inputs; update username suffix generation ([#107](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/107)) ([9208a43](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9208a43937fcb4f28492f9b86ab23b0b9913c037))
* update dependencies ([#100](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/100)) ([07080fb](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/07080fb240ddca572a0f7f824ee93be584bfb989))
* update dependencies ([#97](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/97)) ([066944c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/066944cbf376de3f83993b25fac0335768771bba))

## [1.3.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.2.0...v1.3.0) (2025-05-29)


### Features

* spire-server as part of external-secrets ([8cc7912](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8cc791239792e1310c867d01b613b1b4211c8e9e))


### Bug Fixes

* chart locks ([c95fbe2](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c95fbe2f116a3b2200912ceec2118d7a7c1b3d12))
* conflicts ([fda7584](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/fda7584834cf5688d162092b8e813548300ac60b))
* e2e tests ([#4847](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4847)) ([a47a323](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a47a323089e2b1274e35039a1746eac3fc39555f))
* gcp regional push should have no replications ([#4815](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4815)) ([2740f07](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2740f077275d0ca4924d7c11679a2f754d6bceb0))
* generator state for  pushsecrets ([#4842](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4842)) ([abd9b5d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/abd9b5dabce6cdf3d83d091984b44b5218af0b6c))
* intermittent neo4j tests ([#84](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/84)) ([67d8840](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/67d88405ce2cedbaf3c6ab77ed3ca25b2b389ffb))
* set klog to logger for client-go ([#4818](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4818)) ([031fb75](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/031fb75c6ca79e409b7661f3a4f5ff9c68da744e))
* update aws iam to v2 ([0ecf364](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0ecf364b1266f0b3a0d42e6638dfe97cffa80b58))
* update dependencies ([#87](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/87)) ([82388e5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/82388e59e6bc598d2383b7e2bbaa5950d3db7a1a))
* update from upstream ([654499a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/654499a59d30cb8dacb458ff3072548304afd00c))
* update from upstream ([#86](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/86)) ([7508e68](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7508e6884dfcd388262ea3c4b0d9a80ca60a3d76))

## [1.2.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.7...v1.2.0) (2025-05-26)


### Features

* add mongodb generator ([85bffac](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/85bffac4e961ca0910c8446663fd3f881b3988c2))
* mongodb generator ([0e57df6](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0e57df6abb0c4d280bed8243b41fbc502704012b))

## [1.1.7](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.6...v1.1.7) (2025-05-23)


### Bug Fixes

* bump webhook to 0.3.2 ([#76](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/76)) ([81f0538](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/81f0538e9e4a037552c9e1af14b01106f4092c0f))

## [1.1.6](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.5...v1.1.6) (2025-05-23)


### Bug Fixes

* parsing sni only works if it matches the url - not always the case ([#72](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/72)) ([cc51c9d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/cc51c9d380d1a114b3141ee39148e51eef102ba8))

## [1.1.5](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.4...v1.1.5) (2025-05-23)


### Bug Fixes

* bump pod webhooks to 0.3.1 ([#70](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/70)) ([f7c09e5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f7c09e5ea0003563dc974911cbeec682bf008087))

## [1.1.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.3...v1.1.4) (2025-05-23)


### Bug Fixes

* bump webhook to 0.3.0 ([#67](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/67)) ([652bb64](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/652bb64028d2ba180ed1b415e6a0806c06e5ffd4))

## [1.1.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.2...v1.1.3) (2025-05-23)


### Bug Fixes

* conflicts ([d890382](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d890382ba14a8b173603520d8875b79e4d2a0d45))
* update from upstream ([#63](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/63)) ([7c73b91](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7c73b91ee2132cd2a8f716d73fe9b37027ba8bbc))

## [1.1.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.1...v1.1.2) (2025-05-22)


### Bug Fixes

* bump webhook chart to version 0.2.0 ([#61](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/61)) ([4a5800f](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4a5800f99e2a144c4250634af4a16fa84ffa520f))

## [1.1.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.1.0...v1.1.1) (2025-05-22)


### Bug Fixes

* repository values for webhook and certController ([#59](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/59)) ([4b016d6](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4b016d655de2814bbfe57027b096a1f77b92ac9f))

## [1.1.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.0.0...v1.1.0) (2025-05-22)


### Features

* federation service export ([#58](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/58)) ([e1905e3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e1905e33eb47bc81f54dc6fd6dc490e24e4f449c))
* implement revoke methods for federation server ([#48](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/48)) ([666f0c9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/666f0c9a04dc20ef45e329b76f7cb866edc4c19c))

## [1.0.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v0.16.3...v1.0.0) (2025-05-22)


### Features

* add 1Password SDK based provider ([#4628](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4628)) ([fa1d9bd](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/fa1d9bd3339a33adf0bebff196df56c8c616c0c9))
* add cleanup ([3195c18](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3195c18d65907ee7860c131671b1ef9034bc6db8))
* add create or replace user function ([2f43da4](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2f43da461d696ccb5e4f6cae3b77b5d690031339))
* add MFA token generator Generator ([#4790](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4790)) ([e5fac24](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e5fac2433f488008a498d6ba4f53a9c0c9f23558))
* add roles managements ([742c2cf](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/742c2cf5dc07f55f0c23e9eaafff5e1c1586f6a5))
* add ssh generator ([92c3008](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/92c300834f45f3d06e366387c5c04b3881f3e76c))
* add ssh generator ([854f81c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/854f81c612487a5c378d702901f92c5772ba3d2a))
* add username sufix; add enterprise check; add tests ([1f37808](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1f37808cc7765b599f4979d49a2922e3ccd8667f))
* basic auth generator ([#27](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/27)) ([bcd8b22](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/bcd8b2269e8161650b782b098eedc88e897eba75))
* esi-pod-webhook chart as a part of enterprise ([#43](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/43)) ([ba2ded9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ba2ded93e92a609c0084ff0842f133da61a427c3))
* generate crd; update register ([ca34af0](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ca34af09a1e79e6e471ab646ac6c1740da7207f6))
* generator for federation server ([6ea0255](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/6ea0255b18515577f9afa5f65a11c38cbbd25a87))
* improve tests ([286289d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/286289dde9482f9b55ca96a18b8544294db926aa))
* make sendgrid generator stateful ([#52](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/52)) ([a55e332](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a55e332cd598d90292887080108c69cbc5ad2010))
* neo4j generator ([d41d675](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d41d6751dd5f96cfe17f4fa287c13212b3e41c13))
* release 1.0.0 ([#51](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/51)) ([8c6d13b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8c6d13be8740ddec53bb7fd786cbb27fc4e47d5f))
* release from release please ([#56](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/56)) ([ab009f4](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ab009f4da70bef55c1af9f2281abff7184d4089c))
* Start neo4j generator ([0e85dcd](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0e85dcd4e1538ecacf1e2c58d8c3202c99f2c6a9))
* update RSA to be a ssh type ([b210f1a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b210f1abec8f769d9d9e4db62e1aef6c193764ed))


### Bug Fixes

* adds releases to stability and support. Adds a check on  workflow to force us to add docs to stability and support ([#4776](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4776)) ([9fab8ce](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9fab8ce7255627286edcd0d4b28786849f125c7e))
* allows result.jsonpath to be templated on datafrom calls ([#4808](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4808)) ([c27fb96](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c27fb9605259f2f3c866a6f304304ac20feba0a5))
* bump-chart ([#45](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/45)) ([909f37d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/909f37d163cbdb71c01f9d7ba143b671ac9280a4))
* conflicts ([230eb28](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/230eb287739079fd73eec6c61928227f58baf894))
* release check output is not a string ([#4782](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4782)) ([f90ce3b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f90ce3be860e8732b6ebf8b0270de98a83d83397))
* remove unecessary variables from Neo4jUser ([38208f3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/38208f3ce80480ca9c9a1a1676de24a3b696c097))
* should also bump tests and docs ([#55](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/55)) ([eebd9b6](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/eebd9b6b287f23f48ba737a3c220e32738d4b9cf))
* Support for Non-json secret fetched from Delinea SecretServer ([#4743](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4743)) ([8debc0e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8debc0e7e7e08099a5eecd215b4b1f571d5bedf8))
* typo ([d85231c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d85231c3b5a8bd68d68ea2d86ecb8401a8d04421))
* typo ([#34](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/34)) ([ab7fbe7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ab7fbe72697b0e58671b104959b878979641ca69))
* unused delimiter settings ([#4807](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4807)) ([af8c214](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/af8c21445191d0793eb72ce5dd5985e4bd8de521))
* update provider examples to use apiVersion external-secrets.io/v1 ([#4757](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4757)) ([6deca4a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/6deca4a6af37f17a4d7c76c091a130564b658db3))
* update-upstream ([#31](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/31)) ([0042f10](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0042f102462b40f93ff08db2f45d94eb1e4b17a0))
* upstream ([#32](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/32)) ([4009d6e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4009d6e20c8e98c098169a944734fc38d219df08))
* upstream ([#33](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/33)) ([b6f1149](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b6f1149f684ecfecbdc538122207e781c548831d))
