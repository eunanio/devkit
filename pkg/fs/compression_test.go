package fs

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompressDir(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (string, func(), error)
		expectError bool
	}{
		{
			name: "Compress directory with files and subdirectories",
			setup: func() (string, func(), error) {
				// Create a temporary directory
				dir, err := ioutil.TempDir("", "testdir")
				if err != nil {
					return "", nil, err
				}
				// Create files and subdirectories
				subDir := filepath.Join(dir, "subdir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					return "", nil, err
				}
				if err := ioutil.WriteFile(filepath.Join(dir, "file1.txt"), []byte("Hello, World!"), 0644); err != nil {
					return "", nil, err
				}
				if err := ioutil.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("Hello, Go!"), 0644); err != nil {
					return "", nil, err
				}
				return dir, func() { os.RemoveAll(dir) }, nil
			},
			expectError: false,
		},
		{
			name: "Compress empty directory",
			setup: func() (string, func(), error) {
				dir, err := ioutil.TempDir("", "emptydir")
				if err != nil {
					return "", nil, err
				}
				return dir, func() { os.RemoveAll(dir) }, nil
			},
			expectError: false,
		},
		{
			name: "Non-existent directory",
			setup: func() (string, func(), error) {
				// Return a path that doesn't exist
				return "/path/does/not/exist", func() {}, nil
			},
			expectError: true,
		},
		{
			name: "Directory with symbolic links",
			setup: func() (string, func(), error) {
				dir, err := ioutil.TempDir("", "symlinkdir")
				if err != nil {
					return "", nil, err
				}
				targetFile := filepath.Join(dir, "target.txt")
				if err := ioutil.WriteFile(targetFile, []byte("Target File"), 0644); err != nil {
					return "", nil, err
				}
				symlinkFile := filepath.Join(dir, "symlink.txt")
				if err := os.Symlink(targetFile, symlinkFile); err != nil {
					return "", nil, err
				}
				return dir, func() { os.RemoveAll(dir) }, nil
			},
			expectError: false,
		},
		{
			name: "Directory with files of different permissions",
			setup: func() (string, func(), error) {
				dir, err := ioutil.TempDir("", "permdir")
				if err != nil {
					return "", nil, err
				}
				readOnlyFile := filepath.Join(dir, "readonly.txt")
				if err := ioutil.WriteFile(readOnlyFile, []byte("Read Only"), 0444); err != nil {
					return "", nil, err
				}
				execFile := filepath.Join(dir, "exec.sh")
				if err := ioutil.WriteFile(execFile, []byte("#!/bin/sh\necho Hello"), 0755); err != nil {
					return "", nil, err
				}
				return dir, func() { os.RemoveAll(dir) }, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			srcDir, cleanup, err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer cleanup()

			data, err := CompressDir(srcDir)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}
			if !tt.expectError {
				if len(data) == 0 {
					t.Errorf("Expected non-empty data")
				} else {
					// Optional: Verify the contents of the tar.gz archive
					if err := verifyArchive(data, srcDir); err != nil {
						t.Errorf("Archive verification failed: %v", err)
					}
				}
			}
		})
	}
}

// verifyArchive checks that the tar.gz data contains the same files as the source directory
func verifyArchive(data []byte, srcDir string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	filesInArchive := make(map[string]bool)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		filesInArchive[hdr.Name] = true
	}

	// Walk the source directory and check if all files are in the archive
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		// Convert Windows path separators to Unix style for tar archive
		relativePath = filepath.ToSlash(relativePath)
		if !strings.Contains(relativePath, "/.") && relativePath != "." {
			if !filesInArchive[relativePath] {
				return fmt.Errorf("file %s not found in archive", relativePath)
			}
		}
		return nil
	})
	return err
}

func TestDecompressDir(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() ([]byte, func(), error)
		expectError bool
	}{
		{
			name: "Decompress valid archive with files and directories",
			setup: func() ([]byte, func(), error) {
				// Create a temporary source directory with files and subdirectories
				srcDir, err := ioutil.TempDir("", "testsrcdir")
				if err != nil {
					return nil, nil, err
				}
				// Create files and subdirectories
				subDir := filepath.Join(srcDir, "subdir")
				if err := os.Mkdir(subDir, 0755); err != nil {
					return nil, nil, err
				}
				if err := ioutil.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("Hello, World!"), 0644); err != nil {
					return nil, nil, err
				}
				if err := ioutil.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("Hello, Go!"), 0644); err != nil {
					return nil, nil, err
				}
				// Compress the source directory
				data, err := CompressDir(srcDir)
				if err != nil {
					return nil, nil, err
				}
				// Cleanup function
				cleanup := func() {
					os.RemoveAll(srcDir)
				}
				return data, cleanup, nil
			},
			expectError: false,
		},
		{
			name: "Decompress empty archive",
			setup: func() ([]byte, func(), error) {
				// Create an empty tar.gz archive
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)
				tw.Close()
				gw.Close()
				return buf.Bytes(), func() {}, nil
			},
			expectError: false,
		},
		{
			name: "Decompress invalid archive (corrupted data)",
			setup: func() ([]byte, func(), error) {
				// Create invalid tar.gz data
				data := []byte("this is not a valid gzip data")
				return data, func() {}, nil
			},
			expectError: true,
		},
		{
			name: "Decompress archive with path traversal filenames",
			setup: func() ([]byte, func(), error) {
				// Create a tar.gz archive with a file that has a "../" in its name
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)
				hdr := &tar.Header{
					Name: "../evil.txt",
					Mode: 0600,
					Size: int64(len("You have been hacked!")),
				}
				if err := tw.WriteHeader(hdr); err != nil {
					return nil, nil, err
				}
				if _, err := tw.Write([]byte("You have been hacked!")); err != nil {
					return nil, nil, err
				}
				tw.Close()
				gw.Close()
				return buf.Bytes(), func() {}, nil
			},
			expectError: true,
		},
		{
			name: "Decompress archive with unsupported file type",
			setup: func() ([]byte, func(), error) {
				// Create a tar.gz archive with a symbolic link
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)
				hdr := &tar.Header{
					Name:     "symlink",
					Mode:     0777,
					Linkname: "somefile",
					Typeflag: tar.TypeSymlink,
				}
				if err := tw.WriteHeader(hdr); err != nil {
					return nil, nil, err
				}
				tw.Close()
				gw.Close()
				return buf.Bytes(), func() {}, nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			data, cleanup, err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer cleanup()

			// Create a temporary destination directory
			dstDir, err := ioutil.TempDir("", "testdstdir")
			if err != nil {
				t.Fatalf("Failed to create destination directory: %v", err)
			}
			defer os.RemoveAll(dstDir)

			err = DecompressDir(data, dstDir)
			if (err != nil) != tt.expectError {
				t.Errorf("Expected error: %v, got: %v", tt.expectError, err)
			}

			if !tt.expectError {
				// Optional: Verify the contents of the destination directory
				if err := verifyDecompressedContent(data, dstDir); err != nil {
					t.Errorf("Decompressed content verification failed: %v", err)
				}
			}
		})
	}
}

// verifyDecompressedContent checks that the decompressed files match the original archive contents
func verifyDecompressedContent(data []byte, dstDir string) error {
	// Read the archive contents
	var filesInArchive []string
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		filesInArchive = append(filesInArchive, hdr.Name)
	}

	// Walk the destination directory and check if all files are extracted
	err = filepath.Walk(dstDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(dstDir, path)
		if err != nil {
			return err
		}
		// Ignore the root directory
		if relativePath == "." {
			return nil
		}
		// Convert Windows path separators to Unix style for comparison
		relativePath = filepath.ToSlash(relativePath)
		for _, fileName := range filesInArchive {
			if fileName == relativePath {
				return nil
			}
		}
		return fmt.Errorf("file %s not found in archive", relativePath)
	})
	return err
}
