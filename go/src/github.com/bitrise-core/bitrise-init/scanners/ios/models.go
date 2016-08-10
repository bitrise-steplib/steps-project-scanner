package ios

// SchemeModel ...
type SchemeModel struct {
	Name      string
	Shared    bool
	HasXCTest bool
}

// ProjectModel ...
type ProjectModel struct {
	Pth     string
	Schemes []SchemeModel

	PodWorkspace WorkspaceModel
}

// WorkspaceModel ...
type WorkspaceModel struct {
	Pth     string
	Schemes []SchemeModel

	GeneratedByPod bool
}

// FindProjectWithPth ...
func FindProjectWithPth(projects []ProjectModel, pth string) (ProjectModel, bool) {
	for _, project := range projects {
		if project.Pth == pth {
			return project, true
		}
	}
	return ProjectModel{}, false
}

// FindWorkspaceWithPth ...
func FindWorkspaceWithPth(workspaces []WorkspaceModel, pth string) (WorkspaceModel, bool) {
	for _, workspace := range workspaces {
		if workspace.Pth == pth {
			return workspace, true
		}
	}
	return WorkspaceModel{}, false
}
