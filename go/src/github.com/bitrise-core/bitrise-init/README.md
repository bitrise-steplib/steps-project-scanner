# Bitrise Init Tool

Initialize bitrise config, step template or plugin template

## How to release new bitrise-init version

- update the step versions in steps/const.go
- bump `RELEASE_VERSION` in bitrise.yml
- comit these changes
- call `bitrise run create-release`
- check and update the generated CHANGELOG.md
- test the generated binaries in _bin/ directory
- push these changes to the master branch
- once `create-release` workflow finishes on bitrise.io create a github release with the generate binaries

__Update the [project-scanner step](https://github.com/bitrise-steplib/steps-project-scanner)__

- update bitrise-init dependency
- share a new version into the steplib (check the [README.md](https://github.com/bitrise-steplib/steps-project-scanner/blob/master/README.md))

__Update the [bitrise init plugin]((https://github.com/bitrise-core/bitrise-plugins-init))__

- update bitrise-init dependency
- release a new version (check the [README.md](https://github.com/bitrise-core/bitrise-plugins-init/blob/master/README.md))

