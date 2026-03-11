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
	"time"
)

type BinaryFetcher interface {
	Fetch(ctx context.Context, name, version, arch string) ([]byte, error)
}

type ImageExporter interface {
	Export(ctx context.Context, images []string, outputPath string) error
}

type ChartFetcher interface {
	Fetch(ctx context.Context, repo, chart, version string) ([]byte, error)
}

type BundleSpec struct {
	Name     string
	Version  string
	Distro   string
	Arch     string
	Images   []string
	Charts   []ChartRef
	Binaries []BinaryRef
}

type ChartRef struct {
	Repository string
	Chart      string
	Version    string
}

type BinaryRef struct {
	Name    string
	Version string
}

type BundleManifest struct {
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Distro    string    `json:"distro"`
	Arch      string    `json:"arch"`
	CreatedAt time.Time `json:"createdAt"`
	Binaries  []string  `json:"binaries"`
	Images    []string  `json:"images"`
	Charts    []string  `json:"charts"`
}

type Creator struct {
	binFetcher   BinaryFetcher
	imgExporter  ImageExporter
	chartFetcher ChartFetcher
}

func NewCreator(bin BinaryFetcher, img ImageExporter, chart ChartFetcher) *Creator {
	return &Creator{
		binFetcher:   bin,
		imgExporter:  img,
		chartFetcher: chart,
	}
}

func (c *Creator) Create(ctx context.Context, spec BundleSpec, outputDir string) (*BundleManifest, error) {
	if spec.Name == "" {
		return nil, fmt.Errorf("bundle spec name is required")
	}
	if len(spec.Binaries) == 0 {
		return nil, fmt.Errorf("at least one binary is required")
	}

	dirs := []string{
		filepath.Join(outputDir, "binaries"),
		filepath.Join(outputDir, "images"),
		filepath.Join(outputDir, "charts"),
		filepath.Join(outputDir, "configs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	manifest := &BundleManifest{
		Name:      spec.Name,
		Version:   spec.Version,
		Distro:    spec.Distro,
		Arch:      spec.Arch,
		CreatedAt: time.Now(),
		Binaries:  []string{},
		Images:    []string{},
		Charts:    []string{},
	}

	for _, bin := range spec.Binaries {
		data, err := c.binFetcher.Fetch(ctx, bin.Name, bin.Version, spec.Arch)
		if err != nil {
			return nil, fmt.Errorf("fetching binary %s: %w", bin.Name, err)
		}
		dest := filepath.Join(outputDir, "binaries", bin.Name)
		if err := os.WriteFile(dest, data, 0o755); err != nil {
			return nil, fmt.Errorf("writing binary %s: %w", bin.Name, err)
		}
		manifest.Binaries = append(manifest.Binaries, bin.Name)
	}

	if len(spec.Images) > 0 {
		imagesDir := filepath.Join(outputDir, "images")
		if err := c.imgExporter.Export(ctx, spec.Images, imagesDir); err != nil {
			return nil, fmt.Errorf("exporting images: %w", err)
		}
		manifest.Images = append(manifest.Images, spec.Images...)
	}

	for _, ch := range spec.Charts {
		data, err := c.chartFetcher.Fetch(ctx, ch.Repository, ch.Chart, ch.Version)
		if err != nil {
			return nil, fmt.Errorf("fetching chart %s: %w", ch.Chart, err)
		}
		filename := fmt.Sprintf("%s-%s.tgz", ch.Chart, ch.Version)
		dest := filepath.Join(outputDir, "charts", filename)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return nil, fmt.Errorf("writing chart %s: %w", ch.Chart, err)
		}
		manifest.Charts = append(manifest.Charts, filename)
	}

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling manifest: %w", err)
	}
	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		return nil, fmt.Errorf("writing manifest: %w", err)
	}

	return manifest, nil
}

func (c *Creator) CreateArchive(dir, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating archive file: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})
}
