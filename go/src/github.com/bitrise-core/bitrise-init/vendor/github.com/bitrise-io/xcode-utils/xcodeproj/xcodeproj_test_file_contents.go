package xcodeproj

const (
	schemeContentWithXCTestBuildAction = `<?xml version="1.0" encoding="UTF-8"?>
<Scheme
   LastUpgradeVersion = "0700"
   version = "1.3">
   <BuildAction
      parallelizeBuildables = "YES"
      buildImplicitDependencies = "YES">
      <BuildActionEntries>
         <BuildActionEntry
            buildForTesting = "YES"
            buildForRunning = "YES"
            buildForProfiling = "YES"
            buildForArchiving = "YES"
            buildForAnalyzing = "YES">
            <BuildableReference
               BuildableIdentifier = "primary"
               BlueprintIdentifier = "BAC384091BA9F569005CFE20"
               BuildableName = "BitriseXcode7Sample.app"
               BlueprintName = "BitriseXcode7Sample"
               ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
            </BuildableReference>
         </BuildActionEntry>
      </BuildActionEntries>
   </BuildAction>
   <TestAction
      buildConfiguration = "Debug"
      selectedDebuggerIdentifier = "Xcode.DebuggerFoundation.Debugger.LLDB"
      selectedLauncherIdentifier = "Xcode.DebuggerFoundation.Launcher.LLDB"
      shouldUseLaunchSchemeArgsEnv = "YES">
      <Testables>
         <TestableReference
            skipped = "NO">
            <BuildableReference
               BuildableIdentifier = "primary"
               BlueprintIdentifier = "BAC384221BA9F569005CFE20"
               BuildableName = "BitriseXcode7SampleTests.xctest"
               BlueprintName = "BitriseXcode7SampleTests"
               ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
            </BuildableReference>
         </TestableReference>
         <TestableReference
            skipped = "NO">
            <BuildableReference
               BuildableIdentifier = "primary"
               BlueprintIdentifier = "BAC3842D1BA9F569005CFE20"
               BuildableName = "BitriseXcode7SampleUITests.xctest"
               BlueprintName = "BitriseXcode7SampleUITests"
               ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
            </BuildableReference>
         </TestableReference>
      </Testables>
      <MacroExpansion>
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </MacroExpansion>
      <AdditionalOptions>
      </AdditionalOptions>
   </TestAction>
   <LaunchAction
      buildConfiguration = "Debug"
      selectedDebuggerIdentifier = "Xcode.DebuggerFoundation.Debugger.LLDB"
      selectedLauncherIdentifier = "Xcode.DebuggerFoundation.Launcher.LLDB"
      launchStyle = "0"
      useCustomWorkingDirectory = "NO"
      ignoresPersistentStateOnLaunch = "NO"
      debugDocumentVersioning = "YES"
      debugServiceExtension = "internal"
      allowLocationSimulation = "YES">
      <BuildableProductRunnable
         runnableDebuggingMode = "0">
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </BuildableProductRunnable>
      <AdditionalOptions>
      </AdditionalOptions>
   </LaunchAction>
   <ProfileAction
      buildConfiguration = "Release"
      shouldUseLaunchSchemeArgsEnv = "YES"
      savedToolIdentifier = ""
      useCustomWorkingDirectory = "NO"
      debugDocumentVersioning = "YES">
      <BuildableProductRunnable
         runnableDebuggingMode = "0">
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </BuildableProductRunnable>
   </ProfileAction>
   <AnalyzeAction
      buildConfiguration = "Debug">
   </AnalyzeAction>
   <ArchiveAction
      buildConfiguration = "Release"
      revealArchiveInOrganizer = "YES">
   </ArchiveAction>
</Scheme>
`

	schemeContentWithoutXCTestBuildAction = `<?xml version="1.0" encoding="UTF-8"?>
<Scheme
   LastUpgradeVersion = "0730"
   version = "1.3">
   <BuildAction
      parallelizeBuildables = "YES"
      buildImplicitDependencies = "YES">
      <BuildActionEntries>
         <BuildActionEntry
            buildForTesting = "YES"
            buildForRunning = "YES"
            buildForProfiling = "YES"
            buildForArchiving = "YES"
            buildForAnalyzing = "YES">
            <BuildableReference
               BuildableIdentifier = "primary"
               BlueprintIdentifier = "BAC384091BA9F569005CFE20"
               BuildableName = "BitriseXcode7Sample.app"
               BlueprintName = "BitriseXcode7Sample"
               ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
            </BuildableReference>
         </BuildActionEntry>
      </BuildActionEntries>
   </BuildAction>
   <TestAction
      buildConfiguration = "Debug"
      selectedDebuggerIdentifier = "Xcode.DebuggerFoundation.Debugger.LLDB"
      selectedLauncherIdentifier = "Xcode.DebuggerFoundation.Launcher.LLDB"
      shouldUseLaunchSchemeArgsEnv = "YES">
      <Testables>
      </Testables>
      <MacroExpansion>
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </MacroExpansion>
      <AdditionalOptions>
      </AdditionalOptions>
   </TestAction>
   <LaunchAction
      buildConfiguration = "Debug"
      selectedDebuggerIdentifier = "Xcode.DebuggerFoundation.Debugger.LLDB"
      selectedLauncherIdentifier = "Xcode.DebuggerFoundation.Launcher.LLDB"
      launchStyle = "0"
      useCustomWorkingDirectory = "NO"
      ignoresPersistentStateOnLaunch = "NO"
      debugDocumentVersioning = "YES"
      debugServiceExtension = "internal"
      allowLocationSimulation = "YES">
      <BuildableProductRunnable
         runnableDebuggingMode = "0">
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </BuildableProductRunnable>
      <AdditionalOptions>
      </AdditionalOptions>
   </LaunchAction>
   <ProfileAction
      buildConfiguration = "Release"
      shouldUseLaunchSchemeArgsEnv = "YES"
      savedToolIdentifier = ""
      useCustomWorkingDirectory = "NO"
      debugDocumentVersioning = "YES">
      <BuildableProductRunnable
         runnableDebuggingMode = "0">
         <BuildableReference
            BuildableIdentifier = "primary"
            BlueprintIdentifier = "BAC384091BA9F569005CFE20"
            BuildableName = "BitriseXcode7Sample.app"
            BlueprintName = "BitriseXcode7Sample"
            ReferencedContainer = "container:BitriseXcode7Sample.xcodeproj">
         </BuildableReference>
      </BuildableProductRunnable>
   </ProfileAction>
   <AnalyzeAction
      buildConfiguration = "Debug">
   </AnalyzeAction>
   <ArchiveAction
      buildConfiguration = "Release"
      revealArchiveInOrganizer = "YES">
   </ArchiveAction>
</Scheme>
`

	pbxNativeTargetSectionWithSpace = `/* Begin PBXTargetDependency section */
		BADDFA051A703F87004C3526 /* PBXTargetDependency */ = {
			isa = PBXTargetDependency;
			target = BADDF9E61A703F87004C3526 /* BitriseSampleAppsiOS With Spaces */;
			targetProxy = BADDFA041A703F87004C3526 /* PBXContainerItemProxy */;
		};
/* End PBXTargetDependency section */

/* Begin PBXNativeTarget section */
		BADDF9E61A703F87004C3526 /* BitriseSampleAppsiOS With Spaces */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = BADDFA0D1A703F87004C3526 /* Build configuration list for PBXNativeTarget "BitriseSampleAppsiOS With Spaces" */;
			buildPhases = (
				BADDF9E31A703F87004C3526 /* Sources */,
				BADDF9E41A703F87004C3526 /* Frameworks */,
				BADDF9E51A703F87004C3526 /* Resources */,
			);
			buildRules = (
			);
			dependencies = (
			);
			name = "BitriseSampleAppsiOS With Spaces";
			productName = "BitriseSampleAppsiOS With Spaces";
			productReference = BADDF9E71A703F87004C3526 /* BitriseSampleAppsiOS With Spaces.app */;
			productType = "com.apple.product-type.application";
		};
		BADDFA021A703F87004C3526 /* BitriseSampleAppsiOS With SpacesTests */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = BADDFA101A703F87004C3526 /* Build configuration list for PBXNativeTarget "BitriseSampleAppsiOS With SpacesTests" */;
			buildPhases = (
				BADDF9FF1A703F87004C3526 /* Sources */,
				BADDFA001A703F87004C3526 /* Frameworks */,
				BADDFA011A703F87004C3526 /* Resources */,
			);
			buildRules = (
			);
			dependencies = (
				BADDFA051A703F87004C3526 /* PBXTargetDependency */,
			);
			name = "BitriseSampleAppsiOS With SpacesTests";
			productName = "BitriseSampleAppsiOS With SpacesTests";
			productReference = BADDFA031A703F87004C3526 /* BitriseSampleAppsiOS With SpacesTests.xctest */;
			productType = "com.apple.product-type.bundle.unit-test";
		};
/* End PBXNativeTarget section */
`

	pbxProjContentChunk = `// !$*UTF8*$!
{
	archiveVersion = 1;
	classes = {
	};
	objectVersion = 46;
	objects = {

/* Begin PBXTargetDependency section */
		BAAFFEEF19EE788800F3AC91 /* PBXTargetDependency */ = {
			isa = PBXTargetDependency;
			target = BAAFFED019EE788800F3AC91 /* SampleAppWithCocoapods */;
			targetProxy = BAAFFEEE19EE788800F3AC91 /* PBXContainerItemProxy */;
		};
/* End PBXTargetDependency section */

/* Begin PBXNativeTarget section */
		BAAFFED019EE788800F3AC91 /* SampleAppWithCocoapods */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = BAAFFEF719EE788800F3AC91 /* Build configuration list for PBXNativeTarget "SampleAppWithCocoapods" */;
			buildPhases = (
				BAAFFECD19EE788800F3AC91 /* Sources */,
				BAAFFECE19EE788800F3AC91 /* Frameworks */,
				BAAFFECF19EE788800F3AC91 /* Resources */,
			);
			buildRules = (
			);
			dependencies = (
			);
			name = SampleAppWithCocoapods;
			productName = SampleAppWithCocoapods;
			productReference = BAAFFED119EE788800F3AC91 /* SampleAppWithCocoapods.app */;
			productType = "com.apple.product-type.application";
		};
		BAAFFEEC19EE788800F3AC91 /* SampleAppWithCocoapodsTests */ = {
			isa = PBXNativeTarget;
			buildConfigurationList = BAAFFEFA19EE788800F3AC91 /* Build configuration list for PBXNativeTarget "SampleAppWithCocoapodsTests" */;
			buildPhases = (
				75ACE584234D974D15C5CAE9 /* Check Pods Manifest.lock */,
				BAAFFEE919EE788800F3AC91 /* Sources */,
				BAAFFEEA19EE788800F3AC91 /* Frameworks */,
				BAAFFEEB19EE788800F3AC91 /* Resources */,
				D0F06DBF2FED4262AA6DE7DB /* Copy Pods Resources */,
			);
			buildRules = (
			);
			dependencies = (
				BAAFFEEF19EE788800F3AC91 /* PBXTargetDependency */,
			);
			name = SampleAppWithCocoapodsTests;
			productName = SampleAppWithCocoapodsTests;
			productReference = BAAFFEED19EE788800F3AC91 /* SampleAppWithCocoapodsTests.xctest */;
			productType = "com.apple.product-type.bundle.unit-test";
		};
/* End PBXNativeTarget section */

/* Begin PBXVariantGroup section */
		BAAFFEE119EE788800F3AC91 /* Main.storyboard */ = {
			isa = PBXVariantGroup;
			children = (
				BAAFFEE219EE788800F3AC91 /* Base */,
			);
			name = Main.storyboard;
			sourceTree = "<group>";
		};
		BAAFFEE619EE788800F3AC91 /* LaunchScreen.xib */ = {
			isa = PBXVariantGroup;
			children = (
				BAAFFEE719EE788800F3AC91 /* Base */,
			);
			name = LaunchScreen.xib;
			sourceTree = "<group>";
		};
/* End PBXVariantGroup section */

	rootObject = BAAFFEC919EE788800F3AC91 /* Project object */;
}
`

	pbxTargetDependencies = `
 /* End PBXSourcesBuildPhase section */

/* Begin PBXTargetDependency section */
		BAAFFEEF19EE788800F3AC91 /* PBXTargetDependency */ = {
			isa = PBXTargetDependency;
			target = BAAFFED019EE788800F3AC91 /* SampleAppWithCocoapods */;
			targetProxy = BAAFFEEE19EE788800F3AC91 /* PBXContainerItemProxy */;
		};
/* End PBXTargetDependency section */

/* Begin PBXVariantGroup section */
`
)
