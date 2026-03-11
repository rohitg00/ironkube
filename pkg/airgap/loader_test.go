package airgap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rohitg00/ironkube/pkg/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBinaryInstaller struct {
	installed map[string][]byte
	err       error
}

func newMockBinaryInstaller() *mockBinaryInstaller {
	return &mockBinaryInstaller{installed: make(map[string][]byte)}
}

func (m *mockBinaryInstaller) Install(_ context.Context, _ provider.Machine, name string, data []byte, _ string) error {
	if m.err != nil {
		return m.err
	}
	m.installed[name] = data
	return nil
}

type mockImageLoader struct {
	loaded []string
	err    error
}

func (m *mockImageLoader) Load(_ context.Context, _ provider.Machine, archivePath string) error {
	if m.err != nil {
		return m.err
	}
	m.loaded = append(m.loaded, archivePath)
	return nil
}

func testMachine() provider.Machine {
	return provider.Machine{
		ID: "test-machine",
		Spec: provider.MachineSpec{
			Host: "10.0.0.1",
			User: "root",
		},
		Status: provider.MachineStatusReady,
	}
}

func writeTestManifest(t *testing.T, dir string, manifest *BundleManifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), data, 0o644))
}

func setupBundleDir(t *testing.T) (string, *BundleManifest) {
	t.Helper()
	dir := t.TempDir()
	manifest := &BundleManifest{
		Name:      "test-bundle",
		Version:   "1.0.0",
		Distro:    "k3s",
		Arch:      "amd64",
		CreatedAt: time.Now(),
		Binaries:  []string{"k3s"},
		Images:    []string{"nginx.tar"},
		Charts:    []string{"ingress-4.0.0.tgz"},
	}

	writeTestManifest(t, dir, manifest)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "binaries"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "binaries", "k3s"), []byte("k3s-binary"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "images"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "images", "nginx.tar"), []byte("image-data"), 0o644))

	return dir, manifest
}

func TestLoadManifestReadsValid(t *testing.T) {
	dir := t.TempDir()
	expected := &BundleManifest{
		Name:    "my-bundle",
		Version: "2.0.0",
		Distro:  "kubeadm",
		Arch:    "arm64",
	}
	writeTestManifest(t, dir, expected)

	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	manifest, err := loader.LoadManifest(dir)
	require.NoError(t, err)
	assert.Equal(t, "my-bundle", manifest.Name)
	assert.Equal(t, "2.0.0", manifest.Version)
	assert.Equal(t, "kubeadm", manifest.Distro)
	assert.Equal(t, "arm64", manifest.Arch)
}

func TestLoadManifestHandlesMissingFile(t *testing.T) {
	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	_, err := loader.LoadManifest(t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading manifest")
}

func TestLoadManifestHandlesInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{invalid"), 0o644))

	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	_, err := loader.LoadManifest(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing manifest")
}

func TestLoadBundleInstallsBinaries(t *testing.T) {
	dir, _ := setupBundleDir(t)
	installer := newMockBinaryInstaller()
	loader := NewLoader(installer, &mockImageLoader{})

	err := loader.LoadBundle(context.Background(), testMachine(), dir)
	require.NoError(t, err)
	assert.Contains(t, installer.installed, "k3s")
	assert.Equal(t, []byte("k3s-binary"), installer.installed["k3s"])
}

func TestLoadBundleLoadsImages(t *testing.T) {
	dir, _ := setupBundleDir(t)
	imgLoader := &mockImageLoader{}
	loader := NewLoader(newMockBinaryInstaller(), imgLoader)

	err := loader.LoadBundle(context.Background(), testMachine(), dir)
	require.NoError(t, err)
	require.Len(t, imgLoader.loaded, 1)
	assert.Contains(t, imgLoader.loaded[0], "nginx.tar")
}

func TestLoadBundleHandlesBinaryInstallError(t *testing.T) {
	dir, _ := setupBundleDir(t)
	installer := &mockBinaryInstaller{
		installed: make(map[string][]byte),
		err:       fmt.Errorf("permission denied"),
	}
	loader := NewLoader(installer, &mockImageLoader{})

	err := loader.LoadBundle(context.Background(), testMachine(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "installing binary")
}

func TestLoadBundleHandlesImageLoadError(t *testing.T) {
	dir, _ := setupBundleDir(t)
	imgLoader := &mockImageLoader{err: fmt.Errorf("ctr error")}
	loader := NewLoader(newMockBinaryInstaller(), imgLoader)

	err := loader.LoadBundle(context.Background(), testMachine(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading image")
}

func TestExtractArchiveExtractsTarGz(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "hello.txt"), []byte("world"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "test.tar.gz")
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})
	require.NoError(t, creator.CreateArchive(srcDir, archivePath))

	extractDir := t.TempDir()
	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	err := loader.ExtractArchive(archivePath, extractDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(extractDir, "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, "world", string(data))
}

func TestExtractArchiveHandlesInvalidArchive(t *testing.T) {
	invalidPath := filepath.Join(t.TempDir(), "bad.tar.gz")
	require.NoError(t, os.WriteFile(invalidPath, []byte("not a tar"), 0o644))

	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	err := loader.ExtractArchive(invalidPath, t.TempDir())
	assert.Error(t, err)
}

func TestLoadBundleEmptyBundleDir(t *testing.T) {
	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	err := loader.LoadBundle(context.Background(), testMachine(), t.TempDir())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "loading manifest")
}

func TestExtractArchivePreservesDirectoryStructure(t *testing.T) {
	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "configs")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "cluster.yaml"), []byte("apiVersion: v1"), 0o644))

	archivePath := filepath.Join(t.TempDir(), "nested.tar.gz")
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})
	require.NoError(t, creator.CreateArchive(srcDir, archivePath))

	extractDir := t.TempDir()
	loader := NewLoader(newMockBinaryInstaller(), &mockImageLoader{})
	require.NoError(t, loader.ExtractArchive(archivePath, extractDir))

	data, err := os.ReadFile(filepath.Join(extractDir, "configs", "cluster.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "apiVersion: v1", string(data))
}
