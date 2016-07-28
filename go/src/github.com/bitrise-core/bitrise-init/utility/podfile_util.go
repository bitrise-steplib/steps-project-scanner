package utility

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-utils/cmdex"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
)

const podfileRubyFileContent = `class Podfile
  def method_missing symbol, *args
  end

  def Object.const_missing const, *args
  end

  def uninitialized_constant constant, *args
    puts "Unitialized Constant: #{constant}"
    puts args
  end

  def apply_quotation_fix(str)
    fixed = str.tr('‘', "'")
    fixed = fixed.tr('’', "'")
    fixed = fixed.tr('“', '"')
    fixed = fixed.tr('”', '"')
    return fixed
  end

  def self.from_file(path)
    Podfile.new do
      @full_path = File.expand_path(path)
      @base_dir = File.dirname(@full_path)

      original = File.open(@full_path).read
      fixed = apply_quotation_fix(original)

      eval(fixed, nil, path)
    end
  end

  def initialize(&block)
    @dependencies = []
    instance_eval(&block)
  end

  def target(target_name, *args, &block)
    target_dict = {
      target: target_name,
      project: nil,
      workspace: nil,
      targets: []
    }

    parent_target = @current_target
    @current_target = target_dict

    block.call(self) if block

    if parent_target
      parent_target[:targets] << @current_target
    else
      (@targets ||= []) << @current_target
    end
    @current_target = parent_target
  end

  def project(project, *args)
    project = File.join(File.dirname(project), File.basename(project, File.extname(project)))

    if @current_target
      @current_target[:project] = project
    else
      @base_project = project
    end
  end

  def xcodeproj(project, *args)
    project(project, args)
  end

  def workspace(workspace, *args)
    workspace = File.join(File.dirname(workspace), File.basename(workspace, File.extname(workspace)))

    if @current_target
      @current_target.workspace = workspace
    else
      @base_workspace = workspace
    end
  end

  # Helper

  def fix_targets(dict, parent)
    # If no explicit project is specified, it will use the Xcode project of the parent target.
    if parent != nil
      dict[:project] = parent[:project] unless dict[:project]
    end

    if dict[:project] == nil
      # If none of the target definitions specify an explicit project and there is only one project in the same directory
      # as the Podfile then that project will be used.
      projects = Dir[File.join(@base_dir, "*.xcodeproj")]

      if projects.count == 0
        dict[:error] = "No project found for Podfile at path: #{@base_dir}"
      else
        if projects.count > 1
          dict[:error] = "Multiple projects found for Podfile at path: #{@base_dir}. Check this reference for help: https://guides.cocoapods.org/syntax/podfile.html#xcodeproj"
        end

        dict[:project] = File.basename(projects.first, ".*")
      end
    end

    if dict[:project] != nil
      # Check if the file exists
      project_path = File.join(@base_dir, "#{dict[:project]}.xcodeproj")
      unless File.exists?(project_path)
        dict[:error] = "No project found at path: #{project_path}"
      end

      # If no explicit Xcode workspace is specified and only one project exists in the same directory as the Podfile,
      # then the name of that project is used as the workspace’s name.
      if dict[:workspace] == nil
        if dict[:project] != nil
          dict[:workspace] = File.basename(dict[:project], '.*')
        end
      end
    end

    dict[:targets].each do |t|
      fix_targets(t, dict)
    end

    dict
  end

  def list_targets
    base_target = {
      target: "",
      project: @base_project,
      workspace: @base_workspace,
      targets: @targets || []
    }

    [fix_targets(base_target, nil)]
  end

  def get_workspaces(dict)
    @workspaces ||= {}

    if dict[:error] == nil
      project_path = File.expand_path(File.join(@base_dir, "#{dict[:project]}.xcodeproj"))
      dir = File.dirname(project_path)
      workspace_path = File.join(dir, "#{dict[:workspace]}.xcworkspace")

      @workspaces[workspace_path] = project_path
    else
      puts dict[:error].to_s
      exit(1)
    end

    dict[:targets].each do |target|
      get_workspaces(target)
    end

    @workspaces
  end
end
`

const getWorkspacesRubyFileContent = `require_relative 'podfile'
require 'json'

path = ENV['pod_file_path']

begin
  podfile = Podfile.from_file(path)
  workspaces = podfile.get_workspaces(podfile.list_targets.first)
rescue => ex
  puts(ex.inspect.to_s)
  puts('--- Stack trace: ---')
  puts(ex.backtrace.to_s)
  exit(1)
end

puts workspaces.to_json
`

var (
	logger = NewLogger()
)

func isWorkspaceSpecified(podfileContent string) bool {
	re := regexp.MustCompile(`\s*workspace (.+)`)
	lines := strings.Split(podfileContent, "\n")
	for _, line := range lines {
		if re.FindString(line) != "" {
			return true
		}
	}

	return false
}

func getWorkspaceProjectMap(podfilePth string) (map[string]string, error) {
	// Run simply Podfile anaylzer
	podfileContent, err := fileutil.ReadStringFromFile(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	if !isWorkspaceSpecified(podfileContent) {
		logger.InfofDetails("workspace not specified in podfile (%s)", podfilePth)
		// If no explicit Xcode workspace is specified and
		// only one project exists in the same directory as the Podfile,
		// then the name of that project is used as the workspace’s name.
		podfileDir := filepath.Dir(podfilePth)
		pattern := filepath.Join(podfileDir, "*.xcodeproj")
		projects, err := filepath.Glob(pattern)
		if err != nil {
			return map[string]string{}, err
		}

		if len(projects) > 1 {
			return map[string]string{}, fmt.Errorf("failed to determin workspace name: no workspace specified in the Podfile and more then one xcodeproj exist in Podfile's dir")
		}

		if len(projects) == 1 {
			project := projects[0]
			projectBasename := filepath.Base(project)
			projectName := strings.TrimSuffix(projectBasename, ".xcodeproj")
			workspace := filepath.Join(podfileDir, projectName+".xcworkspace")

			return map[string]string{
				workspace: project,
			}, nil
		}
	}

	logger.Warnf("Workspace specified in podfile (%s)", podfilePth)
	logger.Warnf("Running extended podfile analyzer")

	// Analyze Podfile as a ruby file
	if err := os.Setenv("pod_file_path", podfilePth); err != nil {
		return map[string]string{}, err
	}

	tmpDir, err := pathutil.NormalizedOSTempDirPath("bitrise-init")
	if err != nil {
		return map[string]string{}, err
	}

	podfileRubyFilePath := path.Join(tmpDir, "podfile.rb")
	if err := fileutil.WriteStringToFile(podfileRubyFilePath, podfileRubyFileContent); err != nil {
		return map[string]string{}, err
	}

	getWorkspacesRubyFilePath := path.Join(tmpDir, "get_workspace.rb")
	if err := fileutil.WriteStringToFile(getWorkspacesRubyFilePath, getWorkspacesRubyFileContent); err != nil {
		return map[string]string{}, err
	}

	out, err := cmdex.RunCommandAndReturnCombinedStdoutAndStderr("ruby", getWorkspacesRubyFilePath)
	if err != nil {
		logger.Warnf("Extended analyzer failed against podfile:")
		fmt.Println(podfileContent)

		if errorutil.IsExitStatusError(err) {
			return map[string]string{}, errors.New(out)
		}
		return map[string]string{}, err
	}

	workspaceMap := map[string]string{}
	if err := json.Unmarshal([]byte(out), &workspaceMap); err != nil {
		return map[string]string{}, err
	}

	return workspaceMap, nil
}

// GetRelativeWorkspaceProjectPathMap ...
func GetRelativeWorkspaceProjectPathMap(podfilePth, baseDir string) (map[string]string, error) {
	absPodfilePth, err := pathutil.AbsPath(podfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	workspaceMap, err := getWorkspaceProjectMap(absPodfilePth)
	if err != nil {
		return map[string]string{}, err
	}

	normalizedWorkspaceMap := map[string]string{}
	for workspace, project := range workspaceMap {
		relWorkspacePath, err := filepath.Rel(baseDir, workspace)
		if err != nil {
			return map[string]string{}, err
		}

		relProjectPath, err := filepath.Rel(baseDir, project)
		if err != nil {
			return map[string]string{}, err
		}

		normalizedWorkspaceMap[relWorkspacePath] = relProjectPath
	}

	return normalizedWorkspaceMap, nil
}
