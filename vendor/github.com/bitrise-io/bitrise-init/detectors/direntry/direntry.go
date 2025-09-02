package direntry

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var ignoreDirs = []string{".git", ".github", ".gradle", ".idea", "build", ".kotlin", ".fleet", "CordovaLib", "node_modules"}

type DirEntry struct {
	AbsPath string
	RelPath string
	Name    string
	IsDir   bool
	parent  *DirEntry
	entries []DirEntry
}

// WalkDir walks the directory tree starting from rootDir and returns a DirEntry representing the root directory.
func WalkDir(rootDir string, depth uint) (*DirEntry, error) {
	if depth == 0 {
		return nil, nil
	}

	parent := DirEntry{
		AbsPath: rootDir,
		RelPath: "./",
		Name:    "",
		IsDir:   true,
		parent:  nil,
		entries: nil,
	}

	if err := recursiveWalkDir(rootDir, &parent, 0, depth); err != nil {
		return nil, err
	}

	return &parent, nil
}

// Parent returns the parent directory entry of the current entry.
func (e DirEntry) Parent() *DirEntry {
	return e.parent
}

// FindFirstEntryByName returns the first entry (shortest file path) with the specified name and directory status.
func (e DirEntry) FindFirstEntryByName(name string, isDir bool) *DirEntry {
	return recursiveFindFirstEntryByName([]DirEntry{e}, name, isDir)
}

// FindFirstFileEntryByExtension returns the first file entry (shortest file path) with the specified extension.
func (e DirEntry) FindFirstFileEntryByExtension(extension string) *DirEntry {
	return recursiveFindFirstFileEntryByExtension([]DirEntry{e}, extension)
}

// FindImmediateChildByName returns the immediate child entry with the specified name and directory status.
func (e DirEntry) FindImmediateChildByName(name string, isDir bool) *DirEntry {
	for _, entry := range e.entries {
		if entry.Name == name && entry.IsDir == isDir {
			return &entry
		}
	}
	return nil
}

// FindEntryByPathComponents returns the entry at the path specified by the components.
func (e DirEntry) FindEntryByPathComponents(isDir bool, components ...string) *DirEntry {
	entry := &e
	for i, component := range components {
		var dir bool
		if i == len(components)-1 {
			dir = isDir
		} else {
			dir = true
		}

		entry = entry.FindImmediateChildByName(component, dir)
		if entry == nil {
			return nil
		}
	}
	return entry
}

// FindAllEntriesByName returns all entries with the specified name and directory status.
func (e DirEntry) FindAllEntriesByName(name string, isDir bool) []DirEntry {
	var matchingEntries []DirEntry
	return recursiveFindAllEntriesByName([]DirEntry{e}, matchingEntries, name, isDir)
}

func recursiveFindFirstEntryByName(dirEntries []DirEntry, name string, isDir bool) *DirEntry {
	if len(dirEntries) == 0 {
		return nil
	}

	var nextDirEntries []DirEntry
	for _, dirEntry := range dirEntries {
		for _, entry := range dirEntry.entries {
			if entry.Name == name && entry.IsDir == isDir {
				return &entry
			}
			if entry.IsDir {
				nextDirEntries = append(nextDirEntries, entry)
			}
		}
	}

	return recursiveFindFirstEntryByName(nextDirEntries, name, isDir)
}

func recursiveFindFirstFileEntryByExtension(dirEntries []DirEntry, extension string) *DirEntry {
	if len(dirEntries) == 0 {
		return nil
	}

	var nextDirEntries []DirEntry
	for _, dirEntry := range dirEntries {
		for _, entry := range dirEntry.entries {
			if filepath.Ext(entry.Name) == extension {
				return &entry
			}
			if entry.IsDir {
				nextDirEntries = append(nextDirEntries, entry)
			}
		}
	}

	return recursiveFindFirstFileEntryByExtension(nextDirEntries, extension)
}

func recursiveFindAllEntriesByName(dirEntries []DirEntry, matchingDirEntries []DirEntry, name string, isDir bool) []DirEntry {
	if len(dirEntries) == 0 {
		return matchingDirEntries
	}

	var nextDirEntries []DirEntry
	for _, dirEntry := range dirEntries {
		for _, entry := range dirEntry.entries {
			if entry.Name == name && entry.IsDir == isDir {
				matchingDirEntries = append(matchingDirEntries, entry)
			}
			if entry.IsDir {
				nextDirEntries = append(nextDirEntries, entry)
			}
		}
	}

	return recursiveFindAllEntriesByName(nextDirEntries, matchingDirEntries, name, isDir)
}

func recursiveWalkDir(rootDir string, parent *DirEntry, currentDepth, maxDepth uint) error {
	if currentDepth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(parent.AbsPath)
	if err != nil {
		return err
	}

	parent.entries = make([]DirEntry, 0, len(entries))
	for _, entry := range entries {
		if slices.Contains(ignoreDirs, entry.Name()) {
			continue
		}

		entryAbsPath := filepath.Join(parent.AbsPath, entry.Name())
		dirEntry := DirEntry{
			AbsPath: entryAbsPath,
			RelPath: "./" + filepath.Join("./", strings.TrimPrefix(entryAbsPath, rootDir)),
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			parent:  parent,
			entries: nil,
		}

		if dirEntry.IsDir {
			if err := recursiveWalkDir(rootDir, &dirEntry, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}

		parent.entries = append(parent.entries, dirEntry)
	}

	return nil
}
