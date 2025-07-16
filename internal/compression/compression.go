package compression

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abdullahainun/tenangdb/internal/config"
	"github.com/abdullahainun/tenangdb/internal/logger"
)

// Compressor handles backup compression operations
type Compressor struct {
	config *config.CompressionConfig
	logger *logger.Logger
}

// NewCompressor creates a new compressor instance
func NewCompressor(cfg *config.CompressionConfig, log *logger.Logger) *Compressor {
	return &Compressor{
		config: cfg,
		logger: log,
	}
}

// CompressBackup compresses a backup directory
func (c *Compressor) CompressBackup(backupDir string) (string, error) {
	if !c.config.Enabled {
		return backupDir, nil
	}

	c.logger.WithField("backup_dir", backupDir).Info("Starting backup compression")
	startTime := time.Now()

	// Determine output file name
	var outputFile string
	switch strings.ToLower(c.config.Format) {
	case "tar.gz", "tgz":
		outputFile = backupDir + ".tar.gz"
	case "tar.zst":
		outputFile = backupDir + ".tar.zst"
	case "tar.xz":
		outputFile = backupDir + ".tar.xz"
	default:
		return "", fmt.Errorf("unsupported compression format: %s", c.config.Format)
	}

	// Create compressed archive
	err := c.createTarGz(backupDir, outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to compress backup: %w", err)
	}

	// Calculate compression ratio
	originalSize, _ := c.getDirSize(backupDir)
	compressedSize, _ := c.getFileSize(outputFile)
	ratio := float64(compressedSize) / float64(originalSize) * 100

	c.logger.WithField("original_size", c.formatSize(originalSize)).
		WithField("compressed_size", c.formatSize(compressedSize)).
		WithField("compression_ratio", fmt.Sprintf("%.1f%%", ratio)).
		WithField("duration", time.Since(startTime)).
		Info("Backup compression completed")

	// Remove original directory if not keeping original
	if !c.config.KeepOriginal {
		if err := os.RemoveAll(backupDir); err != nil {
			c.logger.WithError(err).Warn("Failed to remove original backup directory")
		} else {
			c.logger.WithField("directory", backupDir).Info("Removed original backup directory")
		}
	}

	return outputFile, nil
}

// DecompressBackup decompresses a backup archive for restore
func (c *Compressor) DecompressBackup(archiveFile string) (string, error) {
	if !c.isCompressedFile(archiveFile) {
		return archiveFile, nil
	}

	c.logger.WithField("archive", archiveFile).Info("Starting backup decompression")
	startTime := time.Now()

	// Determine output directory
	outputDir := strings.TrimSuffix(archiveFile, filepath.Ext(archiveFile))
	outputDir = strings.TrimSuffix(outputDir, ".tar")

	// Extract archive
	err := c.extractTarGz(archiveFile, outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to decompress backup: %w", err)
	}

	c.logger.WithField("output_dir", outputDir).
		WithField("duration", time.Since(startTime)).
		Info("Backup decompression completed")

	return outputDir, nil
}

// createTarGz creates a tar.gz archive from a directory
func (c *Compressor) createTarGz(sourceDir, targetFile string) error {
	// Create output file
	file, err := os.Create(targetFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	// Set compression level
	if c.config.Level >= 1 && c.config.Level <= 9 {
		gzipWriter.Close()
		gzipWriter, err = gzip.NewWriterLevel(file, c.config.Level)
		if err != nil {
			return err
		}
		defer gzipWriter.Close()
	}

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Update name to be relative to source directory
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// extractTarGz extracts a tar.gz archive to a directory
func (c *Compressor) extractTarGz(archiveFile, outputDir string) error {
	// Open archive file
	file, err := os.Open(archiveFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Determine file path
		filePath := filepath.Join(outputDir, header.Name)

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		// Extract file based on type
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

// isCompressedFile checks if a file is a compressed archive
func (c *Compressor) isCompressedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".gz" || ext == ".zst" || ext == ".xz" || 
		   strings.HasSuffix(strings.ToLower(filename), ".tar.gz") ||
		   strings.HasSuffix(strings.ToLower(filename), ".tar.zst") ||
		   strings.HasSuffix(strings.ToLower(filename), ".tar.xz")
}

// getDirSize calculates the total size of a directory
func (c *Compressor) getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
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

// getFileSize returns the size of a file
func (c *Compressor) getFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// formatSize formats file size in human readable format
func (c *Compressor) formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	if size >= GB {
		return fmt.Sprintf("%.1f GB", float64(size)/GB)
	} else if size >= MB {
		return fmt.Sprintf("%.1f MB", float64(size)/MB)
	} else if size >= KB {
		return fmt.Sprintf("%.1f KB", float64(size)/KB)
	}
	
	return fmt.Sprintf("%d bytes", size)
}