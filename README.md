# Project scanner

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-project-scanner?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-project-scanner/releases)

Scans repository for iOS, macOS, Android, Fastlane, Cordova, Ionic, React Native and Flutter projects

<details>
<summary>Description</summary>

For iOS and macOS projects, the step detects CocoaPods and scans Xcode project files
for valid Xcode command line configurations.

For Android projects, the step checks for build.gradle files and lists all the gradle tasks. It
also checks for gradlew file.

For Fastlane, the step detects Fastfile and lists the available lanes.

For Cordova projects, the step checks for the config.xml file.

For Ionic projects, the step checks for the ionic.config.json and ionic.project files.

For React Native projects, the step checks for package.json files and also runs the
iOS and Android native project scanners.

For Flutter projects, the step checks for the pubspec.yaml files.
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `scan_dir` | The step will look for the projects in this directory. | required | `$BITRISE_SOURCE_DIR` |
| `scan_result_submit_url` | If provided, the scan results will be sent to the given URL, with a POST request.  |  | `$BITRISE_SCAN_RESULT_POST_URL` |
| `scan_result_submit_api_token` | If provided and `scan_result_submit_url` also provided, this API Token will be used for sending the Scan Results.  | sensitive | `$BITRISE_APP_API_TOKEN` |
| `icon_candidates_url` | If provided, the app icons will be uploaded.  |  | `$BITRISE_AVATAR_CANDIDATES_POST_URL` |
| `verbose_log` | You can enable the verbose log for easier debugging.  |  | `false` |
| `enable_repo_clone` | If set to yes then it will setup the ssh key and will clone the repo with the provided url and branch name.  |  | `no` |
| `ssh_rsa_private_key` | SSH key to be used for the git clone. | sensitive | `$SSH_RSA_PRIVATE_KEY` |
| `repository_url` | Url to be used for the git clone. |  | `$GIT_REPOSITORY_URL` |
| `branch` | Branch to be used for the git clone. |  | `$BITRISE_GIT_BRANCH` |
</details>

<details>
<summary>Outputs</summary>
There are no outputs defined in this step
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-project-scanner/pulls) and [issues](https://github.com/bitrise-steplib/steps-project-scanner/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
