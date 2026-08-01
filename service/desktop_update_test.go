package service

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func withDesktopUpdateTestState(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("DESKTOP_UPDATE_DIR", root)
	t.Setenv("DESKTOP_UPDATE_ENABLED", "false")
	t.Setenv("DESKTOP_UPDATE_PUBLIC_BASE_URL", "")
	t.Setenv("DESKTOP_UPDATE_PUBLISH_TOKEN", "")
	setDesktopUpdateTestOption(t, DesktopUpdateSettingsOptionKey, "", false)
	setDesktopUpdateTestOption(t, DesktopUpdateTokenOptionKey, "", false)
	return root
}

func setDesktopUpdateTestOption(t *testing.T, key string, value any, present bool) {
	t.Helper()
	commonOptionMapLock()
	old, existed := commonOptionMapGet(key)
	if present {
		commonOptionMapSet(key, value)
	} else {
		commonOptionMapDelete(key)
	}
	commonOptionMapUnlock()
	t.Cleanup(func() {
		commonOptionMapLock()
		if existed {
			commonOptionMapSet(key, old)
		} else {
			commonOptionMapDelete(key)
		}
		commonOptionMapUnlock()
	})
}

// Small wrappers keep direct access to the shared option map synchronized.
func commonOptionMapLock()   { common.OptionMapRWMutex.Lock() }
func commonOptionMapUnlock() { common.OptionMapRWMutex.Unlock() }
func commonOptionMapGet(key string) (any, bool) {
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	value, ok := common.OptionMap[key]
	return value, ok
}
func commonOptionMapSet(key string, value any) {
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[key] = common.Interface2String(value)
}
func commonOptionMapDelete(key string) { delete(common.OptionMap, key) }

func TestDesktopUpdateValidationRejectsTraversal(t *testing.T) {
	invalidVersions := []string{"", "v", "1", "1.2", "../1.2.3", "1.2.3/evil", "01.2.3", "1.2.3-01", strings.Repeat("1", 129)}
	for _, version := range invalidVersions {
		if _, err := CanonicalDesktopUpdateVersion(version); err == nil {
			t.Fatalf("expected invalid version %q", version)
		}
	}
	if version, err := CanonicalDesktopUpdateVersion("v1.2.3-beta.1"); err != nil || version != "1.2.3-beta.1" {
		t.Fatalf("unexpected canonical version: %q, %v", version, err)
	}

	invalidFiles := []string{"", "../app.tar.gz", "app/evil.exe", ".hidden.exe", "app.txt", "app..exe", "latest.json", " bundle.exe", "bundle.exe "}
	for _, filename := range invalidFiles {
		if err := ValidateDesktopUpdateFilename(filename); err == nil {
			t.Fatalf("expected invalid filename %q", filename)
		}
	}
	validFiles := []string{"YuanHeng.app.tar.gz", "YuanHeng.app.tar.gz.sig", "YuanHeng_0.1.17_x64-setup.exe", "YuanHeng.AppImage"}
	for _, filename := range validFiles {
		if err := ValidateDesktopUpdateFilename(filename); err != nil {
			t.Fatalf("expected valid filename %q: %v", filename, err)
		}
	}
}

func TestDesktopUpdateInvalidEffectiveSettingsFailClosed(t *testing.T) {
	withDesktopUpdateTestState(t)
	t.Setenv("DESKTOP_UPDATE_ENABLED", "true")
	t.Setenv("DESKTOP_UPDATE_PUBLIC_BASE_URL", "ftp://updates.example.com")
	if settings := GetDesktopUpdateSettings(); settings.Enabled {
		t.Fatalf("invalid environment configuration must disable public serving: %+v", settings)
	}

	setDesktopUpdateTestOption(t, DesktopUpdateSettingsOptionKey, `{"enabled":true,"public_base_url":"https://user:pass@updates.example.com","max_upload_mb":256,"retention_count":10}`, true)
	if settings := GetDesktopUpdateSettings(); settings.Enabled {
		t.Fatalf("credential-bearing persisted URL must disable public serving: %+v", settings)
	}
}

func TestDesktopUpdatePublishRewritesURLsAndPreservesPreviousManifest(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	baseURL := "https://updates.example.com/desktop/update"
	artifact := "YuanHeng.app.tar.gz"
	if _, err := SaveDesktopUpdateArtifact("v1.2.3", artifact, strings.NewReader("signed-bundle"), 1024); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"1.2.3","notes":"release","pub_date":"2026-08-01T00:00:00Z","custom":{"keep":true},"platforms":{"darwin-aarch64":{"signature":"trusted-signature","url":"https://github.com/example/release/YuanHeng.app.tar.gz"}}}`
	summary, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), baseURL, 10)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Version != "1.2.3" || len(summary.Platforms) != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	latest, err := os.ReadFile(filepath.Join(root, latestManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(latest, []byte(baseURL+"/releases/1.2.3/"+artifact)) {
		t.Fatalf("manifest URL was not rewritten: %s", latest)
	}
	if !bytes.Contains(latest, []byte(`"custom":{"keep":true}`)) {
		t.Fatalf("unknown manifest fields were not preserved: %s", latest)
	}
	previous := append([]byte(nil), latest...)

	invalid := `{"version":"1.2.4","platforms":{"darwin-aarch64":{"signature":"sig","url":"https://example.com/missing.tar.gz"}}}`
	if _, err = PublishDesktopUpdateManifest(strings.NewReader(invalid), baseURL, 10); err == nil {
		t.Fatal("expected missing artifact error")
	}
	after, err := os.ReadFile(filepath.Join(root, latestManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(previous, after) {
		t.Fatal("failed publication changed the active manifest")
	}
}

func TestDesktopUpdateUploadLimitIsAtomic(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	_, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("12345"), 4)
	if err == nil {
		t.Fatal("expected upload size error")
	}
	releaseDir := filepath.Join(root, "releases", "1.2.3")
	if _, statErr := os.Stat(filepath.Join(releaseDir, "bundle.exe")); !os.IsNotExist(statErr) {
		t.Fatalf("oversized upload left a destination file: %v", statErr)
	}
	entries, readErr := os.ReadDir(releaseDir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("oversized upload left temporary files: %+v", entries)
	}
}

func TestDesktopUpdateRejectsEmptyArtifact(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader(""), 1024); err == nil {
		t.Fatal("expected empty artifact error")
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.2.3", "bundle.exe")); !os.IsNotExist(err) {
		t.Fatalf("empty upload left an artifact: %v", err)
	}
}

func TestDesktopUpdateManifestRejectsEmptyStoredArtifact(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	directory := filepath.Join(root, "releases", "1.2.3")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "bundle.exe"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"1.2.3","platforms":{"windows-x86_64":{"signature":"signed","url":"https://example.com/bundle.exe"}}}`
	if _, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), "https://updates.example.com", 10); err == nil {
		t.Fatal("empty stored artifact must not be published")
	}
	if _, err := os.Stat(filepath.Join(root, latestManifestName)); !os.IsNotExist(err) {
		t.Fatalf("failed publication created an active manifest: %v", err)
	}
}

func TestDesktopUpdatePublishTokenFailsClosed(t *testing.T) {
	withDesktopUpdateTestState(t)
	t.Setenv("DESKTOP_UPDATE_PUBLISH_TOKEN", "  environment-token\n")
	if !ValidateDesktopUpdatePublishToken("environment-token") {
		t.Fatal("environment token should be valid")
	}
	if ValidateDesktopUpdatePublishToken("wrong") {
		t.Fatal("wrong token should be rejected")
	}

	hash := HashDesktopUpdatePublishToken("database-token")
	setDesktopUpdateTestOption(t, DesktopUpdateTokenOptionKey, hash, true)
	if ValidateDesktopUpdatePublishToken("environment-token") {
		t.Fatal("database token must override the environment token after rotation")
	}
	if !ValidateDesktopUpdatePublishToken("database-token") {
		t.Fatal("database token should be valid")
	}

	setDesktopUpdateTestOption(t, DesktopUpdateTokenOptionKey, "invalid-hash", true)
	if ValidateDesktopUpdatePublishToken("environment-token") || ValidateDesktopUpdatePublishToken("database-token") {
		t.Fatal("malformed stored token hash must fail closed")
	}
	configured, source := DesktopUpdatePublishTokenStatus()
	if configured || source != "database" {
		t.Fatalf("unexpected invalid-token status: configured=%v source=%q", configured, source)
	}
}

func TestGenerateDesktopUpdatePublishToken(t *testing.T) {
	token, hash, err := GenerateDesktopUpdatePublishToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(token) < 40 {
		t.Fatalf("generated token is too short: %d", len(token))
	}
	decoded, err := hex.DecodeString(hash)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("invalid token hash: %q, %v", hash, err)
	}
	if hash != HashDesktopUpdatePublishToken(token) {
		t.Fatal("generated token hash mismatch")
	}
}

func TestDesktopUpdateArtifactsAreImmutableAndIdempotent(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("original"), 1024); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("original"), 1024); err != nil {
		t.Fatalf("identical retry should be idempotent: %v", err)
	}
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("changed"), 1024); !errors.Is(err, ErrDesktopUpdateImmutable) {
		t.Fatalf("expected immutable conflict, got %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, "releases", "1.2.3", "bundle.exe"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("immutable artifact was overwritten: %q", content)
	}
}

func TestDesktopUpdateRepublishRewritesExistingManifestBaseURL(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("bundle"), 1024); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"1.2.3","platforms":{"windows-x86_64":{"signature":"signed","url":"https://github.com/example/bundle.exe"}}}`
	if _, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), "https://old.example/update", 10); err != nil {
		t.Fatal(err)
	}
	if err := RepublishCurrentDesktopUpdateManifest("https://new.example/update", 10); err != nil {
		t.Fatal(err)
	}
	latest, err := os.ReadFile(filepath.Join(root, latestManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(latest, []byte("https://new.example/update/releases/1.2.3/bundle.exe")) || bytes.Contains(latest, []byte("https://old.example")) {
		t.Fatalf("manifest base URL was not updated: %s", latest)
	}
}

func TestDesktopUpdateCurrentReleaseCannotBeDeletedWhenArtifactIsMissing(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("bundle"), 1024); err != nil {
		t.Fatal(err)
	}
	manifest := `{"version":"1.2.3","platforms":{"windows-x86_64":{"signature":"signed","url":"https://example.com/bundle.exe"}}}`
	if _, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), "https://updates.example.com", 10); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "releases", "1.2.3", "bundle.exe")); err != nil {
		t.Fatal(err)
	}
	if err := DeleteDesktopUpdateRelease("1.2.3"); !errors.Is(err, ErrDesktopUpdateConflict) {
		t.Fatalf("expected current release protection, got %v", err)
	}
}

func TestDesktopUpdateRetentionKeepsConfiguredTotalReleaseCount(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	baseURL := "https://updates.example.com/desktop/update"
	for _, version := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		if _, err := SaveDesktopUpdateArtifact(version, "bundle.exe", strings.NewReader(version), 1024); err != nil {
			t.Fatal(err)
		}
		// Ensure deterministic newest-first ordering even on coarse filesystems.
		directory := filepath.Join(root, "releases", version)
		stamp := map[string]time.Time{
			"1.0.0": time.Unix(100, 0),
			"1.1.0": time.Unix(200, 0),
			"1.2.0": time.Unix(300, 0),
		}[version]
		if err := os.Chtimes(filepath.Join(directory, "bundle.exe"), stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"version":"1.2.0","platforms":{"windows-x86_64":{"signature":"signed","url":"https://example.com/bundle.exe"}}}`
	if _, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), baseURL, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.1.0")); err != nil {
		t.Fatalf("newest old release should be retained: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.0.0")); !os.IsNotExist(err) {
		t.Fatalf("oldest release should be pruned, got %v", err)
	}
}

func TestDesktopUpdateRetentionProtectsStagedAndRecentReleases(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	baseURL := "https://updates.example.com/desktop/update"
	versions := map[string]time.Time{
		"1.0.0": time.Now().Add(-72 * time.Hour),
		"1.1.0": time.Now().Add(-48 * time.Hour),
		"1.2.0": time.Now().Add(-time.Hour),
	}
	for version, stamp := range versions {
		if _, err := SaveDesktopUpdateArtifact(version, "bundle.exe", strings.NewReader(version), 1024); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(filepath.Join(root, "releases", version, "bundle.exe"), stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"version":"1.1.0","platforms":{"windows-x86_64":{"signature":"signed","url":"https://example.com/bundle.exe"}}}`
	if _, err := PublishDesktopUpdateManifest(strings.NewReader(manifest), baseURL, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.0.0")); !os.IsNotExist(err) {
		t.Fatalf("old historical release should be pruned, got %v", err)
	}
	for _, version := range []string{"1.1.0", "1.2.0"} {
		if _, err := os.Stat(filepath.Join(root, "releases", version)); err != nil {
			t.Fatalf("protected release %s was pruned: %v", version, err)
		}
	}
}

func TestDesktopUpdateDeleteFailsClosedWhenManifestIsCorrupt(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("bundle"), 1024); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, latestManifestName), []byte(`{"version":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := DeleteDesktopUpdateRelease("1.2.3"); err == nil {
		t.Fatal("corrupt active manifest must prevent destructive deletion")
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.2.3", "bundle.exe")); err != nil {
		t.Fatalf("release was deleted despite corrupt manifest: %v", err)
	}
}

func TestDesktopUpdateDeleteFailsClosedWhenManifestIsSymlink(t *testing.T) {
	root := withDesktopUpdateTestState(t)
	if _, err := SaveDesktopUpdateArtifact("1.2.3", "bundle.exe", strings.NewReader("bundle"), 1024); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "external-latest.json")
	if err := os.WriteFile(target, []byte(`{"version":"9.9.9"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, latestManifestName)); err != nil {
		t.Skipf("symlink is not supported: %v", err)
	}
	if err := DeleteDesktopUpdateRelease("1.2.3"); err == nil || errors.Is(err, ErrDesktopUpdateNotFound) {
		t.Fatalf("unsafe active manifest must fail closed, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "releases", "1.2.3", "bundle.exe")); err != nil {
		t.Fatalf("release was deleted despite unsafe manifest: %v", err)
	}
}
