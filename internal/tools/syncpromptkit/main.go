// Command syncpromptkit refreshes the pinned embedded PromptKit component
// snapshot. It is a maintainer tool invoked through go generate.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	maxArchiveBytes = 64 << 20
	maxFileBytes    = 16 << 20
)

type lockFile struct {
	Repository string            `json:"repository"`
	Ref        string            `json:"ref"`
	Commit     string            `json:"commit"`
	Files      map[string]string `json:"files"`
}

func main() {
	lockPath := flag.String("lock", "content/upstream.json", "upstream lock file")
	destination := flag.String("dest", "content/promptkit", "embedded content destination")
	refreshLock := flag.Bool("refresh-lock", false, "replace the expected SHA-256 inventory")
	flag.Parse()

	if err := run(*lockPath, *destination, *refreshLock); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(lockPath, destination string, refreshLock bool) error {
	lock, err := readLock(lockPath)
	if err != nil {
		return err
	}
	if lock.Repository == "" || lock.Ref == "" || len(lock.Commit) != 40 {
		return fmt.Errorf("lock must contain repository, ref, and a full commit SHA")
	}

	temporary, err := os.MkdirTemp("", "promptkitty-sync-*")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(temporary) }()

	extracted := filepath.Join(temporary, "promptkit")
	if err := os.MkdirAll(extracted, 0o750); err != nil {
		return fmt.Errorf("create extraction directory: %w", err)
	}
	if err := downloadAndExtract(lock, extracted); err != nil {
		return err
	}

	inventory, err := hashInventory(extracted)
	if err != nil {
		return err
	}
	if refreshLock {
		lock.Files = inventory
	} else if err := compareInventory(lock.Files, inventory); err != nil {
		return err
	}

	if err := os.RemoveAll(destination); err != nil {
		return fmt.Errorf("remove previous embedded content: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return fmt.Errorf("create content parent: %w", err)
	}
	if err := os.Rename(extracted, destination); err != nil {
		return fmt.Errorf("install embedded content: %w", err)
	}

	if refreshLock {
		encoded, marshalErr := json.MarshalIndent(lock, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("encode upstream lock: %w", marshalErr)
		}
		encoded = append(encoded, '\n')
		if writeErr := os.WriteFile(lockPath, encoded, 0o600); writeErr != nil {
			return fmt.Errorf("write upstream lock: %w", writeErr)
		}
	}

	return nil
}

func readLock(filename string) (lockFile, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return lockFile{}, fmt.Errorf("read upstream lock: %w", err)
	}

	var lock lockFile
	if err := json.Unmarshal(raw, &lock); err != nil {
		return lockFile{}, fmt.Errorf("parse upstream lock: %w", err)
	}

	return lock, nil
}

func downloadAndExtract(lock lockFile, destination string) error {
	url := fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", lock.Repository, lock.Commit)
	client := &http.Client{Timeout: 60 * time.Second}
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create archive request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download PromptKit archive: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download PromptKit archive: HTTP %s", response.Status)
	}

	limited := io.LimitReader(response.Body, maxArchiveBytes+1)
	gzipReader, err := gzip.NewReader(limited)
	if err != nil {
		return fmt.Errorf("open PromptKit archive: %w", err)
	}
	defer func() { _ = gzipReader.Close() }()

	reader := tar.NewReader(gzipReader)
	foundManifest := false
	for {
		header, nextErr := reader.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return fmt.Errorf("read PromptKit archive: %w", nextErr)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}

		_, relative, ok := strings.Cut(header.Name, "/")
		if !ok || !included(relative) {
			continue
		}
		clean := path.Clean(relative)
		if clean != relative || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		if header.Size < 0 || header.Size > maxFileBytes {
			return fmt.Errorf("PromptKit file %q exceeds size limit", relative)
		}

		target := filepath.Join(destination, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return fmt.Errorf("create directory for %q: %w", relative, err)
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("create %q: %w", relative, err)
		}
		_, copyErr := io.CopyN(file, reader, header.Size)
		closeErr := file.Close()
		if copyErr != nil {
			return fmt.Errorf("extract %q: %w", relative, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("close %q: %w", relative, closeErr)
		}
		foundManifest = foundManifest || relative == "manifest.yaml"
	}
	if !foundManifest {
		return fmt.Errorf("PromptKit archive does not contain manifest.yaml")
	}

	return nil
}

func included(filename string) bool {
	if filename == "manifest.yaml" {
		return true
	}
	if path.Ext(filename) != ".md" {
		return false
	}
	for _, directory := range []string{"personas/", "protocols/", "formats/", "taxonomies/", "templates/"} {
		if strings.HasPrefix(filename, directory) {
			return true
		}
	}

	return false
}

func hashInventory(root string) (map[string]string, error) {
	inventory := make(map[string]string)
	err := filepath.WalkDir(root, func(filename string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		raw, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, filename)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		inventory[filepath.ToSlash(relative)] = hex.EncodeToString(sum[:])

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("hash embedded content: %w", err)
	}

	return inventory, nil
}

func compareInventory(expected, actual map[string]string) error {
	if len(expected) == 0 {
		return fmt.Errorf("upstream lock has no file inventory; run with -refresh-lock")
	}
	keys := make([]string, 0, len(expected)+len(actual))
	seen := make(map[string]bool)
	for key := range expected {
		seen[key] = true
		keys = append(keys, key)
	}
	for key := range actual {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if expected[key] != actual[key] {
			return fmt.Errorf("upstream inventory mismatch for %q", key)
		}
	}

	return nil
}
