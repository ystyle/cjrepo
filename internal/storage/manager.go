package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// Manager handles file storage operations
type Manager struct {
	rootPath string
}

// NewStorageManager creates a new storage manager
func NewStorageManager(rootPath string) *Manager {
	// Convert to absolute path
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		absPath = rootPath
	}
	return &Manager{
		rootPath: absPath,
	}
}

// EnsurePath ensures the directory exists for the given org and package
func (m *Manager) EnsurePath(org, name string) error {
	dir := filepath.Join(m.rootPath, org, name)
	return os.MkdirAll(dir, 0755)
}

// GetTarballPath returns the storage path for a tarball file
func (m *Manager) GetTarballPath(org, name, version string) string {
	return filepath.Join(m.rootPath, org, name, version+".cjp")
}

// SaveTarball saves a tarball file to storage
func (m *Manager) SaveTarball(org, name, version string, data []byte) error {
	// Ensure directory exists
	if err := m.EnsurePath(org, name); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save file
	path := m.GetTarballPath(org, name, version)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// DeleteTarball deletes a tarball file from storage
func (m *Manager) DeleteTarball(org, name, version string) error {
	path := m.GetTarballPath(org, name, version)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// FileExists checks if a tarball file exists
func (m *Manager) FileExists(org, name, version string) bool {
	path := m.GetTarballPath(org, name, version)
	_, err := os.Stat(path)
	return err == nil
}

// GetFile returns the file handle for a tarball
func (m *Manager) GetFile(org, name, version string) (*os.File, error) {
	path := m.GetTarballPath(org, name, version)
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	return file, nil
}
