## Changelog (Current version: 1.4.0)

-----------------

### 1.4.0 (2017 Jun 09)

* [60f3302] Prepare for 1.4.0
* [b6b3f54] step version updates
* [96a8056] readme update
* [8de5f82] seperated android workflows (#92)
* [57a6222] android local.properties warning fix, xcode cocoapods do not fail if analyze failed, utility cleanup (#91)
* [487630a] error if Android `local.properties` is committed into the repo - completed (#87)
* [6181c14] Added cache step to all xcode workflows (#90)
* [cc83bdc] Added Cache:Pull and Cache:Push steps for the android scanner (#88)

### 1.3.0 (2017 May 31)

* [d9c9e82] prepare for 1.3.0
* [8e3aa9f] Ionic (#89)
* [4144246] go test -v for integration tests

### 1.2.0 (2017 May 17)

* [42bff70] prepare for 1.2.0
* [767f9ca] godeps-update, Release notes in README.md, bitrise.yml update (#86)
* [27978a1] remove change work-dir step from android workflows (#85)
* [9cabca5] define customeConfigName & CustomProjectType as const (#84)
* [e9893f9] add default target as value to cordova config
* [9968152] gradle runner update
* [4b3192e] fail if no platform detected (#83)
* [398e070] cordova scanner & general revision (#82)
* [6b22a64] Added change-workdir in android workflows (#81)
* [29363cd] public FilterRootBuildGradleFiles function
* [f50c6f0] godeps update (#80)
* [76885b9] dependency fix
* [9d8de83] step updates (#79)
* [a699087] do not fail if no platform detected (#78)
* [8122079] Shared xcode scanner (#76)
* [39458b2] gitignore check fix (#75)

### 1.1.2 (2017 Feb 02)

* [2310097] prepare for 1.1.2
* [1fe7afe] step version updates (#73)

### 1.1.1 (2017 Feb 01)

* [384aa95] prepare for 1.1.1
* [f6ac946] test update
* [f8c5eac] Fastfile inspect fixes (#72)

### 1.1.0 (2017 Jan 25)

* [942eddb] tree file list, if no platform detetcted (#71)
* [191cb9a] Merge branch 'master' of github.com:bitrise-core/bitrise-init
* [7ce11fd] Podfile fix (#70)
* [9d0de5b] prepare for 1.1.0
* [f7475d3] step version updates (#69)
* [e3e400d] Carthage (#68)
* [ece4643] step version update (#67)

### 1.0.0 (2017 Jan 17)

* [065ff67] prepare for 1.0.0
* [bfe6444] InstallMissingAndroidTools step (#66)
* [579eae8] ListPathInDirSortedByComponents allow to list with/without rel paths (#65)
* [0383a2d] godeps-update (#64)
* [ab226f9] custom config -> other config (#63)

### 0.12.0 (2017 Jan 09)

* [4abfb33] prepare for 0.12.0
* [5d708a2] custom config test (#62)
* [f19f162] step version updates
* [7aafd86] warn if xcshareddata is gitignored (#61)
* [570a2a9] Review (#60)
* [750ad77] shared scheme link fix (#59)
* [a84ca68] trigger map fix (#58)
* [b978e1a] Android deps (#57)
* [e8f1347] Macos (#56)
* [60ec76f] Podfile fix (#55)
* [24faf9c] Plugin (#54)
* [a96564a] bump xamarin-archive version
* [577e779] update android extra packages step (#53)

### 0.11.1 (2016 Nov 02)

* [ea4f563] workflow refractors
* [35590f0] prepare for 0.11.1
* [03a8785] bump format version to 1.3.1, use xamarin-archive step instead of xam… (#52)

### 0.11.0 (2016 Oct 11)

* [5544b87] prepare for 0.11.0
* [a828292] godeps update (#51)
* [33ed07c] bitrise.yml update
* [64db30a] Deploy fix (#50)

### 0.10.1 (2016 Sep 30)

* [b3f7924] prepare for 1.10.1
* [4fdad80] step versions (#49)

### 0.10.0 (2016 Sep 26)

* [78be37d] prepare for 0.10.0
* [812974d] format version update to 1.3.0, trigger map format update,  (#48)
* [1f25f6e] godeps update (#47)
* [91e379d] Warnings (#46)

### 0.9.16 (2016 Sep 16)

* [56018e2] prepare for 0.9.16
* [5038efb] ste version updates (#45)

### 0.9.15 (2016 Sep 08)

* [f4c3052] prepare for 0.9.15
* [309935d] recreate-user-schemes step version update (#44)
* [6c0b221] changelog update

### 0.9.14 (2016 Sep 08)

* [ba611ae] prepare for 0.9.14
* [0107d78] fixed scheme generation if no shared scheme found

### 0.9.13 (2016 Sep 07)

* [48c10ed] step version updates (#42)
* [e3d1e61] prepare for 0.9.13
* [a0685c6] step version updates (#41)

### 0.9.12 (2016 Aug 10)

* [405e2ef] step versions
* [25f83f0] prepare for 0.9.12
* [c080c79] godep update ios fixes (#40)
* [6e2dd87] add script step to default workflow (#39)
* [a1cecc1] xcodeproj and xcworkspace should be a dir (#38)

### 0.9.11 (2016 Jul 29)

* [1bad745] prepare for 0.9.11
* [1d56f71] typo fix

### 0.9.10 (2016 Jul 29)

* [ca94c8c] prepare for 0.9.10
* [f541e09] fastalne test, logging updates (#37)
* [0253f44] logging updates, godep update (#36)

### 0.9.9 (2016 Jul 28)

* [e68da49] prepare for 0.9.9
* [02a019d] step version updates (#35)

### 0.9.8 (2016 Jul 28)

* [e126a76] prepare for 0.9.8
* [7cf7fe7] Xcode util, recreate-user schemes (#34)
* [14c97b6] godep update (#33)
* [10c79d8] podfile fix (#32)
* [06f5280] FASTLANE_XCODE_LIST_TIMEOUT (#31)
* [3eca702] improved find podfiles (#30)

### 0.9.7 (2016 Jul 19)

* [e8d39da] prepare for 0.9.7
* [483f065] Merge pull request #29 from bitrise-core/step_version_update
* [77056a3] version updates
* [98176e0] Merge pull request #28 from bitrise-core/user_management
* [76ad7a1] typo
* [8604486] xamarin user management step fix

### 0.9.6 (2016 Jul 05)

* [4bbabbb] prepare for 0.9.6
* [6ca209a] Merge pull request #26 from bitrise-core/framework_filter
* [4037f78] frameworks filter
* [3c31b91] Merge pull request #27 from bitrise-core/script
* [1651bd2] capitalize script step

### 0.9.5 (2016 Jul 01)

* [5e014cd] prepare for 0.9.5
* [9833ab0] Merge pull request #25 from bitrise-core/step_versions
* [132b4bf] Merge pull request #24 from bitrise-core/cli_package_deprecation_fix
* [75ba9d6] step version updates
* [21ff98c] cli package updates

### 0.9.4 (2016 Jul 01)

* [5a5c7a7] prepare for 0.9.4
* [261c144] Merge pull request #22 from bitrise-core/pod_fix
* [61c62dd] workspace regexp fix, test updates, smart quotes test, godep update, pod defined workspace-project mapping fix, pod analyzer fix
* [5672cfb] Merge pull request #23 from bitrise-core/script_step
* [69a7c4c] add title to script step

### 0.9.3 (2016 Jun 27)

* [2d256d2] Merge pull request #21 from bitrise-core/step_version_updates
* [9db4d29] step version updates
* [7fc6581] Merge pull request #20 from bitrise-core/script_step
* [030f85f] Merge pull request #19 from bitrise-core/warnings
* [08665a9] script step added to workflows
* [7dd2b53] test updates
* [5cb4737] warnings: android, ios, fastlane
* [cf0da59] Merge pull request #18 from bitrise-core/chartage_fix
* [9847d3f] Merge pull request #17 from bitrise-core/cocoapods
* [84bf99d] PR fix
* [21c1810] embedded, pod, carthage project filters
* [3f61190] podfile quotation fix
* [39d8991] Merge pull request #16 from bitrise-core/gradle_tasks
* [530b353] default gradle tasks instead of calling gradle tasks cmd
* [65694f1] Merge pull request #15 from bitrise-core/refactor
* [57d5975] refactors
* [a48a969] Merge pull request #14 from bitrise-core/fastlane_work_dir
* [2b60954] fastlane workdir update
* [ce84fbc] fastlane work dir fix

### 0.9.2 (2016 Jun 08)

* [01b88cb] prepare for 0.9.2
* [9ee2bf5] Merge pull request #13 from bitrise-core/workspace_fix
* [4ba0814] step version updates
* [dc3b8c3] inspect fastfile updates
* [ca427db] cocoapods workspace fix
* [c94001c] standalone workspace fix
* [f00b3b7] Merge pull request #12 from bitrise-core/build_gradle_fix
* [9edb298] inspect only top-level build.gradle
* [e128c6e] Merge pull request #11 from bitrise-core/gradlew_fix
* [e9cba9f] root gradlew path fix
* [b77126b] gitignore

### 0.9.1 (2016 Jun 02)

* [0aff8d3] release workflow fix
* [48ae75c] Merge pull request #10 from bitrise-core/scanner_packages
* [1cb400b] missing errcheck fix
* [66a1ae8] ci workflow update, release workflows
* [eb604ed] version command, tool versioning
* [3fd1723] code style
* [27859d2] steps package, refactors
* [67dfb3b] scanners moved to separated packages

-----------------

Updated: 2017 Jun 09