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
	Entries []DirEntry
}

func ListEntries(rootDir string, depth uint) (*DirEntry, error) {
	if depth == 0 {
		return nil, nil
	}

	parent := DirEntry{
		AbsPath: rootDir,
		RelPath: "./",
		Name:    "",
		IsDir:   true,
		Entries: nil,
	}

	if err := recursiveListEntries(rootDir, &parent, 0, depth); err != nil {
		return nil, err
	}

	return &parent, nil
}

func (e DirEntry) FindEntryByName(name string, isDir bool) *DirEntry {
	for _, entry := range e.Entries {
		if entry.Name == name && entry.IsDir == isDir {
			return &entry
		}
	}
	return nil
}

func (e DirEntry) FindEntryByPath(isDir bool, components ...string) *DirEntry {
	entry := &e
	for i, component := range components {
		var dir bool
		if i == len(components)-1 {
			dir = isDir
		} else {
			dir = true
		}

		entry = entry.FindEntryByName(component, dir)
		if entry == nil {
			return nil
		}
	}
	return entry
}

func (e DirEntry) FindAllEntriesByName(name string, isDir bool) []DirEntry {
	return e.recursiveFindAllEntriesByName(name, isDir)
}

func (e DirEntry) recursiveFindAllEntriesByName(name string, isDir bool) []DirEntry {
	var entries []DirEntry
	for _, entry := range e.Entries {
		if entry.IsDir {
			entries = append(entries, entry.recursiveFindAllEntriesByName(name, isDir)...)
		}
		if entry.Name == name && entry.IsDir == isDir {
			entries = append(entries, entry)
		}
	}
	return entries
}

func recursiveListEntries(rootDir string, parent *DirEntry, currentDepth, maxDepth uint) error {
	if currentDepth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(parent.AbsPath)
	if err != nil {
		// TODO: log error
		return nil
	}

	parent.Entries = make([]DirEntry, 0, len(entries))
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
			Entries: nil,
		}

		if dirEntry.IsDir {
			if err := recursiveListEntries(rootDir, &dirEntry, currentDepth+1, maxDepth); err != nil {
				return err
			}
		}

		parent.Entries = append(parent.Entries, dirEntry)
	}

	return nil
}
