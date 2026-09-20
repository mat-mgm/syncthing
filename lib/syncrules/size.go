// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package syncrules

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// SizeRule evaluates files based on byte size constraints.
type SizeRule struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	MinSize string `json:"minSize,omitempty" yaml:"minSize,omitempty"` // e.g. "10KB", "1MB"
	MaxSize string `json:"maxSize,omitempty" yaml:"maxSize,omitempty"` // e.g. "500MB", "2GB"
	Action  Action `json:"action" yaml:"action"`                       // include or exclude

	minBytes int64
	maxBytes int64
	parsed   bool
}

// Compile parses human size strings into bytes.
func (r *SizeRule) Compile() error {
	r.minBytes = 0
	r.maxBytes = math.MaxInt64

	if r.MinSize != "" {
		b, err := ParseSize(r.MinSize)
		if err != nil {
			return fmt.Errorf("size rule %q: invalid minSize: %w", r.Name, err)
		}
		r.minBytes = b
	}

	if r.MaxSize != "" {
		b, err := ParseSize(r.MaxSize)
		if err != nil {
			return fmt.Errorf("size rule %q: invalid maxSize: %w", r.Name, err)
		}
		r.maxBytes = b
	}

	if r.minBytes > r.maxBytes {
		return fmt.Errorf("size rule %q: minSize (%d bytes) cannot be greater than maxSize (%d bytes)", r.Name, r.minBytes, r.maxBytes)
	}

	r.parsed = true
	return nil
}

// Eval evaluates a file entry against size boundaries. Directories are not size-filtered.
func (r *SizeRule) Eval(entry FileEntry) (Action, bool, string) {
	if entry.IsDir {
		return ActionSkip, false, ""
	}

	if !r.parsed {
		if err := r.Compile(); err != nil {
			return ActionSkip, false, err.Error()
		}
	}

	if entry.Size >= r.minBytes && entry.Size <= r.maxBytes {
		reason := fmt.Sprintf("file size %d bytes matches [%d, %d]", entry.Size, r.minBytes, r.maxBytes)
		return r.Action, true, reason
	}

	return ActionSkip, false, ""
}

// ParseSize parses human-readable size strings (e.g. "500B", "10KB", "2.5MB", "1GiB") into bytes.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	// Separate numeric and unit parts
	i := 0
	for i < len(s) && (unicode.IsDigit(rune(s[i])) || s[i] == '.') {
		i++
	}

	numStr := strings.TrimSpace(s[:i])
	unitStr := strings.ToUpper(strings.TrimSpace(s[i:]))

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil || val < 0 {
		return 0, fmt.Errorf("invalid numeric size value %q", numStr)
	}

	var multiplier float64 = 1
	switch unitStr {
	case "", "B", "BYTES":
		multiplier = 1
	case "K", "KB", "KILOBYTES":
		multiplier = 1000
	case "KIB", "KIBIBYTES":
		multiplier = 1024
	case "M", "MB", "MEGABYTES":
		multiplier = 1000 * 1000
	case "MIB", "MEBIBYTES":
		multiplier = 1024 * 1024
	case "G", "GB", "GIGABYTES":
		multiplier = 1000 * 1000 * 1000
	case "GIB", "GIBIBYTES":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB", "TERABYTES":
		multiplier = 1000 * 1000 * 1000 * 1000
	case "TIB", "TEBIBYTES":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown size unit %q in %q", unitStr, s)
	}

	result := int64(val * multiplier)
	return result, nil
}
