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
	"strings"

	"github.com/rohitg00/ironkube/pkg/provider"
)

type BinaryInstaller interface {
	Install(ctx context.Context, machine provider.Machine, name string, data []byte, destPath string) error
}

type ImageLoader interface {
	Load(ctx context.Context, machine provider.Machine, archivePath string) error
}

type Loader struct {
	binInstaller BinaryInstaller
	imgLoader    ImageLoader
}

func NewLoader(bin BinaryInstaller, img ImageLoader) *Loader {
	return &Loader{
		binInstaller: bin,
		imgLoader:    img,
	}
}

func (l *Loader) LoadManifest(bundleDir string) (*BundleManifest, error) {
	manifestPath := filepath.Join(bundleDir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var manifest BundleManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	return &manifest, nil
}

func (l *Loader) LoadBundle(ctx context.Context, machine provider.Machine, bundleDir string) error {
	if l.binInstaller == nil {
		return fmt.Errorf("binary installer is required")
	}
	if l.imgLoader == nil {
		return fmt.Errorf("image loader is required")
	}

	manifest, err := l.LoadManifest(bundleDir)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	for _, binName := range manifest.Binaries {
		if strings.ContainsAny(binName, "/\\") || binName == ".." || binName == "." {
			return fmt.Errorf("invalid binary name: %q", binName)
		}
		binPath := filepath.Join(bundleDir, "binaries", binName)
		data, err := os.ReadFile(binPath)
		if err != nil {
			return fmt.Errorf("reading binary %s: %w", binName, err)
		}
		destPath := filepath.Join("/usr/local/bin", binName)
		if err := l.binInstaller.Install(ctx, machine, binName, data, destPath); err != nil {
			return fmt.Errorf("installing binary %s: %w", binName, err)
		}
	}

	for _, imageName := range manifest.Images {
		if strings.ContainsAny(imageName, "/\\") || imageName == ".." || imageName == "." {
			return fmt.Errorf("invalid image name: %q", imageName)
		}
		archivePath := filepath.Join(bundleDir, "images", imageName)
		if err := l.imgLoader.Load(ctx, machine, archivePath); err != nil {
			return fmt.Errorf("loading image %s: %w", imageName, err)
		}
	}

	return nil
}

func (l *Loader) ExtractArchive(archivePath, outputDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}

		target := filepath.Join(outputDir, header.Name)

		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", target, err)
		}
		absOutputDir, err := filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("resolving output dir: %w", err)
		}
		if !strings.HasPrefix(absTarget, absOutputDir+string(filepath.Separator)) && absTarget != absOutputDir {
			return fmt.Errorf("path traversal detected: %s escapes %s", header.Name, outputDir)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("creating parent directory for %s: %w", target, err)
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("creating file %s: %w", target, err)
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("writing file %s: %w", target, err)
			}
			outFile.Close()
		}
	}

	return nil
}
