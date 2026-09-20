// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syncrules

import (
	"fmt"
	"strings"
)

// GitMode defines how git repositories and their metadata are handled.
type GitMode string

const (
	// GitExcludeMetadata excludes .git directories and internal git structures everywhere.
	GitExcludeMetadata GitMode = "exclude-metadata"
	// GitExcludeRepo excludes entire git repositories (both working tree and .git).
	GitExcludeRepo GitMode = "exclude-repo"
	// GitIncludeAll synchronizes all files including .git directory internals.
	GitIncludeAll GitMode = "include-all"
)

// GitRule controls synchronization of git repositories, metadata, submodules, and worktrees.
type GitRule struct {
	Name              string  `json:"name,omitempty" yaml:"name,omitempty"`
	Mode              GitMode `json:"mode" yaml:"mode"` // exclude-metadata | exclude-repo | include-all
	ExcludeSubmodules bool    `json:"excludeSubmodules,omitempty" yaml:"excludeSubmodules,omitempty"`
	ExcludeWorktrees  bool    `json:"excludeWorktrees,omitempty" yaml:"excludeWorktrees,omitempty"`
	Action            Action  `json:"action,omitempty" yaml:"action,omitempty"` // Defaults based on Mode
}

// Compile validates git rule settings.
func (r *GitRule) Compile() error {
	switch r.Mode {
	case GitExcludeMetadata, GitExcludeRepo, GitIncludeAll, "":
		// valid
	default:
		return fmt.Errorf("git rule %q: unknown mode %q (supported: exclude-metadata, exclude-repo, include-all)", r.Name, r.Mode)
	}
	return nil
}

// IsGitInternal returns true if the path points to or is inside a .git directory.
func IsGitInternal(path string) bool {
	clean := strings.TrimPrefix(path, "/")
	if clean == ".git" {
		return true
	}
	if strings.HasPrefix(clean, ".git/") {
		return true
	}
	if strings.Contains(clean, "/.git/") || strings.HasSuffix(clean, "/.git") {
		return true
	}
	return false
}

// Eval evaluates a file entry against git repository rules.
func (r *GitRule) Eval(entry FileEntry) (Action, bool, string) {
	norm := entry.NormalizedPath()
	isInternal := IsGitInternal(norm)

	switch r.Mode {
	case GitExcludeRepo:
		// If mode is exclude-repo, exclude entire repo or .git internal
		if isInternal {
			return ActionExclude, true, "git metadata excluded by git rule (exclude-repo)"
		}
	case GitExcludeMetadata, "":
		// Exclude .git internals, allow working tree
		if isInternal {
			action := r.Action
			if action == ActionSkip {
				action = ActionExclude
			}
			return action, true, "git internal metadata excluded by git rule"
		}
	case GitIncludeAll:
		if isInternal {
			return ActionInclude, true, "git metadata included by git rule"
		}
	}

	if r.ExcludeSubmodules && (norm == ".gitmodules" || strings.HasSuffix(norm, "/.gitmodules")) {
		return ActionExclude, true, "git submodule configuration excluded"
	}

	return ActionSkip, false, ""
}
