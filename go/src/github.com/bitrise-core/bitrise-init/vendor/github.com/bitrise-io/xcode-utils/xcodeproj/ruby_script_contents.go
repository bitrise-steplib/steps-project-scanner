package xcodeproj

const xcodeprojGemfileContent = `source 'https://rubygems.org'
	
gem 'xcodeproj'
`

const recreateUserSchemesRubyScriptContent = `require 'xcodeproj'

project_path = ENV['project_path']

begin
  raise 'empty path' if project_path.empty?

  project = Xcodeproj::Project.open project_path

  #-----
  # Separate targets
  native_targets = project.native_targets

  build_targets = []
  test_targets = []

  native_targets.each do |target|
    test_targets << target if target.test_target_type?
    build_targets << target unless target.test_target_type?
  end

  raise 'no build target found' unless build_targets.count

  #-----
  # Map targets
  target_mapping = {}

  build_targets.each do |target|
    target_mapping[target] = []
  end

  test_targets.each do |target|
    target_dependencies = target.dependencies

    dependent_targets = []
    target_dependencies.each do |target_dependencie|
      dependent_targets << target_dependencie.target
    end

    dependent_targets.each do |dependent_target|
      if build_targets.include? dependent_target
        target_mapping[dependent_target] = [] unless target_mapping[dependent_target]
        target_mapping[dependent_target] << target
      end
    end
  end

  #-----
  # Create schemes
  target_mapping.each do |build_t, test_ts|
    scheme = Xcodeproj::XCScheme.new

    scheme.set_launch_target build_t
    scheme.add_build_target build_t

    test_ts.each do |test_t|
      scheme.add_test_target test_t
    end

    scheme.save_as project_path, build_t.name
  end
rescue => ex
  puts ex.inspect.to_s
  puts '--- Stack trace: ---'
  puts ex.backtrace.to_s
  exit 1
end
`

const projectBuildTargetTestTargetsMapRubyScriptContent = `
require 'xcodeproj'
require 'json'

project_path = ENV['project_path']

begin
  raise 'empty path' if project_path.empty?

  project = Xcodeproj::Project.open project_path

  #-----
  # Separate targets
  native_targets = project.native_targets

  build_targets = []
  test_targets = []

  native_targets.each do |target|
    test_targets << target if target.test_target_type?
    build_targets << target unless target.test_target_type?
  end

  raise 'no build target found' unless build_targets.count

  #-----
  # Map targets
  target_mapping = {}

  build_targets.each do |target|
    target_mapping[target.name] = []
  end

  test_targets.each do |target|
    target_dependencies = target.dependencies

    dependent_targets = []
    target_dependencies.each do |target_dependencie|
      dependent_targets << target_dependencie.target
    end

    dependent_targets.each do |dependent_target|
      if build_targets.include? dependent_target
        target_mapping[dependent_target.name] = [] unless target_mapping[dependent_target.name]
        target_mapping[dependent_target.name] << target.name
      end
    end
  end

  puts target_mapping.to_json
rescue => ex
  puts ex.inspect.to_s
  puts '--- Stack trace: ---'
  puts ex.backtrace.to_s
  exit 1
end
`
