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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	fullCommitLength = 40
	maxArchiveBytes  = 64 << 20
	maxFileBytes     = 16 << 20
	githubAPIBase    = "https://api.github.com"
)

type lockFile struct {
	Repository    string            `json:"repository"`
	Ref           string            `json:"ref"`
	Commit        string            `json:"commit"`
	LicenseSHA256 string            `json:"license_sha256"`
	Files         map[string]string `json:"files"`
}

func main() {
	lockPath := flag.String("lock", "content/upstream.json", "upstream lock file")
	destination := flag.String("dest", "content/promptkit", "embedded content destination")
	licenseDestination := flag.String("license", "third_party/promptkit/LICENSE", "upstream license destination")
	update := flag.Bool("update", false, "resolve the configured ref and refresh the lock")
	flag.Parse()

	if err := run(*lockPath, *destination, *licenseDestination, *update); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(lockPath, destination, licenseDestination string, update bool) error {
	lock, err := readLock(lockPath)
	if err != nil {
		return err
	}

	if err := validateLock(lock, update); err != nil {
		return err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	if update {
		lock.Commit, err = resolveCommit(client, githubAPIBase, lock.Repository, lock.Ref)
		if err != nil {
			return err
		}
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

	extractedLicense := filepath.Join(temporary, "LICENSE")
	if err := downloadAndExtract(client, archiveURL(lock), extracted, extractedLicense); err != nil {
		return err
	}

	inventory, err := hashInventory(extracted)
	if err != nil {
		return err
	}

	licenseSHA256, err := hashFile(extractedLicense)
	if err != nil {
		return err
	}

	if update {
		lock.Files = inventory
		lock.LicenseSHA256 = licenseSHA256
	} else if err := compareInventory(lock.Files, inventory); err != nil {
		return err
	} else if lock.LicenseSHA256 != licenseSHA256 {
		return fmt.Errorf("upstream license checksum mismatch")
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

	if err := installLicense(extractedLicense, licenseDestination); err != nil {
		return err
	}

	if update {
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

func validateLock(lock lockFile, update bool) error {
	owner, repository, ok := strings.Cut(lock.Repository, "/")
	if !ok || owner == "" || repository == "" || strings.Contains(repository, "/") || lock.Ref == "" {
		return fmt.Errorf("lock must contain repository as owner/name and a ref")
	}

	if !update {
		if err := validateCommit(lock.Commit); err != nil {
			return fmt.Errorf("lock commit: %w", err)
		}

		if lock.LicenseSHA256 == "" {
			return fmt.Errorf("lock must contain license_sha256; regenerate the snapshot")
		}
	}

	return nil
}

func validateCommit(commit string) error {
	if len(commit) != fullCommitLength {
		return fmt.Errorf("must be a full 40-character SHA-1")
	}

	if _, err := hex.DecodeString(commit); err != nil {
		return fmt.Errorf("must be hexadecimal: %w", err)
	}

	return nil
}

func resolveCommit(client *http.Client, apiBase, repository, ref string) (string, error) {
	owner, name, _ := strings.Cut(repository, "/")
	endpoint := fmt.Sprintf(
		"%s/repos/%s/%s/commits/%s",
		strings.TrimRight(apiBase, "/"),
		url.PathEscape(owner),
		url.PathEscape(name),
		url.PathEscape(ref),
	)

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create ref request: %w", err)
	}

	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "promptkitty-sync")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("resolve PromptKit ref %q: %w", ref, err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("resolve PromptKit ref %q: HTTP %s", ref, response.Status)
	}

	var result struct {
		SHA string `json:"sha"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("decode PromptKit ref %q: %w", ref, err)
	}

	if err := validateCommit(result.SHA); err != nil {
		return "", fmt.Errorf("resolve PromptKit ref %q: %w", ref, err)
	}

	return result.SHA, nil
}

func archiveURL(lock lockFile) string {
	return fmt.Sprintf("https://github.com/%s/archive/%s.tar.gz", lock.Repository, lock.Commit)
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

func downloadAndExtract(client *http.Client, archiveURL, destination, licenseDestination string) error {
	request, err := http.NewRequest(http.MethodGet, archiveURL, nil)
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
	foundLicense := false

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
		if !ok || (relative != "LICENSE" && !included(relative)) {
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
		if relative == "LICENSE" {
			target = licenseDestination
		}

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
		foundLicense = foundLicense || relative == "LICENSE"
	}

	if !foundManifest {
		return fmt.Errorf("PromptKit archive does not contain manifest.yaml")
	}

	if !foundLicense {
		return fmt.Errorf("PromptKit archive does not contain LICENSE")
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

func hashFile(filename string) (string, error) {
	raw, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("hash %q: %w", filename, err)
	}

	sum := sha256.Sum256(raw)

	return hex.EncodeToString(sum[:]), nil
}

func installLicense(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return fmt.Errorf("create upstream license directory: %w", err)
	}

	if err := os.Remove(destination); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove previous upstream license: %w", err)
	}

	if err := os.Rename(source, destination); err != nil {
		return fmt.Errorf("install upstream license: %w", err)
	}

	return nil
}

func compareInventory(expected, actual map[string]string) error {
	if len(expected) == 0 {
		return fmt.Errorf("upstream lock has no file inventory; regenerate the snapshot")
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
