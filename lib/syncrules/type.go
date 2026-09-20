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

// Known category mappings for file extension groups
var categoryExtensions = map[string][]string{
	"documents":   {".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp", ".txt", ".md", ".rtf", ".epub"},
	"images":      {".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".tiff", ".raw", ".ico", ".heic", ".avif"},
	"audio":       {".mp3", ".flac", ".wav", ".aac", ".ogg", ".m4a", ".wma", ".opus", ".alac"},
	"video":       {".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v", ".ts"},
	"archives":    {".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".zst", ".iso", ".tgz"},
	"code":        {".go", ".py", ".js", ".ts", ".rs", ".c", ".cpp", ".h", ".hpp", ".java", ".html", ".css", ".json", ".yaml", ".yml", ".xml", ".sh", ".bash", ".sql", ".nix"},
	"executables": {".exe", ".dll", ".so", ".dylib", ".bin", ".elf", ".msi", ".appimage"},
}

// TypeRule matches files based on extensions and categories.
type TypeRule struct {
	Name       string   `json:"name,omitempty" yaml:"name,omitempty"`
	Extensions []string `json:"extensions,omitempty" yaml:"extensions,omitempty"` // e.g. [".jpg", ".png"]
	Categories []string `json:"categories,omitempty" yaml:"categories,omitempty"` // e.g. ["images", "documents"]
	Action     Action   `json:"action" yaml:"action"`                             // include or exclude

	extMap map[string]struct{}
	parsed bool
}

// Compile processes categories and normalizes extensions.
func (r *TypeRule) Compile() error {
	r.extMap = make(map[string]struct{})

	for _, ext := range r.Extensions {
		normalized := strings.ToLower(strings.TrimSpace(ext))
		if normalized != "" {
			if !strings.HasPrefix(normalized, ".") {
				normalized = "." + normalized
			}
			r.extMap[normalized] = struct{}{}
		}
	}

	for _, cat := range r.Categories {
		catLower := strings.ToLower(strings.TrimSpace(cat))
		exts, ok := categoryExtensions[catLower]
		if !ok {
			return fmt.Errorf("type rule %q: unknown category %q (supported: documents, images, audio, video, archives, code, executables)", r.Name, cat)
		}
		for _, ext := range exts {
			r.extMap[ext] = struct{}{}
		}
	}

	r.parsed = true
	return nil
}

// Eval evaluates a file entry against extension and category sets.
func (r *TypeRule) Eval(entry FileEntry) (Action, bool, string) {
	if entry.IsDir {
		return ActionSkip, false, ""
	}

	if !r.parsed {
		if err := r.Compile(); err != nil {
			return ActionSkip, false, err.Error()
		}
	}

	ext := entry.Ext()
	if _, ok := r.extMap[ext]; ok {
		reason := fmt.Sprintf("file extension %q matched type rule", ext)
		return r.Action, true, reason
	}

	return ActionSkip, false, ""
}
