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

// Action defines the synchronization decision returned by a rule or rule set.
type Action int

const (
	// ActionSkip indicates the rule does not apply; continue evaluating next rules.
	ActionSkip Action = iota
	// ActionInclude indicates the file should be synchronized.
	ActionInclude
	// ActionExclude indicates the file should be excluded / ignored from sync.
	ActionExclude
)

func (a Action) String() string {
	switch a {
	case ActionInclude:
		return "include"
	case ActionExclude:
		return "exclude"
	case ActionSkip:
		return "skip"
	default:
		return fmt.Sprintf("Action(%d)", a)
	}
}

// MarshalText implements encoding.TextMarshaler for Action.
func (a Action) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for Action.
func (a *Action) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "include", "allow", "sync":
		*a = ActionInclude
	case "exclude", "ignore", "drop", "skip-file":
		*a = ActionExclude
	case "skip", "none", "pass", "":
		*a = ActionSkip
	default:
		return fmt.Errorf("unknown sync rule action: %q", string(text))
	}
	return nil
}

// MatchResult captures the outcome of evaluating a file against a RuleSet.
type MatchResult struct {
	Action   Action
	Matched  bool
	RuleName string
	Reason   string
}
