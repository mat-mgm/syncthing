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
