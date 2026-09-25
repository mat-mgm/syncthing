// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syncrules_test

import (
	"testing"
	"time"

	"github.com/syncthing/syncthing/lib/syncrules"
)

func TestSizeRule(t *testing.T) {
	rule := syncrules.SizeRule{
		Name:    "ExcludeLargeFiles",
		MinSize: "100MB",
		Action:  syncrules.ActionExclude,
	}

	if err := rule.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	smallFile := syncrules.FileEntry{Path: "data.txt", Size: 50 * 1000 * 1000}
	action, matched, _ := rule.Eval(smallFile)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected small file to skip rule, got action=%v matched=%v", action, matched)
	}

	largeFile := syncrules.FileEntry{Path: "backup.iso", Size: 250 * 1000 * 1000}
	action, matched, _ = rule.Eval(largeFile)
	if !matched || action != syncrules.ActionExclude {
		t.Errorf("Expected large file to be excluded, got action=%v matched=%v", action, matched)
	}

	// Directories must not be excluded by size
	dirEntry := syncrules.FileEntry{Path: "someDir", Size: 500 * 1000 * 1000, IsDir: true}
	action, matched, _ = rule.Eval(dirEntry)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected directory to not match size rule, got action=%v matched=%v", action, matched)
	}
}

func TestSizeParsing(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"0", 0},
		{"500B", 500},
		{"10KB", 10000},
		{"10KiB", 10240},
		{"1MB", 1000000},
		{"1MiB", 1048576},
		{"2.5GB", 2500000000},
		{"2GiB", 2147483648},
	}

	for _, tc := range cases {
		bytes, err := syncrules.ParseSize(tc.input)
		if err != nil {
			t.Errorf("ParseSize(%q) unexpected error: %v", tc.input, err)
		}
		if bytes != tc.expected {
			t.Errorf("ParseSize(%q) expected %d, got %d", tc.input, tc.expected, bytes)
		}
	}

	// Negative tests
	invalidCases := []string{"invalid", "-10MB", "100XYZ"}
	for _, inv := range invalidCases {
		if _, err := syncrules.ParseSize(inv); err == nil {
			t.Errorf("Expected error for invalid size %q, got nil", inv)
		}
	}
}

func TestTypeRule(t *testing.T) {
	rule := syncrules.TypeRule{
		Name:       "MediaOnly",
		Categories: []string{"images", "video"},
		Extensions: []string{".custommedia"},
		Action:     syncrules.ActionInclude,
	}

	if err := rule.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	imgEntry := syncrules.FileEntry{Path: "vacation/photo.JPG"}
	action, matched, _ := rule.Eval(imgEntry)
	if !matched || action != syncrules.ActionInclude {
		t.Errorf("Expected JPG to match image category, got %v, matched=%v", action, matched)
	}

	customEntry := syncrules.FileEntry{Path: "media/sample.custommedia"}
	action, matched, _ = rule.Eval(customEntry)
	if !matched || action != syncrules.ActionInclude {
		t.Errorf("Expected custom extension to match, got %v, matched=%v", action, matched)
	}

	txtEntry := syncrules.FileEntry{Path: "doc.txt"}
	action, matched, _ = rule.Eval(txtEntry)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected txt to skip rule, got %v, matched=%v", action, matched)
	}
}

func TestGitRule(t *testing.T) {
	rule := syncrules.GitRule{
		Name:              "ExcludeGitMetadata",
		Mode:              syncrules.GitExcludeMetadata,
		ExcludeSubmodules: true,
	}

	if err := rule.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	cases := []struct {
		path            string
		expectedExclude bool
	}{
		{".git", true},
		{".git/config", true},
		{".git/objects/0a/12345", true},
		{"projects/myrepo/.git/HEAD", true},
		{"projects/myrepo/src/main.go", false},
		{".gitmodules", true},
		{"nested/.gitmodules", true},
		{"regular_file.txt", false},
	}

	for _, tc := range cases {
		entry := syncrules.FileEntry{Path: tc.path}
		action, matched, _ := rule.Eval(entry)
		isExcluded := matched && action == syncrules.ActionExclude
		if isExcluded != tc.expectedExclude {
			t.Errorf("Path %q expectedExclude=%v, got %v (action=%v, matched=%v)", tc.path, tc.expectedExclude, isExcluded, action, matched)
		}
	}
}

func TestPathRule(t *testing.T) {
	rule := syncrules.PathRule{
		Name:    "ExcludeTempAndLogs",
		Pattern: "**/*.log",
		Action:  syncrules.ActionExclude,
	}

	if err := rule.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	logFile := syncrules.FileEntry{Path: "logs/2026/app.log"}
	action, matched, _ := rule.Eval(logFile)
	if !matched || action != syncrules.ActionExclude {
		t.Errorf("Expected log file to be excluded, got %v, matched=%v", action, matched)
	}

	docFile := syncrules.FileEntry{Path: "logs/2026/readme.txt"}
	action, matched, _ = rule.Eval(docFile)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected txt file to skip rule, got %v, matched=%v", action, matched)
	}
}

func TestRuleSetPrecedence(t *testing.T) {
	rs := syncrules.NewRuleSet()
	rs.DefaultAction = syncrules.ActionInclude

	// 1. Explicitly include important log
	rs.PathRules = append(rs.PathRules, syncrules.PathRule{
		Name:    "IncludeImportantLog",
		Pattern: "logs/important.log",
		Action:  syncrules.ActionInclude,
	})

	// 2. Exclude git internals
	rs.GitRules = append(rs.GitRules, syncrules.GitRule{
		Name: "ExcludeGit",
		Mode: syncrules.GitExcludeMetadata,
	})

	// 3. Exclude files > 10MB
	rs.SizeRules = append(rs.SizeRules, syncrules.SizeRule{
		Name:    "ExcludeLarge",
		MinSize: "10MB",
		Action:  syncrules.ActionExclude,
	})

	// 4. Exclude all logs by default
	rs.PathRules = append(rs.PathRules, syncrules.PathRule{
		Name:    "ExcludeLogs",
		Pattern: "*.log",
		Action:  syncrules.ActionExclude,
	})

	if err := rs.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// Test 1: logs/important.log should be INCLUDED despite *.log rule
	res := rs.Evaluate(syncrules.FileEntry{Path: "logs/important.log", Size: 100})
	if res.Action != syncrules.ActionInclude {
		t.Errorf("Expected logs/important.log to be INCLUDED, got %v (%s)", res.Action, res.Reason)
	}

	// Test 2: other.log should be EXCLUDED
	res = rs.Evaluate(syncrules.FileEntry{Path: "other.log", Size: 100})
	if res.Action != syncrules.ActionExclude {
		t.Errorf("Expected other.log to be EXCLUDED, got %v (%s)", res.Action, res.Reason)
	}

	// Test 3: .git internal should be EXCLUDED
	res = rs.Evaluate(syncrules.FileEntry{Path: ".git/index", Size: 100})
	if res.Action != syncrules.ActionExclude {
		t.Errorf("Expected .git/index to be EXCLUDED, got %v (%s)", res.Action, res.Reason)
	}

	// Test 4: 50MB file should be EXCLUDED by size rule
	res = rs.Evaluate(syncrules.FileEntry{Path: "video.mp4", Size: 50 * 1000 * 1000})
	if res.Action != syncrules.ActionExclude {
		t.Errorf("Expected 50MB file to be EXCLUDED, got %v (%s)", res.Action, res.Reason)
	}

	// Test 5: Standard small file should be INCLUDED by default action
	res = rs.Evaluate(syncrules.FileEntry{Path: "doc.txt", Size: 500})
	if res.Action != syncrules.ActionInclude {
		t.Errorf("Expected small doc.txt to be INCLUDED, got %v (%s)", res.Action, res.Reason)
	}
}

func TestActionMarshaling(t *testing.T) {
	cases := []struct {
		text     string
		expected syncrules.Action
	}{
		{"include", syncrules.ActionInclude},
		{"allow", syncrules.ActionInclude},
		{"sync", syncrules.ActionInclude},
		{"exclude", syncrules.ActionExclude},
		{"ignore", syncrules.ActionExclude},
		{"drop", syncrules.ActionExclude},
		{"skip-file", syncrules.ActionExclude},
		{"skip", syncrules.ActionSkip},
		{"none", syncrules.ActionSkip},
		{"pass", syncrules.ActionSkip},
		{"", syncrules.ActionSkip},
	}

	for _, tc := range cases {
		var a syncrules.Action
		if err := a.UnmarshalText([]byte(tc.text)); err != nil {
			t.Errorf("UnmarshalText(%q) unexpected error: %v", tc.text, err)
		}
		if a != tc.expected {
			t.Errorf("UnmarshalText(%q) expected %v, got %v", tc.text, tc.expected, a)
		}
	}

	var badAction syncrules.Action
	if err := badAction.UnmarshalText([]byte("invalid-action-123")); err == nil {
		t.Error("Expected error for invalid action text, got nil")
	}

	bs, err := syncrules.ActionInclude.MarshalText()
	if err != nil || string(bs) != "include" {
		t.Errorf("MarshalText(ActionInclude) = %s, err = %v", string(bs), err)
	}
}

func TestFileEntryNormalization(t *testing.T) {
	e := syncrules.FileEntry{
		Path: "/folder/subfolder/file.PDF",
	}

	norm := e.NormalizedPath()
	if norm != "folder/subfolder/file.PDF" {
		t.Errorf("NormalizedPath() expected folder/subfolder/file.PDF, got %q", norm)
	}

	if e.Name() != "file.PDF" {
		t.Errorf("Name() expected file.PDF, got %q", e.Name())
	}

	if e.Ext() != ".pdf" {
		t.Errorf("Ext() expected lowercase .pdf, got %q", e.Ext())
	}
}

func TestPathRuleRegexAndCaseSensitivity(t *testing.T) {
	// 1. Regex test
	regexRule := syncrules.PathRule{
		Name:          "LogArchiveRegex",
		Pattern:       `^logs/\d{4}/archive\.tar\.gz$`,
		IsRegex:       true,
		CaseSensitive: true,
		Action:        syncrules.ActionExclude,
	}
	if err := regexRule.Compile(); err != nil {
		t.Fatalf("Compile regex rule failed: %v", err)
	}

	matchEntry := syncrules.FileEntry{Path: "logs/2026/archive.tar.gz"}
	action, matched, _ := regexRule.Eval(matchEntry)
	if !matched || action != syncrules.ActionExclude {
		t.Errorf("Expected regex to match %q, got matched=%v action=%v", matchEntry.Path, matched, action)
	}

	nomatchEntry := syncrules.FileEntry{Path: "logs/abc/archive.tar.gz"}
	action, matched, _ = regexRule.Eval(nomatchEntry)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected regex to not match %q, got matched=%v action=%v", nomatchEntry.Path, matched, action)
	}

	// 2. Case sensitive glob test
	caseRule := syncrules.PathRule{
		Name:          "CaseSensitiveGlob",
		Pattern:       "docs/*.TXT",
		CaseSensitive: true,
		Action:        syncrules.ActionInclude,
	}
	if err := caseRule.Compile(); err != nil {
		t.Fatalf("Compile case rule failed: %v", err)
	}

	upperEntry := syncrules.FileEntry{Path: "docs/readme.TXT"}
	action, matched, _ = caseRule.Eval(upperEntry)
	if !matched || action != syncrules.ActionInclude {
		t.Errorf("Expected case-sensitive match for .TXT, got matched=%v action=%v", matched, action)
	}

	lowerEntry := syncrules.FileEntry{Path: "docs/readme.txt"}
	action, matched, _ = caseRule.Eval(lowerEntry)
	if matched || action != syncrules.ActionSkip {
		t.Errorf("Expected case-sensitive mismatch for .txt, got matched=%v action=%v", matched, action)
	}
}

func TestRuleSetHelperMethodsAndNilSafety(t *testing.T) {
	// 1. nil safety
	var nilRS *syncrules.RuleSet
	if !nilRS.Empty() {
		t.Error("Expected nil RuleSet to report Empty() == true")
	}
	if !nilRS.ShouldInclude(syncrules.FileEntry{Path: "test.txt"}) {
		t.Error("Expected nil RuleSet to ShouldInclude by default")
	}
	if nilRS.ShouldExclude(syncrules.FileEntry{Path: "test.txt"}) {
		t.Error("Expected nil RuleSet to not ShouldExclude")
	}
	if nilRS.ShouldPruneDir("any/dir") {
		t.Error("Expected nil RuleSet to not ShouldPruneDir")
	}

	// 2. Helper methods on populated RuleSet
	rs := syncrules.NewRuleSet()
	if !rs.Empty() {
		t.Error("Expected NewRuleSet() with no rules to report Empty() == true")
	}

	rs.PathRules = append(rs.PathRules, syncrules.PathRule{
		Name:    "ExcludeCacheDir",
		Pattern: "cache/**",
		Action:  syncrules.ActionExclude,
	})
	if rs.Empty() {
		t.Error("Expected populated RuleSet to not be empty")
	}

	if err := rs.Compile(); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if rs.ShouldInclude(syncrules.FileEntry{Path: "cache/item.bin"}) {
		t.Error("Expected cache/item.bin to not be included")
	}
	if !rs.ShouldExclude(syncrules.FileEntry{Path: "cache/item.bin"}) {
		t.Error("Expected cache/item.bin to be excluded")
	}
	if !rs.ShouldPruneDir("cache") {
		t.Error("Expected ShouldPruneDir('cache') to be true")
	}
	if rs.ShouldPruneDir("src") {
		t.Error("Expected ShouldPruneDir('src') to be false")
	}
}

func BenchmarkRuleEvaluation(b *testing.B) {
	rs := syncrules.NewRuleSet()
	rs.PathRules = []syncrules.PathRule{
		{Name: "ExcludeTmp", Pattern: "tmp/**", Action: syncrules.ActionExclude},
		{Name: "ExcludeBuild", Pattern: "build/**", Action: syncrules.ActionExclude},
	}
	rs.GitRules = []syncrules.GitRule{
		{Name: "Git", Mode: syncrules.GitExcludeMetadata},
	}
	rs.SizeRules = []syncrules.SizeRule{
		{Name: "Size", MinSize: "100MB", Action: syncrules.ActionExclude},
	}
	rs.TypeRules = []syncrules.TypeRule{
		{Name: "Images", Categories: []string{"images"}, Action: syncrules.ActionInclude},
	}
	if err := rs.Compile(); err != nil {
		b.Fatalf("Compile failed: %v", err)
	}

	entries := []syncrules.FileEntry{
		{Path: "src/main.go", Size: 2048, ModTime: time.Now()},
		{Path: "tmp/cache.bin", Size: 10000, ModTime: time.Now()},
		{Path: "assets/logo.png", Size: 4096, ModTime: time.Now()},
		{Path: "repo/.git/objects/aa/bb", Size: 100, ModTime: time.Now()},
		{Path: "bigdata.iso", Size: 500 * 1000 * 1000, ModTime: time.Now()},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry := entries[i%len(entries)]
		_ = rs.Evaluate(entry)
	}
}
