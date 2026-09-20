// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at https://mozilla.org/MPL/2.0/.

package yamlcfg_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/config/yamlcfg"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
)

var testDeviceID = protocol.DeviceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

func TestYAMLRoundTrip(t *testing.T) {
	cfg := config.New(testDeviceID)
	cfg.GUI.RawAddress = "127.0.0.1:8384"
	cfg.GUI.User = "admin"

	yamlBytes, err := yamlcfg.ToYAML(cfg)
	if err != nil {
		t.Fatalf("Failed to serialize to YAML: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Fatal("Expected non-empty YAML output")
	}

	loadedCfg, version, err := yamlcfg.FromYAML(yamlBytes, testDeviceID)
	if err != nil {
		t.Fatalf("Failed to deserialize from YAML: %v", err)
	}

	if version != config.CurrentVersion {
		t.Errorf("Expected version %d, got %d", config.CurrentVersion, version)
	}

	if loadedCfg.GUI.Address() != "127.0.0.1:8384" {
		t.Errorf("Expected GUI address 127.0.0.1:8384, got %s", loadedCfg.GUI.Address())
	}

	if loadedCfg.GUI.User != "admin" {
		t.Errorf("Expected GUI user admin, got %s", loadedCfg.GUI.User)
	}
}

func TestBidirectionalXMLYAML(t *testing.T) {
	cfg := config.New(testDeviceID)
	cfg.Options.RawListenAddresses = []string{"tcp://0.0.0.0:22000"}

	var xmlBuf bytes.Buffer
	if err := cfg.WriteXML(&xmlBuf); err != nil {
		t.Fatalf("Failed to write XML: %v", err)
	}

	yamlBytes, err := yamlcfg.XMLToYAML(xmlBuf.Bytes(), testDeviceID)
	if err != nil {
		t.Fatalf("XMLToYAML failed: %v", err)
	}

	convertedXML, err := yamlcfg.YAMLToXML(yamlBytes, testDeviceID)
	if err != nil {
		t.Fatalf("YAMLToXML failed: %v", err)
	}

	finalCfg, _, err := config.ReadXML(bytes.NewReader(convertedXML), testDeviceID)
	if err != nil {
		t.Fatalf("Failed to read converted XML: %v", err)
	}

	if len(finalCfg.Options.RawListenAddresses) == 0 || finalCfg.Options.RawListenAddresses[0] != "tcp://0.0.0.0:22000" {
		t.Errorf("Expected listen address tcp://0.0.0.0:22000, got %v", finalCfg.Options.RawListenAddresses)
	}
}

func TestYAMLDefaultsInheritance(t *testing.T) {
	yamlInput := fmt.Sprintf(`
version: 52
folders:
  - id: default
    label: Default Folder
    path: /tmp/sync
devices:
  - deviceID: %s
    name: TestDevice
`, testDeviceID.String())

	cfg, _, err := yamlcfg.FromYAML([]byte(yamlInput), testDeviceID)
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if len(cfg.Folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(cfg.Folders))
	}

	folder := cfg.Folders[0]
	if folder.ID != "default" || folder.Path != "/tmp/sync" {
		t.Errorf("Unexpected folder config: %+v", folder)
	}

	// Verify defaults were populated on the folder
	if folder.RescanIntervalS <= 0 {
		t.Errorf("Expected default RescanIntervalS > 0, got %d", folder.RescanIntervalS)
	}
}

func TestInvalidYAMLSyntax(t *testing.T) {
	invalidYAML := `
version: [invalid yaml structure: {
`
	err := yamlcfg.Validate([]byte(invalidYAML), testDeviceID)
	if err == nil {
		t.Fatal("Expected error for invalid YAML syntax, got nil")
	}
}

func TestWrapperLoadSaveYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	initialCfg := config.New(testDeviceID)
	initialCfg.GUI.User = "initialUser"
	yamlData, err := yamlcfg.ToYAML(initialCfg)
	if err != nil {
		t.Fatalf("Failed to generate initial YAML: %v", err)
	}

	if err := os.WriteFile(cfgPath, yamlData, 0o600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	wrapper, _, err := config.Load(cfgPath, testDeviceID, events.NoopLogger)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	if wrapper.ConfigPath() != cfgPath {
		t.Errorf("Expected path %s, got %s", cfgPath, wrapper.ConfigPath())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go wrapper.Serve(ctx)

	// Modify config and save
	waiter, err := wrapper.Modify(func(c *config.Configuration) {
		c.GUI.User = "updatedUser"
	})
	if err != nil {
		t.Fatalf("wrapper.Modify failed: %v", err)
	}
	waiter.Wait()

	if err := wrapper.Save(); err != nil {
		t.Fatalf("wrapper.Save failed: %v", err)
	}

	// Re-load and verify
	reloadedWrapper, _, err := config.Load(cfgPath, testDeviceID, events.NoopLogger)
	if err != nil {
		t.Fatalf("Re-loading config failed: %v", err)
	}

	reloadedCfg := reloadedWrapper.RawCopy()
	if reloadedCfg.GUI.User != "updatedUser" {
		t.Errorf("Expected GUI user updatedUser, got %s", reloadedCfg.GUI.User)
	}
}

func TestAutoDetectYAMLWhenXMLMissing(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "config.yaml")
	xmlPath := filepath.Join(tmpDir, "config.xml")

	initialCfg := config.New(testDeviceID)
	initialCfg.GUI.User = "autoDetectUser"
	yamlData, err := yamlcfg.ToYAML(initialCfg)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}
	if err := os.WriteFile(yamlPath, yamlData, 0o600); err != nil {
		t.Fatalf("Failed to write yaml file: %v", err)
	}

	// Request xmlPath, which doesn't exist, but config.yaml exists
	wrapper, _, err := config.Load(xmlPath, testDeviceID, events.NoopLogger)
	if err != nil {
		t.Fatalf("config.Load with fallback failed: %v", err)
	}

	if wrapper.ConfigPath() != yamlPath {
		t.Errorf("Expected wrapper path to be %s, got %s", yamlPath, wrapper.ConfigPath())
	}

	if wrapper.RawCopy().GUI.User != "autoDetectUser" {
		t.Errorf("Expected GUI user autoDetectUser, got %s", wrapper.RawCopy().GUI.User)
	}
}
