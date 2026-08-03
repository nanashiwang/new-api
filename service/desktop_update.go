package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	DesktopUpdateSettingsOptionKey       = "DesktopUpdateSettings"
	DesktopUpdateTokenOptionKey          = "DesktopUpdatePublishToken"
	latestManifestName                   = "latest.json"
	defaultDesktopUpdateDir              = "/data/desktop-updates"
	manifestMaxBytes               int64 = 1024 * 1024
	desktopUpdatePruneGracePeriod        = 24 * time.Hour
)

var (
	ErrDesktopUpdateDisabled  = errors.New("桌面更新服务未启用")
	ErrDesktopUpdateNotFound  = errors.New("更新文件不存在")
	ErrDesktopUpdateConflict  = errors.New("当前版本不能删除")
	ErrDesktopUpdateImmutable = errors.New("版本文件已存在且内容不同，请先删除未发布版本")

	desktopUpdateVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
	desktopUpdateFilePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+() -]{0,199}$`)
	desktopUpdateWriteMu        sync.Mutex
)

type DesktopUpdateSettings struct {
	Enabled        bool   `json:"enabled"`
	PublicBaseURL  string `json:"public_base_url"`
	MaxUploadMB    int    `json:"max_upload_mb"`
	RetentionCount int    `json:"retention_count"`
}

type DesktopUpdateFile struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	URL        string    `json:"url,omitempty"`
}

type DesktopUpdateRelease struct {
	Version    string              `json:"version"`
	Current    bool                `json:"current"`
	TotalSize  int64               `json:"total_size"`
	ModifiedAt time.Time           `json:"modified_at"`
	Files      []DesktopUpdateFile `json:"files"`
}

type DesktopUpdateManifestSummary struct {
	Version   string   `json:"version"`
	Notes     string   `json:"notes,omitempty"`
	PubDate   string   `json:"pub_date,omitempty"`
	Platforms []string `json:"platforms"`
}

type DesktopDownloadPackage struct {
	ID       string `json:"id"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Format   string `json:"format"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

type DesktopDownloadCatalog struct {
	Version  string                   `json:"version"`
	Notes    string                   `json:"notes,omitempty"`
	PubDate  string                   `json:"pub_date,omitempty"`
	Packages []DesktopDownloadPackage `json:"packages"`
}

type desktopDownloadDefinition struct {
	ID       string
	OS       string
	Arch     string
	Format   string
	Suffixes []string
}

var desktopDownloadDefinitions = []desktopDownloadDefinition{
	{ID: "macos-arm64", OS: "macos", Arch: "arm64", Format: "dmg", Suffixes: []string{"_aarch64.dmg", "_arm64.dmg"}},
	{ID: "macos-x64", OS: "macos", Arch: "x86_64", Format: "dmg", Suffixes: []string{"_x64.dmg", "_x86_64.dmg"}},
	{ID: "windows-x64", OS: "windows", Arch: "x86_64", Format: "exe", Suffixes: []string{"_x64-setup.exe", "_x86_64-setup.exe"}},
}

type DesktopUpdateStorageStatus struct {
	Directory string `json:"directory"`
	Exists    bool   `json:"exists"`
	Writable  bool   `json:"writable"`
	Error     string `json:"error,omitempty"`
}

func defaultDesktopUpdateSettings() DesktopUpdateSettings {
	return DesktopUpdateSettings{
		Enabled:        common.GetEnvOrDefaultBool("DESKTOP_UPDATE_ENABLED", false),
		PublicBaseURL:  strings.TrimRight(strings.TrimSpace(os.Getenv("DESKTOP_UPDATE_PUBLIC_BASE_URL")), "/"),
		MaxUploadMB:    common.GetEnvOrDefault("DESKTOP_UPDATE_MAX_UPLOAD_MB", 256),
		RetentionCount: common.GetEnvOrDefault("DESKTOP_UPDATE_RETENTION_COUNT", 10),
	}
}

func GetDesktopUpdateSettings() DesktopUpdateSettings {
	settings := defaultDesktopUpdateSettings()
	common.OptionMapRWMutex.RLock()
	raw, ok := common.OptionMap[DesktopUpdateSettingsOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if ok {
		var persisted DesktopUpdateSettings
		if err := common.UnmarshalJsonStr(common.Interface2String(raw), &persisted); err == nil {
			settings = persisted
		} else {
			// Invalid persisted configuration must fail closed.
			settings.Enabled = false
		}
	}
	settings.PublicBaseURL = strings.TrimRight(strings.TrimSpace(settings.PublicBaseURL), "/")
	if settings.MaxUploadMB < 1 || settings.MaxUploadMB > 4096 {
		settings.MaxUploadMB = 256
	}
	if settings.RetentionCount < 0 || settings.RetentionCount > 100 {
		settings.RetentionCount = 10
	}
	if _, err := ValidateDesktopUpdateSettings(settings); err != nil {
		// Environment/database edits bypass the API validator, so public serving
		// must still fail closed when the effective configuration is invalid.
		settings.Enabled = false
	}
	return settings
}

func ValidateDesktopUpdateSettings(settings DesktopUpdateSettings) (DesktopUpdateSettings, error) {
	settings.PublicBaseURL = strings.TrimRight(strings.TrimSpace(settings.PublicBaseURL), "/")
	if settings.MaxUploadMB < 1 || settings.MaxUploadMB > 4096 {
		return settings, errors.New("单文件大小限制必须在 1 到 4096 MB 之间")
	}
	if settings.RetentionCount < 0 || settings.RetentionCount > 100 {
		return settings, errors.New("保留版本数量必须在 0 到 100 之间")
	}
	if settings.PublicBaseURL != "" {
		if err := validateDesktopUpdatePublicBaseURL(settings.PublicBaseURL); err != nil {
			return settings, err
		}
	}
	if settings.Enabled && settings.PublicBaseURL == "" {
		return settings, errors.New("启用桌面更新服务前必须配置对外基础地址")
	}
	return settings, nil
}

func validateDesktopUpdatePublicBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.Opaque != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("对外基础地址必须是不含凭据、查询参数或片段的 HTTP(S) 地址")
	}
	return nil
}

func DesktopUpdateStorageDir() string {
	if configured := strings.TrimSpace(os.Getenv("DESKTOP_UPDATE_DIR")); configured != "" {
		return filepath.Clean(configured)
	}
	return defaultDesktopUpdateDir
}

func EnsureDesktopUpdateStorage() error {
	root := DesktopUpdateStorageDir()
	if err := os.MkdirAll(filepath.Join(root, "releases"), 0o755); err != nil {
		return fmt.Errorf("创建桌面更新存储目录失败: %w", err)
	}
	return nil
}

func GetDesktopUpdateStorageStatus() DesktopUpdateStorageStatus {
	root := DesktopUpdateStorageDir()
	status := DesktopUpdateStorageStatus{Directory: root}
	info, err := os.Stat(root)
	if err != nil {
		if !os.IsNotExist(err) {
			status.Error = err.Error()
		}
		return status
	}
	status.Exists = info.IsDir()
	if !status.Exists {
		status.Error = "存储路径不是目录"
		return status
	}
	temp, err := os.CreateTemp(root, ".write-check-")
	if err != nil {
		status.Error = err.Error()
		return status
	}
	name := temp.Name()
	if closeErr := temp.Close(); closeErr != nil {
		status.Error = closeErr.Error()
		_ = os.Remove(name)
		return status
	}
	if err = os.Remove(name); err != nil {
		status.Error = err.Error()
		return status
	}
	status.Writable = true
	return status
}

func CanonicalDesktopUpdateVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	if len(version) > 128 || !desktopUpdateVersionPattern.MatchString(version) {
		return "", errors.New("版本号必须是有效的 SemVer，例如 0.1.17")
	}
	prerelease := strings.SplitN(strings.SplitN(version, "+", 2)[0], "-", 2)
	if len(prerelease) == 2 {
		for _, identifier := range strings.Split(prerelease[1], ".") {
			if len(identifier) > 1 && identifier[0] == '0' && strings.Trim(identifier, "0123456789") == "" {
				return "", errors.New("版本号必须是有效的 SemVer，例如 0.1.17")
			}
		}
	}
	return version, nil
}

func ValidateDesktopUpdateFilename(filename string) error {
	if filename == "" || filename != strings.TrimSpace(filename) || filename != filepath.Base(filename) || filename == "." || filename == ".." || strings.Contains(filename, "..") || !desktopUpdateFilePattern.MatchString(filename) {
		return errors.New("文件名不合法")
	}
	lower := strings.ToLower(filename)
	if strings.HasSuffix(lower, ".sig") {
		lower = strings.TrimSuffix(lower, ".sig")
	}
	allowed := []string{".tar.gz", ".appimage", ".msi", ".exe", ".dmg", ".zip", ".deb", ".rpm"}
	for _, suffix := range allowed {
		if strings.HasSuffix(lower, suffix) {
			return nil
		}
	}
	return errors.New("不支持的更新文件类型")
}

func desktopUpdateReleaseDir(version string) (string, string, error) {
	canonical, err := CanonicalDesktopUpdateVersion(version)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(DesktopUpdateStorageDir(), "releases", canonical), canonical, nil
}

func ensureDesktopUpdateReleaseDir(version string) (string, string, error) {
	directory, canonical, err := desktopUpdateReleaseDir(version)
	if err != nil {
		return "", "", err
	}
	if err = EnsureDesktopUpdateStorage(); err != nil {
		return "", "", err
	}
	if info, statErr := os.Lstat(directory); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return "", "", errors.New("版本存储路径不是安全目录")
		}
	} else if !os.IsNotExist(statErr) {
		return "", "", statErr
	} else if err = os.Mkdir(directory, 0o755); err != nil && !os.IsExist(err) {
		return "", "", err
	}
	return directory, canonical, nil
}

func copyDesktopUpdateFileAtomic(destination string, reader io.Reader, maxBytes int64, allowReplace bool) (int64, error) {
	if maxBytes < 1 {
		return 0, errors.New("上传大小限制无效")
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".upload-")
	if err != nil {
		return 0, err
	}
	tempName := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempName)
		}
	}()

	written, err := io.Copy(temp, io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return 0, err
	}
	if written > maxBytes {
		return 0, fmt.Errorf("文件超过 %d MB 限制", maxBytes/(1024*1024))
	}
	if written == 0 {
		return 0, errors.New("更新文件不能为空")
	}
	if err = temp.Sync(); err != nil {
		return 0, err
	}
	if err = temp.Chmod(0o644); err != nil {
		return 0, err
	}
	if err = temp.Close(); err != nil {
		return 0, err
	}
	if !allowReplace {
		for attempts := 0; attempts < 2; attempts++ {
			if existingInfo, statErr := os.Lstat(destination); statErr == nil {
				if existingInfo.Mode()&os.ModeSymlink != 0 || !existingInfo.Mode().IsRegular() {
					return 0, ErrDesktopUpdateImmutable
				}
				identical, compareErr := desktopUpdateFilesEqual(destination, tempName)
				if compareErr != nil {
					return 0, compareErr
				}
				if !identical {
					return 0, ErrDesktopUpdateImmutable
				}
				return written, nil
			} else if !os.IsNotExist(statErr) {
				return 0, statErr
			}
			// Hard-linking a completed temp file provides create-if-absent
			// semantics across multiple NewAPI processes sharing the volume.
			if linkErr := os.Link(tempName, destination); linkErr == nil {
				return written, nil
			} else if !os.IsExist(linkErr) {
				return 0, linkErr
			}
		}
		return 0, ErrDesktopUpdateImmutable
	}
	if err = os.Rename(tempName, destination); err != nil {
		return 0, err
	}
	keep = true
	return written, nil
}

func desktopUpdateFilesEqual(left, right string) (bool, error) {
	leftInfo, err := os.Stat(left)
	if err != nil {
		return false, err
	}
	rightInfo, err := os.Stat(right)
	if err != nil {
		return false, err
	}
	if leftInfo.Size() != rightInfo.Size() {
		return false, nil
	}
	hashFile := func(filename string) ([sha256.Size]byte, error) {
		file, openErr := os.Open(filename)
		if openErr != nil {
			return [sha256.Size]byte{}, openErr
		}
		defer file.Close()
		hasher := sha256.New()
		if _, copyErr := io.Copy(hasher, file); copyErr != nil {
			return [sha256.Size]byte{}, copyErr
		}
		var result [sha256.Size]byte
		copy(result[:], hasher.Sum(nil))
		return result, nil
	}
	leftHash, err := hashFile(left)
	if err != nil {
		return false, err
	}
	rightHash, err := hashFile(right)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(leftHash[:], rightHash[:]) == 1, nil
}

func SaveDesktopUpdateArtifact(version, filename string, reader io.Reader, maxBytes int64) (DesktopUpdateFile, error) {
	if err := ValidateDesktopUpdateFilename(filename); err != nil {
		return DesktopUpdateFile{}, err
	}
	desktopUpdateWriteMu.Lock()
	defer desktopUpdateWriteMu.Unlock()

	directory, canonical, err := ensureDesktopUpdateReleaseDir(version)
	if err != nil {
		return DesktopUpdateFile{}, err
	}
	destination := filepath.Join(directory, filename)
	if _, err = copyDesktopUpdateFileAtomic(destination, reader, maxBytes, false); err != nil {
		return DesktopUpdateFile{}, err
	}
	info, err := os.Stat(destination)
	if err != nil {
		return DesktopUpdateFile{}, err
	}
	return DesktopUpdateFile{
		Name:       filename,
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
		URL:        desktopUpdateArtifactURL(GetDesktopUpdateSettings().PublicBaseURL, canonical, filename),
	}, nil
}

func readDesktopUpdateManifest(reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, manifestMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > manifestMaxBytes {
		return nil, errors.New("latest.json 不能超过 1 MB")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, errors.New("latest.json 不能为空")
	}
	return data, nil
}

func desktopUpdateArtifactURL(baseURL, version, filename string) string {
	if baseURL == "" {
		return ""
	}
	return strings.TrimRight(baseURL, "/") + "/releases/" + url.PathEscape(version) + "/" + url.PathEscape(filename)
}

func parseAndRewriteDesktopUpdateManifest(data []byte, baseURL string) ([]byte, DesktopUpdateManifestSummary, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, DesktopUpdateManifestSummary{}, errors.New("请先配置对外基础地址")
	}
	if err := validateDesktopUpdatePublicBaseURL(baseURL); err != nil {
		return nil, DesktopUpdateManifestSummary{}, err
	}
	var manifest map[string]any
	if err := common.Unmarshal(data, &manifest); err != nil {
		return nil, DesktopUpdateManifestSummary{}, errors.New("latest.json 不是有效的 JSON")
	}
	versionValue, ok := manifest["version"].(string)
	if !ok {
		return nil, DesktopUpdateManifestSummary{}, errors.New("latest.json 缺少 version")
	}
	version, err := CanonicalDesktopUpdateVersion(versionValue)
	if err != nil {
		return nil, DesktopUpdateManifestSummary{}, err
	}
	platforms, ok := manifest["platforms"].(map[string]any)
	if !ok || len(platforms) == 0 {
		return nil, DesktopUpdateManifestSummary{}, errors.New("latest.json 缺少 platforms")
	}

	directory, _, err := desktopUpdateReleaseDir(version)
	if err != nil {
		return nil, DesktopUpdateManifestSummary{}, err
	}
	platformNames := make([]string, 0, len(platforms))
	for platformName, rawPlatform := range platforms {
		platform, ok := rawPlatform.(map[string]any)
		if !ok {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 配置不合法", platformName)
		}
		rawURL, ok := platform["url"].(string)
		if !ok || strings.TrimSpace(rawURL) == "" {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 缺少下载地址", platformName)
		}
		signature, ok := platform["signature"].(string)
		if !ok || strings.TrimSpace(signature) == "" {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 缺少更新签名", platformName)
		}
		parsedURL, parseErr := url.Parse(rawURL)
		if parseErr != nil {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 下载地址不合法", platformName)
		}
		filename := path.Base(parsedURL.Path)
		if err = ValidateDesktopUpdateFilename(filename); err != nil {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s: %w", platformName, err)
		}
		artifactPath := filepath.Join(directory, filename)
		info, statErr := os.Lstat(artifactPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 引用的文件 %s 尚未上传", platformName, filename)
			}
			return nil, DesktopUpdateManifestSummary{}, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 引用的文件不安全", platformName)
		}
		if info.Size() == 0 {
			return nil, DesktopUpdateManifestSummary{}, fmt.Errorf("平台 %s 引用的文件不能为空", platformName)
		}
		platform["url"] = desktopUpdateArtifactURL(baseURL, version, filename)
		platformNames = append(platformNames, platformName)
	}
	sort.Strings(platformNames)
	manifest["version"] = version
	rewritten, err := common.Marshal(manifest)
	if err != nil {
		return nil, DesktopUpdateManifestSummary{}, err
	}
	summary := DesktopUpdateManifestSummary{
		Version:   version,
		Notes:     stringValue(manifest["notes"]),
		PubDate:   stringValue(manifest["pub_date"]),
		Platforms: platformNames,
	}
	return rewritten, summary, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func PublishDesktopUpdateManifest(reader io.Reader, baseURL string, retentionCount int) (DesktopUpdateManifestSummary, error) {
	data, err := readDesktopUpdateManifest(reader)
	if err != nil {
		return DesktopUpdateManifestSummary{}, err
	}
	desktopUpdateWriteMu.Lock()
	defer desktopUpdateWriteMu.Unlock()

	if err = EnsureDesktopUpdateStorage(); err != nil {
		return DesktopUpdateManifestSummary{}, err
	}
	rewritten, summary, err := parseAndRewriteDesktopUpdateManifest(data, baseURL)
	if err != nil {
		return DesktopUpdateManifestSummary{}, err
	}
	if _, err = copyDesktopUpdateFileAtomic(filepath.Join(DesktopUpdateStorageDir(), latestManifestName), strings.NewReader(string(rewritten)), manifestMaxBytes, true); err != nil {
		return DesktopUpdateManifestSummary{}, err
	}
	if retentionCount > 0 {
		if err = pruneDesktopUpdateReleasesLocked(summary.Version, retentionCount); err != nil {
			common.SysError("failed to prune desktop update releases: " + err.Error())
		}
	}
	return summary, nil
}

func readCurrentDesktopUpdateManifest() ([]byte, error) {
	filename := filepath.Join(DesktopUpdateStorageDir(), latestManifestName)
	info, err := os.Lstat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDesktopUpdateNotFound
		}
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, errors.New("latest.json 不是安全的常规文件")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readDesktopUpdateManifest(file)
}

func currentDesktopUpdateVersion() (string, error) {
	data, err := readCurrentDesktopUpdateManifest()
	if err != nil {
		return "", err
	}
	var manifest map[string]any
	if err = common.Unmarshal(data, &manifest); err != nil {
		return "", err
	}
	return CanonicalDesktopUpdateVersion(stringValue(manifest["version"]))
}

func GetDesktopUpdateManifestSummary() (*DesktopUpdateManifestSummary, error) {
	data, err := readCurrentDesktopUpdateManifest()
	if err != nil {
		return nil, err
	}
	_, summary, err := parseAndRewriteDesktopUpdateManifest(data, GetDesktopUpdateSettings().PublicBaseURL)
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func RepublishCurrentDesktopUpdateManifest(baseURL string, retentionCount int) error {
	data, err := readCurrentDesktopUpdateManifest()
	if err != nil {
		return err
	}
	_, err = PublishDesktopUpdateManifest(strings.NewReader(string(data)), baseURL, retentionCount)
	return err
}

func GetDesktopDownloadCatalog(baseURL string) (*DesktopDownloadCatalog, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("请先配置对外基础地址")
	}
	if err := validateDesktopUpdatePublicBaseURL(baseURL); err != nil {
		return nil, err
	}
	manifestData, err := readCurrentDesktopUpdateManifest()
	if err != nil {
		return nil, err
	}
	_, manifest, err := parseAndRewriteDesktopUpdateManifest(manifestData, baseURL)
	if err != nil {
		return nil, err
	}
	directory, version, err := desktopUpdateReleaseDir(manifest.Version)
	if err != nil {
		return nil, err
	}
	directoryInfo, err := os.Lstat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDesktopUpdateNotFound
		}
		return nil, err
	}
	if directoryInfo.Mode()&os.ModeSymlink != 0 || !directoryInfo.IsDir() {
		return nil, ErrDesktopUpdateNotFound
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	packagesByID := make(map[string]DesktopDownloadPackage, len(desktopDownloadDefinitions))
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		definition, ok := classifyDesktopDownload(entry.Name())
		if !ok {
			continue
		}
		if err = ValidateDesktopUpdateFilename(entry.Name()); err != nil {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, infoErr
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 {
			continue
		}
		if _, exists := packagesByID[definition.ID]; exists {
			return nil, fmt.Errorf("当前版本包含重复的 %s 安装包", definition.ID)
		}
		packagesByID[definition.ID] = DesktopDownloadPackage{
			ID:       definition.ID,
			OS:       definition.OS,
			Arch:     definition.Arch,
			Format:   definition.Format,
			Filename: entry.Name(),
			Size:     info.Size(),
			URL:      desktopUpdateArtifactURL(baseURL, version, entry.Name()),
		}
	}

	packages := make([]DesktopDownloadPackage, 0, len(packagesByID))
	for _, definition := range desktopDownloadDefinitions {
		if item, ok := packagesByID[definition.ID]; ok {
			packages = append(packages, item)
		}
	}
	if len(packages) == 0 {
		return nil, ErrDesktopUpdateNotFound
	}
	return &DesktopDownloadCatalog{
		Version:  version,
		Notes:    manifest.Notes,
		PubDate:  manifest.PubDate,
		Packages: packages,
	}, nil
}

func classifyDesktopDownload(filename string) (desktopDownloadDefinition, bool) {
	lower := strings.ToLower(filename)
	for _, definition := range desktopDownloadDefinitions {
		for _, suffix := range definition.Suffixes {
			if strings.HasSuffix(lower, suffix) {
				return definition, true
			}
		}
	}
	return desktopDownloadDefinition{}, false
}

func OpenDesktopUpdateManifest() (*os.File, os.FileInfo, error) {
	return openDesktopUpdateRegularFile(filepath.Join(DesktopUpdateStorageDir(), latestManifestName))
}

func OpenDesktopUpdateArtifact(version, filename string) (*os.File, os.FileInfo, error) {
	if err := ValidateDesktopUpdateFilename(filename); err != nil {
		return nil, nil, ErrDesktopUpdateNotFound
	}
	directory, _, err := desktopUpdateReleaseDir(version)
	if err != nil {
		return nil, nil, ErrDesktopUpdateNotFound
	}
	return openDesktopUpdateRegularFile(filepath.Join(directory, filename))
}

func openDesktopUpdateRegularFile(filename string) (*os.File, os.FileInfo, error) {
	info, err := os.Lstat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, ErrDesktopUpdateNotFound
		}
		return nil, nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, nil, ErrDesktopUpdateNotFound
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	return file, info, nil
}

func ListDesktopUpdateReleases(baseURL string) ([]DesktopUpdateRelease, error) {
	currentVersion, err := currentDesktopUpdateVersion()
	if err != nil && !errors.Is(err, ErrDesktopUpdateNotFound) {
		currentVersion = ""
	}
	return listDesktopUpdateReleases(baseURL, currentVersion)
}

func listDesktopUpdateReleases(baseURL, currentVersion string) ([]DesktopUpdateRelease, error) {
	releasesRoot := filepath.Join(DesktopUpdateStorageDir(), "releases")
	entries, err := os.ReadDir(releasesRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []DesktopUpdateRelease{}, nil
		}
		return nil, err
	}
	releases := make([]DesktopUpdateRelease, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		version, versionErr := CanonicalDesktopUpdateVersion(entry.Name())
		if versionErr != nil || version != entry.Name() {
			continue
		}
		files, readErr := os.ReadDir(filepath.Join(releasesRoot, entry.Name()))
		if readErr != nil {
			return nil, readErr
		}
		release := DesktopUpdateRelease{Version: version, Current: version == currentVersion}
		for _, fileEntry := range files {
			if fileEntry.IsDir() || fileEntry.Type()&os.ModeSymlink != 0 || strings.HasPrefix(fileEntry.Name(), ".") {
				continue
			}
			if err = ValidateDesktopUpdateFilename(fileEntry.Name()); err != nil {
				continue
			}
			info, infoErr := fileEntry.Info()
			if infoErr != nil || !info.Mode().IsRegular() {
				continue
			}
			file := DesktopUpdateFile{
				Name:       fileEntry.Name(),
				Size:       info.Size(),
				ModifiedAt: info.ModTime(),
				URL:        desktopUpdateArtifactURL(baseURL, version, fileEntry.Name()),
			}
			release.Files = append(release.Files, file)
			release.TotalSize += info.Size()
			if info.ModTime().After(release.ModifiedAt) {
				release.ModifiedAt = info.ModTime()
			}
		}
		sort.Slice(release.Files, func(i, j int) bool { return release.Files[i].Name < release.Files[j].Name })
		releases = append(releases, release)
	}
	sort.Slice(releases, func(i, j int) bool {
		if releases[i].Current != releases[j].Current {
			return releases[i].Current
		}
		return releases[i].ModifiedAt.After(releases[j].ModifiedAt)
	})
	return releases, nil
}

func DeleteDesktopUpdateRelease(version string) error {
	desktopUpdateWriteMu.Lock()
	defer desktopUpdateWriteMu.Unlock()

	canonical, err := CanonicalDesktopUpdateVersion(version)
	if err != nil {
		return err
	}
	currentVersion, versionErr := currentDesktopUpdateVersion()
	if versionErr != nil && !errors.Is(versionErr, ErrDesktopUpdateNotFound) {
		return fmt.Errorf("读取当前版本失败，已拒绝删除: %w", versionErr)
	}
	if currentVersion == canonical {
		return ErrDesktopUpdateConflict
	}
	directory, _, err := desktopUpdateReleaseDir(canonical)
	if err != nil {
		return err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrDesktopUpdateNotFound
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return ErrDesktopUpdateNotFound
	}
	return os.RemoveAll(directory)
}

func pruneDesktopUpdateReleasesLocked(currentVersion string, retentionCount int) error {
	releases, err := listDesktopUpdateReleases("", currentVersion)
	if err != nil {
		return err
	}
	var currentModifiedAt time.Time
	for _, release := range releases {
		if release.Current {
			currentModifiedAt = release.ModifiedAt
			break
		}
	}
	kept := 0
	now := time.Now()
	for _, release := range releases {
		if release.Current {
			continue
		}
		// Never prune a staged release uploaded after the active release, and
		// leave a grace window for clients that fetched the previous manifest.
		if (!currentModifiedAt.IsZero() && !release.ModifiedAt.Before(currentModifiedAt)) || now.Sub(release.ModifiedAt) < desktopUpdatePruneGracePeriod {
			continue
		}
		kept++
		if kept < retentionCount {
			continue
		}
		directory, _, pathErr := desktopUpdateReleaseDir(release.Version)
		if pathErr != nil {
			return pathErr
		}
		if removeErr := os.RemoveAll(directory); removeErr != nil {
			return removeErr
		}
	}
	return nil
}

func GenerateDesktopUpdatePublishToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, HashDesktopUpdatePublishToken(token), nil
}

func HashDesktopUpdatePublishToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func DesktopUpdatePublishTokenStatus() (bool, string) {
	if stored, exists := storedDesktopUpdatePublishTokenHash(); exists {
		decoded, err := hex.DecodeString(stored)
		return err == nil && len(decoded) == sha256.Size, "database"
	}
	if strings.TrimSpace(os.Getenv("DESKTOP_UPDATE_PUBLISH_TOKEN")) != "" {
		return true, "environment"
	}
	return false, ""
}

func ValidateDesktopUpdatePublishToken(token string) bool {
	if token == "" {
		return false
	}
	candidate := sha256.Sum256([]byte(token))
	if stored, exists := storedDesktopUpdatePublishTokenHash(); exists {
		expected, err := hex.DecodeString(stored)
		if err != nil || len(expected) != sha256.Size {
			return false
		}
		return subtle.ConstantTimeCompare(candidate[:], expected) == 1
	}
	environmentToken := strings.TrimSpace(os.Getenv("DESKTOP_UPDATE_PUBLISH_TOKEN"))
	if environmentToken == "" {
		return false
	}
	expected := sha256.Sum256([]byte(environmentToken))
	return subtle.ConstantTimeCompare(candidate[:], expected[:]) == 1
}

func storedDesktopUpdatePublishTokenHash() (string, bool) {
	common.OptionMapRWMutex.RLock()
	value, ok := common.OptionMap[DesktopUpdateTokenOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if !ok {
		return "", false
	}
	stored := strings.TrimSpace(common.Interface2String(value))
	return stored, stored != ""
}
