// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syncrules

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// PathRule matches file paths using exact matching, glob patterns, or regular expressions.
type PathRule struct {
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	Pattern       string `json:"pattern" yaml:"pattern"` // e.g. "photos/**", "*.iso", "build/*"
	IsRegex       bool   `json:"isRegex,omitempty" yaml:"isRegex,omitempty"`
	CaseSensitive bool   `json:"caseSensitive,omitempty" yaml:"caseSensitive,omitempty"`
	Action        Action `json:"action" yaml:"action"` // include or exclude

	compiledGlob  glob.Glob
	compiledRegex *regexp.Regexp
	parsed        bool
}

// Compile prepares the glob or regex matcher.
func (r *PathRule) Compile() error {
	pat := r.Pattern
	if pat == "" {
		return fmt.Errorf("path rule %q: empty pattern", r.Name)
	}

	if r.IsRegex {
		if !r.CaseSensitive {
			pat = "(?i)" + pat
		}
		re, err := regexp.Compile(pat)
		if err != nil {
			return fmt.Errorf("path rule %q: invalid regex pattern %q: %w", r.Name, r.Pattern, err)
		}
		r.compiledRegex = re
	} else {
		var g glob.Glob
		var err error
		if r.CaseSensitive {
			g, err = glob.Compile(pat, '/')
		} else {
			g, err = glob.Compile(strings.ToLower(pat), '/')
		}
		if err != nil {
			return fmt.Errorf("path rule %q: invalid glob pattern %q: %w", r.Name, r.Pattern, err)
		}
		r.compiledGlob = g
	}

	r.parsed = true
	return nil
}

// Eval evaluates a file entry against the path pattern.
func (r *PathRule) Eval(entry FileEntry) (Action, bool, string) {
	if !r.parsed {
		if err := r.Compile(); err != nil {
			return ActionSkip, false, err.Error()
		}
	}

	norm := entry.NormalizedPath()
	matched := false

	if r.IsRegex {
		matched = r.compiledRegex.MatchString(norm)
	} else {
		target := norm
		if !r.CaseSensitive {
			target = strings.ToLower(target)
		}

		if r.compiledGlob.Match(target) {
			matched = true
		} else if entry.IsDir && r.compiledGlob.Match(target+"/") {
			matched = true
		} else if base := filepath.Base(target); r.compiledGlob.Match(base) {
			matched = true
		}
	}

	if matched {
		reason := fmt.Sprintf("path %q matched pattern %q", norm, r.Pattern)
		return r.Action, true, reason
	}

	return ActionSkip, false, ""
}
