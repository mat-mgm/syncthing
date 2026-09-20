// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syncrules

import (
	"path/filepath"
	"strings"
	"time"
)

// FileEntry represents a filesystem item to be evaluated by the sync rules engine.
type FileEntry struct {
	Path      string    // Slash-separated relative path from folder root
	Size      int64     // Size in bytes
	IsDir     bool      // True if entry is a directory
	IsSymlink bool      // True if entry is a symbolic link
	ModTime   time.Time // Modification time
}

// NormalizedPath returns the path clean and slash-standardized.
func (e FileEntry) NormalizedPath() string {
	clean := filepath.ToSlash(filepath.Clean(e.Path))
	return strings.TrimPrefix(clean, "/")
}

// Name returns the base filename.
func (e FileEntry) Name() string {
	return filepath.Base(e.NormalizedPath())
}

// Ext returns the lowercase file extension including dot (e.g. ".pdf").
func (e FileEntry) Ext() string {
	return strings.ToLower(filepath.Ext(e.NormalizedPath()))
}
