package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type syntheticBodyReader struct{}

func (syntheticBodyReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'A'
	}
	return len(p), nil
}

// Real HTTP transport at the configured byte boundary. The payload is a
// synthetic body, not a video/model request; semantic relay is tested elsewhere.
func TestDialog192MiBHTTPByteIntegrity(t *testing.T) {
	if os.Getenv("NEW_API_LARGE_BODY_TEST") != "1" {
		t.Skip("set NEW_API_LARGE_BODY_TEST=1 for large HTTP body validation")
	}
	oldGlobal, oldBusiness, oldDisk := constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB, common.GetDiskCacheConfig()
	constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = 256, 192
	common.SetDiskCacheConfig(common.DiskCacheConfig{Enabled: true, ThresholdMB: 1, MaxSizeMB: 512, Path: t.TempDir()})
	t.Cleanup(func() {
		constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = oldGlobal, oldBusiness
		common.SetDiskCacheConfig(oldDisk)
	})
	const limit int64 = 192 << 20
	client := &http.Client{Timeout: 20 * time.Second}
	defer client.CloseIdleConnections()
	var upstreamCalls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		hash := sha256.New()
		n, err := io.Copy(hash, r.Body)
		if err != nil || n != limit {
			http.Error(w, "incomplete request body", 400)
			return
		}
		_, _ = io.WriteString(w, hex.EncodeToString(hash.Sum(nil)))
	}))
	defer upstream.Close()
	r := gin.New()
	r.Use(DecompressRequestMiddleware(), BodyStorageCleanup())
	for _, path := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
		r.POST(path, func(c *gin.Context) {
			storage, err := common.GetBodyStorage(c)
			if common.IsRequestBodyTooLargeError(err) {
				abortWithRequestBodyTooLarge(c)
				return
			}
			if err != nil {
				http.Error(c.Writer, "storage error", 500)
				return
			}
			req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstream.URL, common.ReaderOnly(storage))
			if err != nil {
				http.Error(c.Writer, "request error", 500)
				return
			}
			req.ContentLength = storage.Size()
			resp, err := client.Do(req)
			if err != nil {
				http.Error(c.Writer, "upstream error", 502)
				return
			}
			defer resp.Body.Close()
			c.Status(resp.StatusCode)
			_, _ = io.Copy(c.Writer, resp.Body)
		})
	}
	server := httptest.NewServer(r)
	defer server.Close()
	baseline := common.GetDiskCacheStats()
	for _, path := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
		for _, length := range []int64{limit, limit + 1} {
			calls := upstreamCalls.Load()
			hash := sha256.New()
			req, err := http.NewRequest(http.MethodPost, server.URL+path, io.TeeReader(io.LimitReader(syntheticBodyReader{}, length), hash))
			require.NoError(t, err)
			req.ContentLength = length
			resp, err := client.Do(req)
			require.NoError(t, err)
			result, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			if length == limit {
				require.Equal(t, 200, resp.StatusCode)
				require.Equal(t, hex.EncodeToString(hash.Sum(nil)), string(result))
				require.Equal(t, calls+1, upstreamCalls.Load())
			} else {
				require.Equal(t, 413, resp.StatusCode)
				require.Equal(t, calls, upstreamCalls.Load(), "oversize requests must not reach upstream")
			}
			require.Eventually(t, func() bool {
				return common.GetDiskCacheStats().ActiveDiskFiles == baseline.ActiveDiskFiles
			}, time.Second, time.Millisecond)
		}
	}
}
