package common

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type repeatingBodyReader struct{}

func (repeatingBodyReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'A'
	}
	return len(p), nil
}

func TestRequestBodyStorageBoundaryAndLengthHonesty(t *testing.T) {
	oldConfig := GetDiskCacheConfig()
	t.Cleanup(func() { SetDiskCacheConfig(oldConfig) })
	SetDiskCacheConfig(DiskCacheConfig{Enabled: false})
	for _, declared := range []int64{-1, 1, 1024} {
		before := GetDiskCacheStats()
		body := bytes.Repeat([]byte("A"), 1024)
		storage, err := CreateBodyStorageFromReader(bytes.NewReader(body), declared, 1024)
		require.NoError(t, err)
		got, err := io.ReadAll(storage)
		require.NoError(t, err)
		require.Equal(t, body, got)
		require.NoError(t, storage.Close())
		_, err = CreateBodyStorageFromReader(bytes.NewReader(append(body, 'A')), declared, 1024)
		require.ErrorIs(t, err, ErrRequestBodyTooLarge, "actual length, not Content-Length, controls acceptance")
		require.Equal(t, before.ActiveMemoryBuffers, GetDiskCacheStats().ActiveMemoryBuffers)
	}
}

// Opt-in to keep routine/race tests light. This is a byte-transport test, not a
// claim that any provider understands a video this large.
func TestRequestBody192MiBStorageIntegrity(t *testing.T) {
	if os.Getenv("NEW_API_LARGE_BODY_TEST") != "1" {
		t.Skip("set NEW_API_LARGE_BODY_TEST=1 for the real 192 MiB boundary")
	}
	oldConfig := GetDiskCacheConfig()
	t.Cleanup(func() { SetDiskCacheConfig(oldConfig) })
	SetDiskCacheConfig(DiskCacheConfig{Enabled: true, ThresholdMB: 1, MaxSizeMB: 512, Path: t.TempDir()})
	const limit int64 = 192 << 20
	before := GetDiskCacheStats()
	inputHash := sha256.New()
	storage, err := CreateBodyStorageFromReader(io.TeeReader(io.LimitReader(repeatingBodyReader{}, limit), inputHash), limit, limit)
	require.NoError(t, err)
	t.Cleanup(func() { _ = storage.Close() })
	require.True(t, storage.IsDisk())
	require.Equal(t, limit, storage.Size())
	for i := 0; i < 2; i++ {
		_, err = storage.Seek(0, io.SeekStart)
		require.NoError(t, err)
		outputHash := sha256.New()
		n, err := io.Copy(outputHash, ReaderOnly(storage))
		require.NoError(t, err)
		require.Equal(t, limit, n)
		require.Equal(t, inputHash.Sum(nil), outputHash.Sum(nil))
	}
	require.NoError(t, storage.Close())
	_, err = CreateBodyStorageFromReader(io.LimitReader(repeatingBodyReader{}, limit+1), limit+1, limit)
	require.ErrorIs(t, err, ErrRequestBodyTooLarge)
	require.Equal(t, before.ActiveDiskFiles, GetDiskCacheStats().ActiveDiskFiles)
	require.Equal(t, before.CurrentDiskUsageBytes, GetDiskCacheStats().CurrentDiskUsageBytes)
	files, err := os.ReadDir(GetDiskCacheDir())
	require.NoError(t, err)
	require.Empty(t, files)
}
