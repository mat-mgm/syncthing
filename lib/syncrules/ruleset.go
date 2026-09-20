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

// RuleSet contains a collection of filtering and synchronization rules evaluated with precedence.
type RuleSet struct {
	DefaultAction Action     `json:"defaultAction,omitempty" yaml:"defaultAction,omitempty"`
	GitRules      []GitRule  `json:"gitRules,omitempty" yaml:"gitRules,omitempty"`
	PathRules     []PathRule `json:"pathRules,omitempty" yaml:"pathRules,omitempty"`
	SizeRules     []SizeRule `json:"sizeRules,omitempty" yaml:"sizeRules,omitempty"`
	TypeRules     []TypeRule `json:"typeRules,omitempty" yaml:"typeRules,omitempty"`

	compiled bool
}

// NewRuleSet creates an initialized RuleSet with default Include policy.
func NewRuleSet() *RuleSet {
	return &RuleSet{
		DefaultAction: ActionInclude,
		GitRules:      make([]GitRule, 0),
		PathRules:     make([]PathRule, 0),
		SizeRules:     make([]SizeRule, 0),
		TypeRules:     make([]TypeRule, 0),
	}
}

// Compile validates and pre-compiles all inner rules for high evaluation throughput.
func (rs *RuleSet) Compile() error {

	for i := range rs.GitRules {
		if err := rs.GitRules[i].Compile(); err != nil {
			return err
		}
	}
	for i := range rs.PathRules {
		if err := rs.PathRules[i].Compile(); err != nil {
			return err
		}
	}
	for i := range rs.SizeRules {
		if err := rs.SizeRules[i].Compile(); err != nil {
			return err
		}
	}
	for i := range rs.TypeRules {
		if err := rs.TypeRules[i].Compile(); err != nil {
			return err
		}
	}

	rs.compiled = true
	return nil
}

// Evaluate runs all rules against a file entry in priority sequence:
// 1. Path Rules (highest explicit priority)
// 2. Git Rules
// 3. Size Rules
// 4. Type / Extension Rules
// 5. Default Action
func (rs *RuleSet) Evaluate(entry FileEntry) MatchResult {
	if rs == nil {
		return MatchResult{Action: ActionInclude, Matched: false, Reason: "no ruleset configured"}
	}

	if !rs.compiled {
		if err := rs.Compile(); err != nil {
			return MatchResult{Action: ActionExclude, Matched: false, Reason: err.Error()}
		}
	}

	// 1. Evaluate Path Rules
	for _, rule := range rs.PathRules {
		if action, matched, reason := rule.Eval(entry); matched && action != ActionSkip {
			return MatchResult{
				Action:   action,
				Matched:  true,
				RuleName: rule.Name,
				Reason:   reason,
			}
		}
	}

	// 2. Evaluate Git Rules
	for _, rule := range rs.GitRules {
		if action, matched, reason := rule.Eval(entry); matched && action != ActionSkip {
			return MatchResult{
				Action:   action,
				Matched:  true,
				RuleName: rule.Name,
				Reason:   reason,
			}
		}
	}

	// 3. Evaluate Size Rules
	for _, rule := range rs.SizeRules {
		if action, matched, reason := rule.Eval(entry); matched && action != ActionSkip {
			return MatchResult{
				Action:   action,
				Matched:  true,
				RuleName: rule.Name,
				Reason:   reason,
			}
		}
	}

	// 4. Evaluate Type Rules
	for _, rule := range rs.TypeRules {
		if action, matched, reason := rule.Eval(entry); matched && action != ActionSkip {
			return MatchResult{
				Action:   action,
				Matched:  true,
				RuleName: rule.Name,
				Reason:   reason,
			}
		}
	}

	// 5. Fallback to default action
	defaultAct := rs.DefaultAction
	if defaultAct == ActionSkip {
		defaultAct = ActionInclude
	}

	return MatchResult{
		Action:   defaultAct,
		Matched:  false,
		RuleName: "default",
		Reason:   fmt.Sprintf("default rule action (%s)", defaultAct),
	}
}

// ShouldInclude returns true if the entry is permitted to sync.
func (rs *RuleSet) ShouldInclude(entry FileEntry) bool {
	res := rs.Evaluate(entry)
	return res.Action == ActionInclude
}

// ShouldExclude returns true if the entry is excluded from sync.
func (rs *RuleSet) ShouldExclude(entry FileEntry) bool {
	res := rs.Evaluate(entry)
	return res.Action == ActionExclude
}

// ShouldPruneDir checks if a directory path should be skipped entirely during scanning.
func (rs *RuleSet) ShouldPruneDir(path string) bool {
	entry := FileEntry{
		Path:  strings.TrimSuffix(path, "/"),
		IsDir: true,
	}
	res := rs.Evaluate(entry)
	return res.Action == ActionExclude
}

// Empty returns true if no rules are configured in this RuleSet.
func (rs *RuleSet) Empty() bool {
	if rs == nil {
		return true
	}
	return len(rs.GitRules) == 0 && len(rs.PathRules) == 0 && len(rs.SizeRules) == 0 && len(rs.TypeRules) == 0
}
