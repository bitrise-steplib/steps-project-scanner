# Bitrise Init Tool

Initialize bitrise config, step template or plugin template

## How to release new bitrise-init version

- update the step versions in steps/const.go
- bump `RELEASE_VERSION` in bitrise.yml
- commit these changes
- call `bitrise run create-release`
- check and update the generated CHANGELOG.md
- test the generated binaries in _bin/ directory
- push these changes to the master branch
- once `create-release` workflow finishes on bitrise.io test the build generated binaries
- create a github release with the build generated binaries

__Update manual config on website__

- use the generated binaries in `./_bin/` directory to generate the manual config by calling: `BIN_PATH --ci manual-config` this will generate the manual.config.yml at: `CURRENT_DIR/_defaults/result.yml`
- throw the generated `result.yml` to the frontend team, to update the manual-config on the website
- once they put the new config in the website project, check the git changes to make sure, everything looks great

__Update the [project-scanner step](https://github.com/bitrise-steplib/steps-project-scanner)__

- update bitrise-init dependency
- share a new version into the steplib (check the [README.md](https://github.com/bitrise-steplib/steps-project-scanner/blob/master/README.md))

__Update the [bitrise init plugin]((https://github.com/bitrise-core/bitrise-plugins-init))__

- update bitrise-init dependency
- release a new version (check the [README.md](https://github.com/bitrise-core/bitrise-plugins-init/blob/master/README.md))