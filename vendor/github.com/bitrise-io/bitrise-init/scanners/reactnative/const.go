package reactnative

const (
	deployWorkflowDescription = `Tests, builds and deploys the app using *Deploy to bitrise.io* Step.

Next steps:
- Set up an [Apple service with API key](https://devcenter.bitrise.io/en/accounts/connecting-to-services/connecting-to-an-apple-service-with-api-key.html).
- Check out [Getting started with React Native apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-react-native-apps.html).
`

	primaryWorkflowDescription = `Runs tests.

Next steps:
- Check out [Getting started with React Native apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-react-native-apps.html).
`

	primaryWorkflowNoTestsDescription = `Installs dependencies.

Next steps:
- Add tests to your project and configure the workflow to run them.
- Check out [Getting started with React Native apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-react-native-apps.html).
`
)

const (
	expoDeployWorkflowDescription = `Tests the app and runs a build on Expo Application Services (EAS).

Next steps:
- Configure the ` + "`Run Expo Application Services (EAS) build`" + ` Step's ` + "`Access Token`" + ` input.
- Check out [Getting started with Expo apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-expo-apps.html).
- For an alternative deploy workflow checkout the [(React Native) Expo: Build using Turtle CLI recipe](https://github.com/bitrise-io/workflow-recipes/blob/main/recipes/rn-expo-turtle-build.md).
`

	expoDeployWorkflowNoTestsDescription = `Runs a build on Expo Application Services (EAS).

Next steps:
- Configure the ` + "`Run Expo Application Services (EAS) build`" + ` Step's ` + "`Access Token`" + ` input.
- Check out [Getting started with Expo apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-expo-apps.html).
- For an alternative deploy workflow checkout the [(React Native) Expo: Build using Turtle CLI recipe](https://github.com/bitrise-io/workflow-recipes/blob/main/recipes/rn-expo-turtle-build.md).
`

	expoPrimaryWorkflowDescription = `Runs tests.

Next steps:
- Check out [Getting started with Expo apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-expo-apps.html).
`

	expoPrimaryWorkflowNoTestsDescription = `Installs dependencies.

Next steps:
- Add tests to your project and configure the workflow to run them.
- Check out [Getting started with Expo apps](https://devcenter.bitrise.io/en/getting-started/getting-started-with-expo-apps.html).
`
)
