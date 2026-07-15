package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const testCommit = "5fb0b1a53b2abe13a80123a77b4110bcd074e449"

func TestResolveCommit(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got, want := request.URL.Path, "/repos/microsoft/PromptKit/commits/v0.6.1"; got != want {
			t.Errorf("request path = %q, want %q", got, want)
		}

		if got, want := request.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Errorf("Authorization = %q, want %q", got, want)
		}

		_, _ = writer.Write([]byte(`{"sha":"` + testCommit + `"}`))
	}))
	t.Cleanup(server.Close)

	got, err := resolveCommit(server.Client(), server.URL, "microsoft/PromptKit", "v0.6.1")
	if err != nil {
		t.Fatalf("resolveCommit() returned unexpected error: %v", err)
	}

	if got != testCommit {
		t.Errorf("resolveCommit() = %q, want %q", got, testCommit)
	}
}

func TestDownloadAndExtract(t *testing.T) {
	archive := promptKitArchive(t, map[string]string{
		"PromptKit-test/LICENSE":                  "upstream license\n",
		"PromptKit-test/README.md":                "not vendored\n",
		"PromptKit-test/manifest.yaml":            "version: test\n",
		"PromptKit-test/templates/review-code.md": "# Review\n",
	})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/gzip")
		_, _ = writer.Write(archive)
	}))
	t.Cleanup(server.Close)

	temporary := t.TempDir()
	destination := filepath.Join(temporary, "content")
	licenseDestination := filepath.Join(temporary, "LICENSE")
	if err := os.MkdirAll(destination, 0o750); err != nil {
		t.Fatal(err)
	}

	if err := downloadAndExtract(server.Client(), server.URL, destination, licenseDestination); err != nil {
		t.Fatalf("downloadAndExtract() returned unexpected error: %v", err)
	}

	for filename, want := range map[string]string{
		filepath.Join(destination, "manifest.yaml"):               "version: test\n",
		filepath.Join(destination, "templates", "review-code.md"): "# Review\n",
		licenseDestination: "upstream license\n",
	} {
		got, err := os.ReadFile(filename)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) returned unexpected error: %v", filename, err)
		}

		if string(got) != want {
			t.Errorf("%s = %q, want %q", filename, got, want)
		}
	}

	if _, err := os.Stat(filepath.Join(destination, "README.md")); !os.IsNotExist(err) {
		t.Errorf("README.md was extracted: %v", err)
	}
}

func TestValidateLock(t *testing.T) {
	tests := []struct {
		name   string
		lock   lockFile
		update bool
		wantOK bool
	}{
		{
			name:   "update needs ref only",
			lock:   lockFile{Repository: "microsoft/PromptKit", Ref: "v0.6.1"},
			update: true,
			wantOK: true,
		},
		{
			name:   "verification needs complete lock",
			lock:   lockFile{Repository: "microsoft/PromptKit", Ref: "v0.6.1", Commit: testCommit, LicenseSHA256: "checksum"},
			wantOK: true,
		},
		{name: "invalid repository", lock: lockFile{Repository: "PromptKit", Ref: "v0.6.1"}},
		{name: "missing commit", lock: lockFile{Repository: "microsoft/PromptKit", Ref: "v0.6.1"}},
		{name: "missing license hash", lock: lockFile{Repository: "microsoft/PromptKit", Ref: "v0.6.1", Commit: testCommit}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateLock(test.lock, test.update)
			if gotOK := err == nil; gotOK != test.wantOK {
				t.Errorf("validateLock() error = %v, wantOK %t", err, test.wantOK)
			}
		})
	}
}

func TestPinnedSnapshotMatchesLock(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	lock, err := readLock(filepath.Join(root, "content", "upstream.json"))
	if err != nil {
		t.Fatalf("readLock() returned unexpected error: %v", err)
	}

	inventory, err := hashInventory(filepath.Join(root, "content", "promptkit"))
	if err != nil {
		t.Fatalf("hashInventory() returned unexpected error: %v", err)
	}

	if err := compareInventory(lock.Files, inventory); err != nil {
		t.Fatalf("compareInventory() returned unexpected error: %v", err)
	}

	licenseSHA256, err := hashFile(filepath.Join(root, "third_party", "promptkit", "LICENSE"))
	if err != nil {
		t.Fatalf("hashFile(upstream license) returned unexpected error: %v", err)
	}

	if licenseSHA256 != lock.LicenseSHA256 {
		t.Errorf("upstream license SHA-256 = %q, want %q", licenseSHA256, lock.LicenseSHA256)
	}
}

func promptKitArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for filename, content := range files {
		header := &tar.Header{Name: filename, Mode: 0o600, Size: int64(len(content)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}

		if _, err := tarWriter.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}

	return buffer.Bytes()
}
