# Changelog

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
