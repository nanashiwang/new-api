package controller

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type preparedInvoiceAttachments struct {
	files    model.InvoiceEmailAttachments
	email    []*common.EmailAttachment
	newPaths []string
}

func (p *preparedInvoiceAttachments) cleanup() {
	for _, path := range p.newPaths {
		_ = os.Remove(path)
	}
}

func (p *preparedInvoiceAttachments) retain() {
	p.newPaths = nil
}

func prepareInvoiceEmailAttachments(request *model.InvoiceRequest, payload invoiceEmailPayload) (*preparedInvoiceAttachments, error) {
	prepared := &preparedInvoiceAttachments{}
	attachments := []struct {
		selected                bool
		upload                  *multipart.FileHeader
		label, kind, name, path string
		savedName, savedPath    *string
	}{
		{payload.SendDetailBill, payload.DetailBillFileHeader, "明细账单", "detail_bill", request.DetailBillFileName, request.DetailBillFilePath, &prepared.files.DetailBillFileName, &prepared.files.DetailBillFilePath},
		{payload.SendServiceConfirmation, payload.ServiceConfirmationFileHeader, "产品明细清单", "service_confirmation", request.ServiceConfirmationFileName, request.ServiceConfirmationFilePath, &prepared.files.ServiceConfirmationFileName, &prepared.files.ServiceConfirmationFilePath},
	}
	for _, item := range attachments {
		if !item.selected {
			continue
		}
		var attachment *common.EmailAttachment
		var err error
		if item.upload != nil {
			attachment, err = readInvoicePDFAttachment(item.upload, item.label, item.label+".pdf")
			if err == nil {
				var path string
				path, err = saveInvoiceEmailAttachment(request.Id, item.kind, attachment.Data)
				if err == nil {
					prepared.newPaths = append(prepared.newPaths, path)
					*item.savedName = attachment.Filename
					*item.savedPath = path
				}
			}
		} else {
			attachment, err = readSavedInvoiceEmailAttachment(item.path, item.name, item.label)
		}
		if err != nil {
			prepared.cleanup()
			return nil, err
		}
		prepared.email = append(prepared.email, attachment)
	}
	return prepared, nil
}

func saveInvoiceEmailAttachment(invoiceID int, kind string, data []byte) (string, error) {
	now := time.Now()
	dir, err := filepath.Abs(filepath.Join("data", "invoices", now.Format("2006"), now.Format("01")))
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", err
	}
	// Each replacement has its own path. Keep older files available to resends
	// already in progress that may still be reading the previous attachment.
	file, err := os.CreateTemp(dir, fmt.Sprintf("invoice_%d_%s_*.pdf", invoiceID, kind))
	if err != nil {
		return "", err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(file.Name())
		if writeErr != nil {
			return "", writeErr
		}
		return "", closeErr
	}
	return file.Name(), nil
}

func readSavedInvoiceEmailAttachment(path, name, label string) (*common.EmailAttachment, error) {
	if path == "" {
		return nil, fmt.Errorf("尚未保存%s PDF，请上传一次后再重发", label)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("已保存的%s PDF 无法读取，请重新上传", label)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxInvoiceDetailBillSize+1))
	if err != nil || len(data) > maxInvoiceDetailBillSize || !bytes.HasPrefix(bytes.TrimSpace(data), []byte("%PDF-")) {
		return nil, fmt.Errorf("已保存的%s PDF 无效，请重新上传", label)
	}
	return &common.EmailAttachment{
		Filename:    sanitizeInvoicePDFAttachmentFilename(name, label+".pdf"),
		ContentType: "application/pdf",
		Data:        data,
	}, nil
}
