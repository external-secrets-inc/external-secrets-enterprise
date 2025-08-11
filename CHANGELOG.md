# Changelog

## [1.17.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.17.3...v1.17.4) (2025-08-11)


### Bug Fixes

* mark status as genstate subresource ([#418](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/418)) ([4e47cae](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4e47caeb913d8556e71d4c412c0c5d9121b6f993))

## [1.17.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.17.2...v1.17.3) (2025-08-11)


### Bug Fixes

* generator state status ([#416](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/416)) ([3b8a5bb](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3b8a5bbef073b0728744b91880510051b84afa43))

## [1.17.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.17.1...v1.17.2) (2025-08-11)


### Bug Fixes

* webhook permissions ([#414](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/414)) ([083992a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/083992aaf4efacd71877950ba91db459420a57c0))

## [1.17.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.17.0...v1.17.1) (2025-08-11)


### Bug Fixes

* bump golang ([256dec7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/256dec7cd04f3c531e20e1d4896d1926a7ad9ccf))
* conflicts ([378393b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/378393b46c0d0b3e1af457bbdd33fd529cae13cf))
* do not run ApplyTemplate for immutable secrets in `mutationFunc` ([#5110](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5110)) ([df939d8](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/df939d824ddbd0fd6b3ae310a928cc1ec3aa7de4))
* generator status permission: ([#413](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/413)) ([e919274](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e9192743aa840b17c3af2cf07b085e73bb07ad56))
* makefile and helm schema ([354cfb0](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/354cfb07c63393e22f084e1b5323f0b8380a267e))
* several fixes to secretstore crds ([#412](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/412)) ([63985b7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/63985b7e45c685a281d6e525dae499d2b55fe836))
* update from upstream ([#410](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/410)) ([41c3104](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/41c310416447708a30d830674d5dfaad5c3b744e))

## [1.17.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.16.3...v1.17.0) (2025-08-07)


### Features

* migration from endpoint to endpointslice ([#5008](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5008)) ([e212695](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e21269538d1e987820935be68673b2ac99d76a73))


### Bug Fixes

* check-diff ([e00a30e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e00a30e0e470c1e86e50e84d43567a70b8e9a1cb))
* conflicts ([814c3a6](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/814c3a676b1b74bb21cf580de6f5e4be491d7049))
* fail helm install if ClusterPushSecrets processing is enabled but PushSecrets processing is disabled. ([#4896](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4896)) ([3c847e3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3c847e36c35b770063fa73fb3ed004097453cb41))
* make secretstore change/addition/update to trigger scan jobs ([#406](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/406)) ([d169979](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d169979b6c1396fa1dcc1f4465aad357149d5099))
* move folders to enterprise ([#348](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/348)) ([e00a30e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e00a30e0e470c1e86e50e84d43567a70b8e9a1cb))
* update dependencies ([#354](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/354)) ([5d3b9c0](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/5d3b9c0b7eec8d6f1d83839d811f83aad116798d))
* update from upstream ([#355](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/355)) ([577fa3b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/577fa3b751bf86c5c762a6a9ccfedf3d933d65af))
* use server-side apply for CRD installation in Makefile ([#5103](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5103)) ([2bdee92](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2bdee9240dd1c3b26d3e6aef72e2cc97588023b5))

## [1.16.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.16.2...v1.16.3) (2025-08-04)


### Bug Fixes

* remove ubi ([#361](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/361)) ([82796b5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/82796b5935b87f1c656a3d4099cd0c73f3bbdce0))

## [1.16.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.16.1...v1.16.2) (2025-08-04)


### Bug Fixes

* build to public ([#358](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/358)) ([9421a16](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9421a1606bd829332fac9b6bc751f37cfb1fa299))
* build to public ([#360](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/360)) ([c2a24cf](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c2a24cf13c8178e05fd01e0e66d4b4ff6381b4ab))

## [1.16.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.16.0...v1.16.1) (2025-08-04)


### Bug Fixes

* promote images on public repository ([#356](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/356)) ([8ffaebf](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8ffaebf8fa907976eaef7f045063d6bb665e01e6))

## [1.16.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.15.4...v1.16.0) (2025-08-01)


### Features

* add generator state status and generator output ([#323](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/323)) ([b18cbf8](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b18cbf8fde747b257647d033a70b278944a5f434))
* add object[type] for workflow template ([#326](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/326)) ([6ad78d5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/6ad78d53e1af630f387e75056ebe40c0c79e30ec))
* add optional cleanup on generator step ([#310](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/310)) ([2d7acdd](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2d7acdda8ce5a659defc76a366061ef334889d7e))
* add workflow template finding type ([#307](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/307)) ([3ef7640](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3ef7640096d4607759bb031df530ecb0550f2c47))
* auto bump esi bundle ([0031865](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0031865c7c95b5a8a619985ca808a5c92d0752eb))
* auto bump esi bundle ([#352](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/352)) ([d19d035](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d19d035214ce1973a5ab9428d6f19c4a552bb5b0))
* **aws:** secretsmanager to update/patch/delete tags ([#4984](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4984)) ([014fb64](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/014fb645cc12f8f266a24539391e3e5a91f67e19))
* generator state cleanup policy ([#340](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/340)) ([5859ef3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/5859ef35aa8973d481878befb5a4befc50bdcd58))
* **infisical:** auth methods ([#5040](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5040)) ([c2bac01](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c2bac0199ad21001b5ee7a5cd34818b69e230114))
* openai idle cleanup policy implementation ([#341](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/341)) ([ac15bbc](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ac15bbc51d8b6bd94e5bc0951293a299e3dadebb))
* parse findings type to array[secretlocation] ([#320](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/320)) ([c434f74](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c434f74b982fb79835a563e45aca8afdfb041666))
* push to public registry ([#308](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/308)) ([253ee51](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/253ee51dd1ca643ec5e2c28818a289756c17289b))


### Bug Fixes

* add kubebuilder validation to VirtualMachineSpec auth ([#351](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/351)) ([406f248](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/406f248a339e0e726ef0937ebebc36b29441366f))
* add validation constraints to ExternalSecretRewrite to enforce single property selection ([#5006](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5006)) ([6f12eb9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/6f12eb909d6188d1d5515bd8c666dbebb370ab72))
* bug in script ([#5043](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5043)) ([9e6b82b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9e6b82b16e814ddc14ae53cec8671d6b5577e2d6))
* bumps release-support ([1e2f295](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1e2f295ee4572d7ee80580c65381dc2895e49b7e))
* bumps release-support ([9e6b82b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9e6b82b16e814ddc14ae53cec8671d6b5577e2d6))
* conflicts ([fe76493](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/fe76493b728ae15524337e8ea2848f3397f5b674))
* conflicts ([5e969cf](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/5e969cf2e1fe11c867053220bcda2636eb1b5901))
* conflicts ([741068f](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/741068f220afa529eca4de68820454d9283df0b3))
* files ([7591a94](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7591a94a942bef78dc4258403f44888237839a11))
* go.mod version ([e5357ef](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e5357ef0c9c4d1af50d92087d8fb02687279356d))
* **helm:** grafana dashboard: add widget for sum of not ready secrets ([#5086](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5086)) ([d172fcf](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d172fcf6db61863bc5fc8b295d251b3e0045151a))
* **helm:** grafana dashboard: fix heatmaps to actually be heatmaps, not time series ([#5069](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5069)) ([a96c38a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a96c38a57b6bf4a4ea9159d772e329d658e3006f))
* iam tests ([f9269f2](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f9269f292dfe64cf2837b6b169964f300340a0bf))
* only one pr for update deps or update upstream ([#345](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/345)) ([9c57e57](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9c57e57028a12a04870a796c8393469c5851284d))
* remove authentication option with JWT token from STSSessionToken generator ([#5026](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5026)) ([308712f](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/308712f10f0c7fd7d8ba732f79f85c8473493d7d))
* restore AWS credential chain resolution for ECRAuthorizationToken generator ([#5082](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5082)) ([7acab69](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7acab694b69841b8aee4bbaaca1043aa22f6c0f3))
* update custom object format to 'object[argName]resourceType' ([#329](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/329)) ([8978cc8](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8978cc8651dbc4a55e6c7959bd0c8ce8678b6ede))
* update dependencies ([#343](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/343)) ([42c03c9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/42c03c9492cf84a566c93b4f381ac65051aff96c))
* update dependencies ([#350](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/350)) ([df7546a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/df7546ac1a29c32da5c2c9a443d4f2b7d0bf9a1c))
* update from upstream ([#324](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/324)) ([034babc](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/034babcc029401fb775f70e7562b844ff9353a9c))
* update from upstream ([#342](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/342)) ([1e2f295](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1e2f295ee4572d7ee80580c65381dc2895e49b7e))
* update from upstream ([#349](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/349)) ([4b75901](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4b759016006412af7cbda5ed3a2c87b8c9e4b652))
* update tests ([75c03b2](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/75c03b25d4496a2de70b2eda33093509208f1093))
* update tests ([#353](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/353)) ([0b6635e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0b6635e0c8563cfeb9b59a84c11ece317bff6e2c))

## [1.15.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.15.3...v1.15.4) (2025-07-16)


### Bug Fixes

* add targets to webhook rbac ([#299](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/299)) ([b38d15e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b38d15e35301e4e876face39e2e6859ecc06a48b))

## [1.15.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.15.2...v1.15.3) (2025-07-16)


### Bug Fixes

* allow vms to work ([#296](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/296)) ([884f6c1](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/884f6c1ff118a2167cd9809f3d8db93690f85cb4))
* just in case ([#298](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/298)) ([18d7e1c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/18d7e1c8da36526a32a68c0decc674295a01491a))

## [1.15.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.15.1...v1.15.2) (2025-07-16)


### Bug Fixes

* recreate findings if name changes ([#294](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/294)) ([b16b3cc](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b16b3ccf2045f796a0a2f8f95f10dc7ca70e55d4))

## [1.15.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.15.0...v1.15.1) (2025-07-15)


### Bug Fixes

* api version for secretstores ([#288](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/288)) ([be1d306](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/be1d3065f959cb0326391fbf38f3d0e1eb956bdb))
* register openai controller ([#286](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/286)) ([7678e36](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7678e3665ba060e04360cce97ddefdbddef1894e))

## [1.15.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.14.0...v1.15.0) (2025-07-15)


### Features

* add openAI generator ([#266](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/266)) ([14fb4a5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/14fb4a58d69cb145b5f6e39ddac4d6413444ed15))

## [1.14.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.5...v1.14.0) (2025-07-15)


### Features

* adds vm pushsecret ([#249](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/249)) ([8253af7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8253af72d93999246608240ac542fa4a2458cd7c))
* **wip:** adds vm target ([8253af7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/8253af72d93999246608240ac542fa4a2458cd7c))


### Bug Fixes

* add timeout to job; change scheduled to poll ([#284](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/284)) ([dd30aa8](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/dd30aa8ad572d1331c603fb7744f956269a7cf3b))
* cleanup findings if no longer valid ([#282](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/282)) ([3d8414c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3d8414c3161e77744b8eb91e741d9585ebe17954))

## [1.13.5](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.4...v1.13.5) (2025-07-14)


### Bug Fixes

* sanitize names ([#273](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/273)) ([3c369c1](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3c369c1425286d7c8f48671f195f336e55a2c6cd))

## [1.13.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.3...v1.13.4) (2025-07-14)


### Bug Fixes

* close manager so jobs do not get stuck for locking stores like gcp ([#269](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/269)) ([9fcbb16](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9fcbb165ca53e2b3801591424136e860b1452ab7))
* lint ([#271](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/271)) ([48284bb](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/48284bb6ecae5921e8b9fa7fc2442fd197226f4a))

## [1.13.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.2...v1.13.3) (2025-07-14)


### Bug Fixes

* sanitize name for k8s ([#267](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/267)) ([4cf3088](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4cf3088e7560d4109259672cf71541c82083132e))

## [1.13.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.1...v1.13.2) (2025-07-14)


### Bug Fixes

* findings names ([#264](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/264)) ([cd5b7ef](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/cd5b7ef471212ffa9f79666c765a025fa99b6bad))
* only scan for objects in the same namespace ([cd5b7ef](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/cd5b7ef471212ffa9f79666c765a025fa99b6bad))

## [1.13.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.13.0...v1.13.1) (2025-07-14)


### Bug Fixes

* skip stores with unsupported methods ([#260](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/260)) ([4f32448](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4f324481ee2aee4669025678ec18232c7baf97e9))

## [1.13.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.12.0...v1.13.0) (2025-07-13)


### Features

* adds target CRDs and VirtualMachine Target logic ([#247](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/247)) ([50dc548](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/50dc54895fbb433074cc842cae21d97fa1270c96))


### Bug Fixes

* conflicts ([90f2290](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/90f2290471ffdfa5a08c4704e6981e92c61cf023))
* types ([#258](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/258)) ([624b159](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/624b159e9b48630c3f1ceb696f99bdcaf597fcee))
* update dependencies ([#256](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/256)) ([1f152f3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1f152f38eb4466bc28e4e00be0ff2296d540bdab))
* update from upstream ([#255](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/255)) ([2daf7ba](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2daf7ba3f3b7daa280735f48b0f86241776ecc8b))

## [1.12.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.11.0...v1.12.0) (2025-07-11)


### Features

* add secretlocation and secretlocation array ([#250](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/250)) ([e422b11](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e422b11e5e9725ca02c5633e1b3d2dbc05cfc9b1))


### Bug Fixes

* generator executor ([#251](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/251)) ([662394a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/662394ad5dbd1b4209a1905ef34bf5382fa37fe8))

## [1.11.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.10.0...v1.11.0) (2025-07-10)


### Features

* support objects as workflowtemplates parameters ([#240](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/240)) ([351a984](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/351a984b71e28aa0c68d2cc9cffc4364357c45df))
* workflow template support generators ([#230](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/230)) ([a8718cb](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a8718cbdf64f822d380e4b22ae445c3729b10a3e))


### Bug Fixes

* do not turn original value into string on value scope ([#5011](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/5011)) ([1b3fc5d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1b3fc5d9162e1e8298bfd1b4b2e2155211390604))
* steps mapping output ([#243](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/243)) ([e3e8cbe](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e3e8cbe58496d247d79fecf7b4af1c4a1de651d3))
* update dependencies ([#238](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/238)) ([4a7624d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4a7624d52e7253c42d1376fdc730644cd10130e6))
* update dependencies ([#242](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/242)) ([59a87b9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/59a87b90c23c66029305d53ec0d1a59459cf0d0f))
* update from upstream ([#237](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/237)) ([9fa70ad](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9fa70ad235f57b064b8cf41e3d1fcaad1dd445a9))
* update from upstream ([#241](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/241)) ([9bc024d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9bc024dda1695a96f4761cdc8af8f3f8ce31a9e0))
* updates not working when creating new findings ([#239](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/239)) ([dd0565a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/dd0565a92fb7e03924f21b42a849c4b3225f3379))

## [1.10.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.5...v1.10.0) (2025-07-08)


### Features

* add scan jobs and scan findings ([afff536](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/afff536af569546d7452e7c474dc795095855a0d))
* add scan jobs and scan findings ([#232](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/232)) ([e276eed](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e276eedccb184e4d7599bc5fe4868f1966c04723))


### Bug Fixes

* add rbac ([f3284ef](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f3284efbba7c0e4c38e5f56ac36ffc309d536546))
* boilerplate for findings ([ef1fb0a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ef1fb0a78f0c6b9ad2b521c088d37d8f24c76088))
* conflicts ([7352d07](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/7352d079c6305e2bf8ebd22c5967c28c580c937e))
* license ([2eea927](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2eea927a622e685c75902bf6b5406afff4a6a717))
* lint ([f8a0ffd](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f8a0ffd389932486abcd8463e62cb9c298be679f))
* lint and package move ([d80d8ef](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d80d8efbaa541815bde6f4cf767021c6a3bddeee))
* only update findings if needed ([31ef272](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/31ef2720e2bce95d9d9ff6d42c06c35374e8f590))
* support runtime for workflow run and workflow ([#222](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/222)) ([daeea59](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/daeea59df59e12c02c4524c3bf1f17e5759009ad))
* typo on aws key list resource ([#229](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/229)) ([e3f9534](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e3f95341492a9d39398b9cccdd268c1d6be55a6f))
* update dependencies ([#233](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/233)) ([b6c75ed](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b6c75ed067483ecceafd33dc479e278138102623))
* update from upstream ([#231](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/231)) ([ed6c52a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ed6c52a34b738c4ed9295c2e1ea200303934a37e))

## [1.9.5](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.4...v1.9.5) (2025-07-04)


### Bug Fixes

* release new version ([#220](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/220)) ([affd0d5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/affd0d5538f95b78d175b24f377a74f9039b8df9))

## [1.9.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.3...v1.9.4) (2025-07-02)


### Bug Fixes

* use es templates on transform step ([#209](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/209)) ([482b186](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/482b186cc6af6ca0ba41c3254f8f1ba4de9fcde6))

## [1.9.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.2...v1.9.3) (2025-07-02)


### Bug Fixes

* add isjson template function ([#207](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/207)) ([c4d52e9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/c4d52e94d783d70fbe9fb7f982e343ad7c84aee0))

## [1.9.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.1...v1.9.2) (2025-07-02)


### Bug Fixes

* remove masking ([#205](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/205)) ([332c52a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/332c52a96f0b5a7ca8310a1b5d1409be55344db4))

## [1.9.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.9.0...v1.9.1) (2025-07-01)


### Bug Fixes

* resolve conflicts ([ea49a66](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ea49a6648c1dd61483fb8a0c879a7393581edd68))
* update dependencies ([#201](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/201)) ([b67be63](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b67be63a8addff92abbbf747a13b826ad26c08e4))
* update from upstream ([#200](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/200)) ([a8145a1](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a8145a14f18293e162366f066c3f4000d567f158))

## [1.9.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.8.1...v1.9.0) (2025-06-30)


### Features

* add phase, start time, and completion time tracking for workflow runs ([#198](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/198)) ([2ad7a6f](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/2ad7a6f85f9843e996545e102e98eaafc5c9cf83))

## [1.8.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.8.0...v1.8.1) (2025-06-29)


### Bug Fixes

* conflicts ([d34f9a1](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d34f9a1b530b56ca3927cb54d9efb331fc95f577))
* update dependencies ([#193](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/193)) ([4d0a808](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/4d0a808fca8d08dd1914053a2219952d580f0e85))
* update from upstream ([#192](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/192)) ([a64cec7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a64cec7e2438f31f2a143442ba3398957b9e9790))
* webhook permissions ([#189](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/189)) ([9911d94](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/9911d94f0824fe1359f609da8f74db901b2af126))
* workflowrun permissions ([#187](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/187)) ([1f5cce3](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1f5cce3d025a79527187984f1a6cb9dcdb7f2830))

## [1.8.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.7.4...v1.8.0) (2025-06-27)


### Features

* secretstore array type for workflowruntemplate ([#183](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/183)) ([16edf1a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/16edf1a34693eb3b5adb38ea716a2bc71e878e54))


### Bug Fixes

* update dependencies ([#181](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/181)) ([628a04a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/628a04af16db76878895b76b1c21fef8d56c57af))
* update from upstream ([#180](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/180)) ([a6caa4c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a6caa4cca00b751f89d6a9ad400e3a061da1b216))

## [1.7.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.7.3...v1.7.4) (2025-06-25)


### Bug Fixes

* update dependencies ([#178](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/178)) ([76cf36d](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/76cf36dda5a43e00eaf5a07efa542b25cfeecee9))
* update from upstream ([#177](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/177)) ([b4b0fd7](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b4b0fd7808738415cb6187ffd0a2ccd65b315e91))
* workflow  run templates issues ([#176](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/176)) ([59d2723](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/59d2723e6eb3dd3009a4bae3c5741cf478b77acf))

## [1.7.3](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.7.2...v1.7.3) (2025-06-23)


### Bug Fixes

* webhook for workflowruntemplate ([#174](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/174)) ([f6cda8a](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f6cda8af072ff142f522a5eefbdac3feae73b36b))

## [1.7.2](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.7.1...v1.7.2) (2025-06-23)


### Bug Fixes

* custom path ([#172](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/172)) ([e1b1568](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/e1b156890a573402545e0271344991a15fd20325))

## [1.7.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.7.0...v1.7.1) (2025-06-23)


### Bug Fixes

* deployment issues with workflowruntemplates and webhooks ([#170](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/170)) ([0efb393](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/0efb3938b4b48a675301f486363ca8c5e598032a))

## [1.7.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.6.1...v1.7.0) (2025-06-23)


### Features

* workflow run templates ([#161](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/161)) ([1804f5c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1804f5cfec8dd1f560d7aa88ea7f46f2f3fa9390))
* workflowtemplate parameters should allow types ([#169](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/169)) ([1ad7c09](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1ad7c091f0cb7709ec0ddfb56cfa74bb1e32d6b0))


### Bug Fixes

* lint ([1804f5c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1804f5cfec8dd1f560d7aa88ea7f46f2f3fa9390))
* update dependencies ([#162](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/162)) ([30ad9f4](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/30ad9f4457c7c50f5f0a83430227d04eb35d98f2))
* update dependencies ([#166](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/166)) ([21a67a4](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/21a67a4d76b5e9fc1419a266c86cb04f0a8c9bbf))
* update dependencies ([#168](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/168)) ([68e1903](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/68e1903a34efe47c5399cb5e8cc6bfd7f3193c6b))
* update from upstream ([#167](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/167)) ([a4bc571](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a4bc571dee8d4a04db503b8b3f6e11c6c7f70ce1))

## [1.6.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.6.0...v1.6.1) (2025-06-18)


### Bug Fixes

* conflicts ([f0de5ed](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f0de5edef825a13f27cf0d447feafbe412bec632))
* update dependencies ([#158](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/158)) ([351d8ba](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/351d8ba9b3d91cf6adce34ac4fab9ad57ce39692))
* update from upstream ([#157](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/157)) ([1c3da76](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1c3da76577f7e2a00d0b89014ce7948f33b5c822))

## [1.6.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.5.1...v1.6.0) (2025-06-17)


### Features

* add username ref to mongodb auth ([#155](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/155)) ([57e0338](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/57e03389119deb698748530853519e5c91f1bda0))

## [1.5.1](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.5.0...v1.5.1) (2025-06-16)


### Bug Fixes

* update dependencies ([#151](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/151)) ([d9b7f64](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d9b7f64d1036f07d525e3974715e8d3d4dd68912))
* update from upstream ([#149](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/149)) ([240a423](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/240a42329c4c5f9fca73af0b948abfb8d3adf7f3))
* update go ([f821664](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/f821664e355c75cace0c48dfed18faae25e15fee))
* workflow push steps now works on more PushSecret cases ([#143](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/143)) ([fd0adeb](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/fd0adeba8ba761966e2afcb0df3e260ad00ed6e6))

## [1.5.0](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.8...v1.5.0) (2025-06-13)


### Features

* bump esi-pod-webhook version ([#142](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/142)) ([3439c59](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/3439c594f612f0ae77a9bd4a5a11ca5524c38cbf))

## [1.4.8](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.7...v1.4.8) (2025-06-13)


### Bug Fixes

* conflicts ([80a4fb5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/80a4fb5a3c5cdb115be70d8c089768e7ff8ae81b))
* helm release running  always ([#4898](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/4898)) ([955cc69](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/955cc6944cde26ac7ecf5d716ca077a62a89adb4))
* rebuilds ubi ([80a4fb5](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/80a4fb5a3c5cdb115be70d8c089768e7ff8ae81b))
* update dependencies ([#141](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/141)) ([937b665](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/937b66546132f49f4172cb1db3c8904be1e1957d))
* update from upstream ([1d3eaa1](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1d3eaa14829fbc1fd8447ee126f03d89aa061d64))
* **workflows:** pull step template support ([#138](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/138)) ([1557f6b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/1557f6b2d4efb7fde777eb19eb427de80bab84bf))

## [1.4.7](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.6...v1.4.7) (2025-06-12)


### Bug Fixes

* bump esi-pod-webhook version ([#135](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/135)) ([d118b1b](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/d118b1b58b3c869860be77b6ab11a549bf758117))

## [1.4.6](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.5...v1.4.6) (2025-06-10)


### Bug Fixes

* esi issues ([#130](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/130)) ([58c1185](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/58c11856d7d0c79706ad37d4592da5a81c606039))

## [1.4.5](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.4...v1.4.5) (2025-06-10)


### Bug Fixes

* authentication registration ([#127](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/127)) ([a16bd9c](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/a16bd9c056912b7150719a20ffcc2228c857fa84))
* remove specMap from spiffe auth ([#128](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/128)) ([b19a4f9](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/b19a4f9e9a22f94b99c531ab16eb1ca2c6e0c62d))

## [1.4.4](https://github.com/external-secrets-inc/external-secrets-enterprise/compare/v1.4.3...v1.4.4) (2025-06-09)


### Bug Fixes

* federation tls server ([#125](https://github.com/external-secrets-inc/external-secrets-enterprise/issues/125)) ([ee5206e](https://github.com/external-secrets-inc/external-secrets-enterprise/commit/ee5206e43442780af53cbdbe35a06cc3509f97ce))

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
