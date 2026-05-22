package compression

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

type Compressor struct {
	config *config.CompressionConfig
	logger *logger.Logger
}

func NewCompressor(cfg *config.CompressionConfig, log *logger.Logger) *Compressor {
	return &Compressor{config: cfg, logger: log}
}

func (c *Compressor) CompressBackup(backupDir string) (string, error) {
	if !c.config.Enabled {
		return backupDir, nil
	}

	c.logger.WithField("backup_dir", backupDir).Info("Starting backup compression")
	startTime := time.Now()

	var outputFile string
	var err error
	switch strings.ToLower(c.config.Format) {
	case "tar.gz", "tgz":
		outputFile = backupDir + ".tar.gz"
		err = c.createTarGz(backupDir, outputFile)
	case "tar.zst":
		outputFile = backupDir + ".tar.zst"
		err = c.createTarZst(backupDir, outputFile)
	case "tar.xz":
		outputFile = backupDir + ".tar.xz"
		err = c.createTarXz(backupDir, outputFile)
	default:
		return "", fmt.Errorf("unsupported compression format: %s", c.config.Format)
	}

	if err != nil {
		return "", fmt.Errorf("failed to compress backup: %w", err)
	}

	originalSize, _ := c.getDirSize(backupDir)
	compressedSize, _ := c.getFileSize(outputFile)
	ratio := float64(compressedSize) / float64(originalSize) * 100

	c.logger.WithField("original_size", c.formatSize(originalSize)).
		WithField("compressed_size", c.formatSize(compressedSize)).
		WithField("compression_ratio", fmt.Sprintf("%.1f%%", ratio)).
		WithField("duration", time.Since(startTime)).
		Info("Backup compression completed")

	if !c.config.KeepOriginal {
		if err := os.RemoveAll(backupDir); err != nil {
			c.logger.WithError(err).Warn("Failed to remove original backup directory")
		} else {
			c.logger.WithField("directory", backupDir).Info("Removed original backup directory")
		}
	}

	return outputFile, nil
}

func (c *Compressor) DecompressBackup(archiveFile string) (string, error) {
	if !c.isCompressedFile(archiveFile) {
		return archiveFile, nil
	}

	c.logger.WithField("archive", archiveFile).Info("Starting backup decompression")
	startTime := time.Now()

	outputDir := strings.TrimSuffix(archiveFile, filepath.Ext(archiveFile))
	outputDir = strings.TrimSuffix(outputDir, ".tar")

	lower := strings.ToLower(archiveFile)
	var err error
	switch {
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		err = c.extractTarGz(archiveFile, outputDir)
	case strings.HasSuffix(lower, ".tar.zst"):
		err = c.extractTarZst(archiveFile, outputDir)
	case strings.HasSuffix(lower, ".tar.xz"):
		err = c.extractTarXz(archiveFile, outputDir)
	default:
		return "", fmt.Errorf("unsupported archive format: %s", archiveFile)
	}

	if err != nil {
		return "", fmt.Errorf("failed to decompress backup: %w", err)
	}

	c.logger.WithField("output_dir", outputDir).
		WithField("duration", time.Since(startTime)).
		Info("Backup decompression completed")

	return outputDir, nil
}

// writeTarTo writes a tar archive of sourceDir to w.
func (c *Compressor) writeTarTo(sourceDir string, w io.Writer) error {
	tarWriter := tar.NewWriter(w)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}
		header.Name = relPath
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tarWriter, f)
			return err
		}
		return nil
	})
}

// extractTarFrom extracts a tar archive from r into outputDir.
func (c *Compressor) extractTarFrom(r io.Reader, outputDir string) error {
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		filePath := filepath.Join(outputDir, header.Name)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filePath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Compressor) createTarGz(sourceDir, targetFile string) error {
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var gzipWriter *gzip.Writer
	if c.config.Level >= 1 && c.config.Level <= 9 {
		gzipWriter, err = gzip.NewWriterLevel(file, c.config.Level)
		if err != nil {
			return err
		}
	} else {
		gzipWriter = gzip.NewWriter(file)
	}
	defer gzipWriter.Close()

	return c.writeTarTo(sourceDir, gzipWriter)
}

func (c *Compressor) createTarZst(sourceDir, targetFile string) error {
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	level := zstd.SpeedDefault
	if c.config.Level >= 7 {
		level = zstd.SpeedBestCompression
	} else if c.config.Level <= 2 && c.config.Level >= 1 {
		level = zstd.SpeedFastest
	}

	encoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(level))
	if err != nil {
		return err
	}
	defer encoder.Close()

	return c.writeTarTo(sourceDir, encoder)
}

func (c *Compressor) createTarXz(sourceDir, targetFile string) error {
	outFile, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	pr, pw := io.Pipe()
	cmd := exec.Command("xz", "--compress", "--stdout")
	cmd.Stdin = pr
	cmd.Stdout = outFile

	if err := cmd.Start(); err != nil {
		pr.Close()
		pw.Close()
		return fmt.Errorf("xz binary not found (install xz-utils): %w", err)
	}

	var tarErr error
	go func() {
		tarErr = c.writeTarTo(sourceDir, pw)
		pw.Close()
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("xz compression failed: %w", err)
	}
	return tarErr
}

func (c *Compressor) extractTarGz(archiveFile, outputDir string) error {
	file, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	return c.extractTarFrom(gzipReader, outputDir)
}

func (c *Compressor) extractTarZst(archiveFile, outputDir string) error {
	file, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder, err := zstd.NewReader(file)
	if err != nil {
		return err
	}
	defer decoder.Close()

	return c.extractTarFrom(decoder, outputDir)
}

func (c *Compressor) extractTarXz(archiveFile, outputDir string) error {
	cmd := exec.Command("xz", "--decompress", "--stdout", archiveFile)
	pr, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("xz binary not found (install xz-utils): %w", err)
	}

	extractErr := c.extractTarFrom(pr, outputDir)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("xz decompression failed: %w", err)
	}
	return extractErr
}

func (c *Compressor) isCompressedFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.zst") ||
		strings.HasSuffix(lower, ".tar.xz")
}

func (c *Compressor) getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func (c *Compressor) getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (c *Compressor) formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d bytes", size)
	}
}
