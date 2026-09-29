package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDialogBodyLimitRejectsDecompressedOverflowAndCleansStorage(t *testing.T) {
	oldGlobal, oldBusiness := constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB
	constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = 2, 1
	t.Cleanup(func() {
		constant.MaxRequestBodyMB, constant.ResponsesRequestBodyLimitMB = oldGlobal, oldBusiness
	})
	for _, path := range []string{"/v1/chat/completions", "/v1/messages", "/v1/responses"} {
		for _, compressed := range []bool{false, true} {
			for _, size := range []int{1 << 20, 1<<20 + 1} {
				before := common.GetDiskCacheStats()
				payload := bytes.Repeat([]byte("x"), size)
				if compressed {
					var encoded bytes.Buffer
					writer := gzip.NewWriter(&encoded)
					_, err := writer.Write(payload)
					require.NoError(t, err)
					require.NoError(t, writer.Close())
					payload = encoded.Bytes()
				}
				r := gin.New()
				r.Use(DecompressRequestMiddleware(), BodyStorageCleanup())
				r.POST(path, func(c *gin.Context) {
					storage, err := common.GetBodyStorage(c)
					if common.IsRequestBodyTooLargeError(err) {
						abortWithRequestBodyTooLarge(c)
						return
					}
					require.NoError(t, err)
					n, err := io.Copy(io.Discard, storage)
					require.NoError(t, err)
					require.EqualValues(t, size, n)
					c.Status(http.StatusNoContent)
				})
				req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
				req.ContentLength = -1 // chunked/unknown length cannot bypass the cap
				if compressed {
					req.Header.Set("Content-Encoding", "gzip")
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				if size > 1<<20 {
					require.Equal(t, http.StatusRequestEntityTooLarge, w.Code, path)
				} else {
					require.Equal(t, http.StatusNoContent, w.Code, path)
				}
				after := common.GetDiskCacheStats()
				require.Equal(t, before.ActiveMemoryBuffers, after.ActiveMemoryBuffers)
				require.Equal(t, before.ActiveDiskFiles, after.ActiveDiskFiles)
			}
		}
	}
}
