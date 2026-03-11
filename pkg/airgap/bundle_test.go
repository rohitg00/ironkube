package airgap

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBinaryFetcher struct {
	data map[string][]byte
	err  error
}

func (m *mockBinaryFetcher) Fetch(_ context.Context, name, version, arch string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := fmt.Sprintf("%s-%s-%s", name, version, arch)
	if d, ok := m.data[key]; ok {
		return d, nil
	}
	return []byte(fmt.Sprintf("binary-%s", name)), nil
}

type mockImageExporter struct {
	exported []string
	err      error
}

func (m *mockImageExporter) Export(_ context.Context, images []string, outputPath string) error {
	if m.err != nil {
		return m.err
	}
	m.exported = images
	for _, img := range images {
		f, err := os.Create(filepath.Join(outputPath, filepath.Base(img)+".tar"))
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

type mockChartFetcher struct {
	data map[string][]byte
	err  error
}

func (m *mockChartFetcher) Fetch(_ context.Context, repo, chart, version string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := fmt.Sprintf("%s/%s-%s", repo, chart, version)
	if d, ok := m.data[key]; ok {
		return d, nil
	}
	return []byte(fmt.Sprintf("chart-%s", chart)), nil
}

func testSpec() BundleSpec {
	return BundleSpec{
		Name:    "test-bundle",
		Version: "1.0.0",
		Distro:  "k3s",
		Arch:    "amd64",
		Binaries: []BinaryRef{
			{Name: "k3s", Version: "v1.28.0"},
		},
		Images: []string{"nginx:latest", "coredns:1.11"},
		Charts: []ChartRef{
			{Repository: "https://charts.example.com", Chart: "ingress", Version: "4.0.0"},
		},
	}
}

func TestCreateProducesManifest(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)
	assert.Equal(t, "test-bundle", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "k3s", manifest.Distro)
	assert.Equal(t, "amd64", manifest.Arch)
}

func TestCreateFetchesBinaries(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)
	assert.Contains(t, manifest.Binaries, "k3s")

	binPath := filepath.Join(dir, "binaries", "k3s")
	_, err = os.Stat(binPath)
	assert.NoError(t, err)
}

func TestCreateExportsImages(t *testing.T) {
	dir := t.TempDir()
	imgExporter := &mockImageExporter{}
	creator := NewCreator(&mockBinaryFetcher{}, imgExporter, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"nginx:latest", "coredns:1.11"}, manifest.Images)
	assert.Equal(t, []string{"nginx:latest", "coredns:1.11"}, imgExporter.exported)
}

func TestCreateFetchesCharts(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)
	assert.Contains(t, manifest.Charts, "ingress-4.0.0.tgz")

	chartPath := filepath.Join(dir, "charts", "ingress-4.0.0.tgz")
	_, err = os.Stat(chartPath)
	assert.NoError(t, err)
}

func TestCreateHandlesBinaryFetchError(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(
		&mockBinaryFetcher{err: fmt.Errorf("network error")},
		&mockImageExporter{},
		&mockChartFetcher{},
	)

	_, err := creator.Create(context.Background(), testSpec(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching binary")
}

func TestCreateHandlesImageExportError(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(
		&mockBinaryFetcher{},
		&mockImageExporter{err: fmt.Errorf("docker error")},
		&mockChartFetcher{},
	)

	_, err := creator.Create(context.Background(), testSpec(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exporting images")
}

func TestCreateHandlesChartFetchError(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(
		&mockBinaryFetcher{},
		&mockImageExporter{},
		&mockChartFetcher{err: fmt.Errorf("helm error")},
	)

	_, err := creator.Create(context.Background(), testSpec(), dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching chart")
}

func TestCreateWithNoImages(t *testing.T) {
	dir := t.TempDir()
	spec := testSpec()
	spec.Images = nil
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), spec, dir)
	require.NoError(t, err)
	assert.Empty(t, manifest.Images)
}

func TestCreateWithNoCharts(t *testing.T) {
	dir := t.TempDir()
	spec := testSpec()
	spec.Charts = nil
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	manifest, err := creator.Create(context.Background(), spec, dir)
	require.NoError(t, err)
	assert.Empty(t, manifest.Charts)
}

func TestCreateWithNoBinariesFails(t *testing.T) {
	dir := t.TempDir()
	spec := testSpec()
	spec.Binaries = nil
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	_, err := creator.Create(context.Background(), spec, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one binary is required")
}

func TestCreateArchiveCreatesTarGz(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("hello"), 0o644))

	outputPath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	err := creator.CreateArchive(srcDir, outputPath)
	require.NoError(t, err)

	f, err := os.Open(outputPath)
	require.NoError(t, err)
	defer f.Close()

	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if hdr.Name == "test.txt" {
			found = true
		}
	}
	assert.True(t, found, "test.txt should be in archive")
}

func TestManifestHasCorrectTimestamp(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	before := time.Now()
	manifest, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)
	after := time.Now()

	assert.False(t, manifest.CreatedAt.Before(before))
	assert.False(t, manifest.CreatedAt.After(after))
}

func TestEmptySpecNameReturnsError(t *testing.T) {
	dir := t.TempDir()
	spec := testSpec()
	spec.Name = ""
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	_, err := creator.Create(context.Background(), spec, dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestCreateWritesManifestFile(t *testing.T) {
	dir := t.TempDir()
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	_, err := creator.Create(context.Background(), testSpec(), dir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	require.NoError(t, err)

	var manifest BundleManifest
	require.NoError(t, json.Unmarshal(data, &manifest))
	assert.Equal(t, "test-bundle", manifest.Name)
}

func TestCreateArchiveContainsSubdirectories(t *testing.T) {
	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "binaries")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "k3s"), []byte("binary"), 0o755))

	outputPath := filepath.Join(t.TempDir(), "bundle.tar.gz")
	creator := NewCreator(&mockBinaryFetcher{}, &mockImageExporter{}, &mockChartFetcher{})

	err := creator.CreateArchive(srcDir, outputPath)
	require.NoError(t, err)

	f, err := os.Open(outputPath)
	require.NoError(t, err)
	defer f.Close()

	gr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer gr.Close()

	tr := tar.NewReader(gr)
	names := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names[hdr.Name] = true
	}
	assert.True(t, names["binaries"], "should contain binaries directory")
	assert.True(t, names[filepath.Join("binaries", "k3s")], "should contain binaries/k3s")
}
