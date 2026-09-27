package controller

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http/httptest"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupInvoiceAttachmentTest(t *testing.T) *model.InvoiceRequest {
	t.Helper()
	t.Chdir(t.TempDir())
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	originalDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = originalDB
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.InvoiceRequest{}, &model.InvoiceRequestItem{}, &model.InvoiceRequestProductItem{}))
	request := &model.InvoiceRequest{
		UserId: 1, Title: "测试发票", Email: "recipient@example.test", Status: model.InvoiceStatusPending,
		InvoiceSentTo: "recipient@example.test", ServiceFeeQuota: 500, TotalMoney: 100,
	}
	require.NoError(t, db.Create(request).Error)
	return request
}

func invoiceAttachmentContext(t *testing.T, id int, fields, files map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	for key, value := range files {
		part, err := writer.CreateFormFile(key, key+".pdf")
		require.NoError(t, err)
		_, err = io.WriteString(part, value)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/", body)
	context.Request.Header.Set("Content-Type", writer.FormDataContentType())
	context.Params = gin.Params{{Key: "id", Value: fmt.Sprint(id)}}
	context.Set("id", 1)
	t.Cleanup(func() {
		if context.Request.MultipartForm != nil {
			_ = context.Request.MultipartForm.RemoveAll()
		}
	})
	return context, recorder
}

func invoiceAttachmentResponse(t *testing.T, recorder *httptest.ResponseRecorder) (bool, string, model.InvoiceRequest) {
	t.Helper()
	var response struct {
		Success bool                 `json:"success"`
		Message string               `json:"message"`
		Data    model.InvoiceRequest `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response.Success, response.Message, response.Data
}

// Only listens on loopback and captures MIME messages; no real emails are sent.
func captureInvoiceSMTP(t *testing.T) <-chan []byte {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	originalServer, originalPort, originalFrom, originalSSL := common.SMTPServer, common.SMTPPort, common.SMTPFrom, common.SMTPSSLEnabled
	common.SMTPServer, common.SMTPPort = "127.0.0.1", listener.Addr().(*net.TCPAddr).Port
	common.SMTPFrom, common.SMTPSSLEnabled = "sender@example.test", false
	finished := make(chan struct{})
	messages := make(chan []byte, 16)
	go func() {
		defer close(finished)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			protocol := textproto.NewConn(conn)
			_ = protocol.PrintfLine("220 invoice test SMTP")
			for {
				line, err := protocol.ReadLine()
				if err != nil {
					break
				}
				if strings.HasPrefix(line, "DATA") {
					_ = protocol.PrintfLine("354 send message")
					message, err := protocol.ReadDotBytes()
					if err != nil {
						break
					}
					messages <- message
				}
				if line == "QUIT" {
					_ = protocol.PrintfLine("221 goodbye")
					break
				}
				_ = protocol.PrintfLine("250 OK")
			}
			_ = protocol.Close()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		<-finished
		common.SMTPServer, common.SMTPPort, common.SMTPFrom, common.SMTPSSLEnabled = originalServer, originalPort, originalFrom, originalSSL
	})
	return messages
}

func invoiceMIMEAttachments(t *testing.T, messages <-chan []byte) map[string]string {
	t.Helper()
	var raw []byte
	select {
	case raw = <-messages:
	case <-time.After(5 * time.Second):
		t.Fatal("no captured email")
	}
	message, err := mail.ReadMessage(bytes.NewReader(raw))
	require.NoError(t, err)
	_, params, err := mime.ParseMediaType(message.Header.Get("Content-Type"))
	require.NoError(t, err)
	reader := multipart.NewReader(message.Body, params["boundary"])
	attachments := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if filename := part.FileName(); filename != "" {
			data, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, part))
			require.NoError(t, err)
			attachments[filename] = string(data)
		}
	}
	return attachments
}

func TestInvoiceAttachmentsApproveAndResendReuseExactPDFs(t *testing.T) {
	request := setupInvoiceAttachmentTest(t)
	messages := captureInvoiceSMTP(t)
	fields := map[string]string{"send_email": "true", "send_detail_bill": "true", "send_service_confirmation": "true"}
	files := map[string]string{"invoice_file": "%PDF-1.4 original invoice", "detail_bill_file": "%PDF-1.4 original detail", "service_confirmation_file": "%PDF-1.4 original products"}
	context, recorder := invoiceAttachmentContext(t, request.Id, fields, files)
	ApproveInvoiceRequest(context)
	success, message, response := invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Equal(t, model.InvoiceSendStatusSent, response.InvoiceSendStatus)
	want := map[string]string{"invoice_file.pdf": files["invoice_file"], "detail_bill_file.pdf": files["detail_bill_file"], "service_confirmation_file.pdf": files["service_confirmation_file"]}
	require.Equal(t, want, invoiceMIMEAttachments(t, messages))
	stored, err := model.GetInvoiceRequestDetail(request.Id)
	require.NoError(t, err)
	require.FileExists(t, stored.DetailBillFilePath)
	require.FileExists(t, stored.ServiceConfirmationFilePath)
	require.NotContains(t, recorder.Body.String(), stored.DetailBillFilePath)
	require.NotContains(t, recorder.Body.String(), stored.ServiceConfirmationFilePath)

	// No uploads on resend. Verify actual MIME bytes, not just response metadata.
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, nil)
	ResendInvoiceEmail(context)
	success, message, response = invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Equal(t, model.InvoiceSendStatusSent, response.InvoiceSendStatus)
	require.Equal(t, want, invoiceMIMEAttachments(t, messages))
	require.Equal(t, stored.ReviewedAt, response.ReviewedAt)
	require.Equal(t, stored.ServiceFeeQuota, response.ServiceFeeQuota)

	// Selecting neither attachment sends only the invoice and keeps both saved.
	context, recorder = invoiceAttachmentContext(t, request.Id, nil, nil)
	ResendInvoiceEmail(context)
	success, message, response = invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Len(t, invoiceMIMEAttachments(t, messages), 1)
	require.Equal(t, stored.DetailBillFileName, response.DetailBillFileName)
	require.Equal(t, stored.ServiceConfirmationFileName, response.ServiceConfirmationFileName)

	// A new upload replaces only that attachment, even when SMTP fails.
	common.SMTPFrom = "invalid"
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, map[string]string{"detail_bill_file": "%PDF-1.4 replacement detail"})
	ResendInvoiceEmail(context)
	success, message, response = invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Equal(t, model.InvoiceSendStatusFailed, response.InvoiceSendStatus)
	updated, err := model.GetInvoiceRequestDetail(request.Id)
	require.NoError(t, err)
	require.NotEqual(t, stored.DetailBillFilePath, updated.DetailBillFilePath)
	require.Equal(t, stored.ServiceConfirmationFilePath, updated.ServiceConfirmationFilePath)
	require.FileExists(t, updated.DetailBillFilePath)
	common.SMTPFrom = "sender@example.test"
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, nil)
	ResendInvoiceEmail(context)
	success, message, response = invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Equal(t, model.InvoiceSendStatusSent, response.InvoiceSendStatus)
	want["detail_bill_file.pdf"] = "%PDF-1.4 replacement detail"
	require.Equal(t, want, invoiceMIMEAttachments(t, messages))

	// Missing saved files must not silently send an incomplete email.
	require.NoError(t, os.Remove(updated.ServiceConfirmationFilePath))
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, nil)
	ResendInvoiceEmail(context)
	success, message, _ = invoiceAttachmentResponse(t, recorder)
	require.False(t, success)
	require.Contains(t, message, "产品明细清单 PDF 无法读取")
	require.Empty(t, messages)
}

func TestInvoiceAttachmentsLegacyBackfillAndValidation(t *testing.T) {
	request := setupInvoiceAttachmentTest(t)
	messages := captureInvoiceSMTP(t)
	require.NoError(t, os.WriteFile("invoice.pdf", []byte("%PDF-1.4 legacy invoice"), 0600))
	_, err := model.ApproveInvoiceRequest(request.Id, 1, model.InvoiceReviewInput{InvoiceFileName: "invoice.pdf", InvoiceFilePath: "invoice.pdf"})
	require.NoError(t, err)
	fields := map[string]string{"send_detail_bill": "true", "send_service_confirmation": "true"}
	context, recorder := invoiceAttachmentContext(t, request.Id, fields, nil)
	ResendInvoiceEmail(context)
	success, message, _ := invoiceAttachmentResponse(t, recorder)
	require.False(t, success)
	require.Contains(t, message, "尚未保存明细账单")
	require.Empty(t, messages)

	// A valid first upload is cleaned up if the second upload is invalid.
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, map[string]string{"detail_bill_file": "%PDF-1.4 detail", "service_confirmation_file": "not a PDF"})
	ResendInvoiceEmail(context)
	success, message, _ = invoiceAttachmentResponse(t, recorder)
	require.False(t, success)
	require.Contains(t, message, "有效 PDF")
	paths, err := filepath.Glob("data/invoices/*/*/*.pdf")
	require.NoError(t, err)
	require.Empty(t, paths)
	stored, err := model.GetInvoiceRequestDetail(request.Id)
	require.NoError(t, err)
	require.Empty(t, stored.DetailBillFilePath)

	files := map[string]string{"detail_bill_file": "%PDF-1.4 detail", "service_confirmation_file": "%PDF-1.4 products"}
	context, recorder = invoiceAttachmentContext(t, request.Id, fields, files)
	ResendInvoiceEmail(context)
	success, message, response := invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.NotEmpty(t, response.DetailBillFileName)
	require.NotEmpty(t, response.ServiceConfirmationFileName)
	want := invoiceMIMEAttachments(t, messages)
	// JSON clients may also select saved attachments without multipart uploads.
	context, recorder = invoiceAttachmentContext(t, request.Id, nil, nil)
	context.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"send_detail_bill":true,"send_service_confirmation":true}`))
	context.Request.Header.Set("Content-Type", "application/json")
	ResendInvoiceEmail(context)
	success, message, _ = invoiceAttachmentResponse(t, recorder)
	require.True(t, success, message)
	require.Equal(t, want, invoiceMIMEAttachments(t, messages))
}

func TestInvoiceAttachmentsCleanupWhenApprovalRejected(t *testing.T) {
	request := setupInvoiceAttachmentTest(t)
	require.NoError(t, model.DB.Model(request).Update("status", model.InvoiceStatusRejected).Error)
	context, recorder := invoiceAttachmentContext(t, request.Id,
		map[string]string{"send_email": "true", "send_detail_bill": "true"},
		map[string]string{"invoice_file": "%PDF-1.4 invoice", "detail_bill_file": "%PDF-1.4 detail"})
	ApproveInvoiceRequest(context)
	success, _, _ := invoiceAttachmentResponse(t, recorder)
	require.False(t, success)
	paths, err := filepath.Glob("data/invoices/*/*/*.pdf")
	require.NoError(t, err)
	require.Empty(t, paths)
}

func TestInvoiceAttachmentsDatabaseFailureDoesNotSendOrKeepNewFiles(t *testing.T) {
	request := setupInvoiceAttachmentTest(t)
	messages := captureInvoiceSMTP(t)
	_, err := model.ApproveInvoiceRequest(request.Id, 1, model.InvoiceReviewInput{InvoiceNo: "TEST"})
	require.NoError(t, err)
	require.NoError(t, model.DB.Callback().Update().Before("gorm:update").Register("invoice_test_failure", func(tx *gorm.DB) {
		tx.AddError(fmt.Errorf("test database unavailable"))
	}))
	context, recorder := invoiceAttachmentContext(t, request.Id,
		map[string]string{"send_detail_bill": "true"}, map[string]string{"detail_bill_file": "%PDF-1.4 detail"})
	ResendInvoiceEmail(context)
	success, message, _ := invoiceAttachmentResponse(t, recorder)
	require.False(t, success)
	require.Contains(t, message, "test database unavailable")
	require.Empty(t, messages)
	paths, err := filepath.Glob("data/invoices/*/*/*.pdf")
	require.NoError(t, err)
	require.Empty(t, paths)
}

func TestInvoiceAttachmentsRejectCorruptOrOversizedSavedFiles(t *testing.T) {
	for _, size := range []int{0, 64, maxInvoiceDetailBillSize + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "saved.pdf")
			file, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, file.Truncate(int64(size)))
			require.NoError(t, file.Close())
			_, err = readSavedInvoiceEmailAttachment(path, "saved.pdf", "明细账单")
			require.ErrorContains(t, err, "无效")
		})
	}
}

func TestInvoiceAttachmentsMigrateExistingRecord(t *testing.T) {
	request := setupInvoiceAttachmentTest(t)
	// Model the previous schema, then rerun the same AutoMigrate used at startup.
	for _, column := range []string{"detail_bill_file_name", "detail_bill_file_path", "service_confirmation_file_name", "service_confirmation_file_path"} {
		require.NoError(t, model.DB.Migrator().DropColumn(&model.InvoiceRequest{}, column))
	}
	require.NoError(t, model.DB.AutoMigrate(&model.InvoiceRequest{}))
	stored, err := model.GetInvoiceRequestDetail(request.Id)
	require.NoError(t, err)
	require.Empty(t, stored.InvoiceEmailAttachments)
	require.Equal(t, request.Status, stored.Status)
	require.Equal(t, request.ServiceFeeQuota, stored.ServiceFeeQuota)
}
