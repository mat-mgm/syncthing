// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

// Package yamlcfg provides declarative YAML configuration support for Syncthing.
package yamlcfg

import (
	"bytes"
	"fmt"
	"io"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

// Read reads YAML configuration from an io.Reader, applies default values,
// validates structure, and returns the Configuration and original version.
func Read(r io.Reader, myID protocol.DeviceID) (config.Configuration, int, error) {
	return config.ReadYAML(r, myID)
}

// Write serializes a Configuration struct to YAML and writes it to an io.Writer.
func Write(w io.Writer, cfg config.Configuration) error {
	return cfg.WriteYAML(w)
}

// FromYAML parses YAML bytes into a Configuration object.
func FromYAML(data []byte, myID protocol.DeviceID) (config.Configuration, int, error) {
	return Read(bytes.NewReader(data), myID)
}

// ToYAML serializes a Configuration object to YAML formatted bytes.
func ToYAML(cfg config.Configuration) ([]byte, error) {
	var buf bytes.Buffer
	if err := cfg.WriteYAML(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize configuration to YAML: %w", err)
	}
	return buf.Bytes(), nil
}

// Validate parses and validates YAML data without mutating any files.
func Validate(data []byte, myID protocol.DeviceID) error {
	_, _, err := FromYAML(data, myID)
	return err
}

// XMLToYAML converts existing XML configuration bytes to declarative YAML format.
func XMLToYAML(xmlData []byte, myID protocol.DeviceID) ([]byte, error) {
	cfg, _, err := config.ReadXML(bytes.NewReader(xmlData), myID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML config: %w", err)
	}
	return ToYAML(cfg)
}

// YAMLToXML converts declarative YAML configuration bytes to standard XML format.
func YAMLToXML(yamlData []byte, myID protocol.DeviceID) ([]byte, error) {
	cfg, _, err := FromYAML(yamlData, myID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}
	var buf bytes.Buffer
	if err := cfg.WriteXML(&buf); err != nil {
		return nil, fmt.Errorf("failed to serialize config to XML: %w", err)
	}
	return buf.Bytes(), nil
}
