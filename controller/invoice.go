package controller

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const maxInvoicePDFSize = 10 * 1024 * 1024

type createInvoiceRequestPayload struct {
	TitleType string                  `json:"title_type"`
	Title     string                  `json:"title"`
	TaxNumber string                  `json:"tax_number"`
	Email     string                  `json:"email"`
	Phone     string                  `json:"phone"`
	Remark    string                  `json:"remark"`
	Orders    []model.InvoiceOrderRef `json:"orders"`
}

type reviewInvoiceRequestPayload struct {
	InvoiceNo   string `json:"invoice_no"`
	InvoiceUrl  string `json:"invoice_url"`
	AdminRemark string `json:"admin_remark"`
}

type invoiceApprovePayload struct {
	InvoiceNo     string
	InvoiceUrl    string
	AdminRemark   string
	InvoiceSentTo string
	SendEmail     bool
	FileHeader    *multipart.FileHeader
}

func GetEligibleInvoiceOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	records, total, err := model.GetEligibleInvoiceOrders(c.GetInt("id"), pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(records)
	common.ApiSuccess(c, pageInfo)
}

func CreateInvoiceRequest(c *gin.Context) {
	var req createInvoiceRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}

	invoiceRequest, err := model.CreateInvoiceRequest(c.GetInt("id"), model.CreateInvoiceRequestInput{
		TitleType: req.TitleType,
		Title:     req.Title,
		TaxNumber: req.TaxNumber,
		Email:     req.Email,
		Phone:     req.Phone,
		Remark:    req.Remark,
		Orders:    req.Orders,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, invoiceRequest)
}

func GetUserInvoiceRequests(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.InvoiceRequestSearchParams{
		Status: c.Query("status"),
	}
	requests, total, err := model.GetUserInvoiceRequestsByParams(c.GetInt("id"), params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(requests)
	common.ApiSuccess(c, pageInfo)
}

func GetUserInvoiceRequestDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	request, err := model.GetUserInvoiceRequestDetail(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func GetAllInvoiceRequests(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	params := model.InvoiceRequestSearchParams{
		Status:   c.Query("status"),
		Username: c.Query("username"),
	}
	requests, total, err := model.GetAllInvoiceRequestsByParams(params, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(requests)
	common.ApiSuccess(c, pageInfo)
}

func GetInvoiceRequestDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	request, err := model.GetInvoiceRequestDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func ApproveInvoiceRequest(c *gin.Context) {
	reviewInvoiceRequest(c, "approve")
}

func RejectInvoiceRequest(c *gin.Context) {
	reviewInvoiceRequest(c, "reject")
}

func reviewInvoiceRequest(c *gin.Context, action string) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}

	var request *model.InvoiceRequest
	if action == "approve" {
		req, err := parseInvoiceApprovePayload(c, id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		fileName := ""
		filePath := ""
		cleanupFile := false
		if req.FileHeader != nil {
			fileName, filePath, err = saveInvoicePDFFile(id, req.FileHeader)
			if err != nil {
				common.ApiError(c, err)
				return
			}
			cleanupFile = true
			defer func() {
				if cleanupFile && filePath != "" {
					_ = os.Remove(filePath)
				}
			}()
		}
		request, err = model.ApproveInvoiceRequest(id, c.GetInt("id"), model.InvoiceReviewInput{
			InvoiceNo:         req.InvoiceNo,
			InvoiceUrl:        req.InvoiceUrl,
			InvoiceFileName:   fileName,
			InvoiceFilePath:   filePath,
			InvoiceSentTo:     req.InvoiceSentTo,
			InvoiceSendStatus: model.InvoiceSendStatusPending,
			AdminRemark:       req.AdminRemark,
		})
		if err == nil {
			cleanupFile = false
		}
		if err == nil && req.SendEmail {
			request, err = sendInvoiceFileAndUpdateStatus(request)
		}
	} else {
		var req reviewInvoiceRequestPayload
		if err := c.ShouldBindJSON(&req); err != nil {
			common.ApiError(c, err)
			return
		}
		request, err = model.RejectInvoiceRequest(id, c.GetInt("id"), req.AdminRemark)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func ResendInvoiceEmail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	request, err := model.GetInvoiceRequestDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if request.Status != model.InvoiceStatusInvoiced {
		common.ApiErrorMsg(c, "发票尚未开具")
		return
	}
	request, err = sendInvoiceFileAndUpdateStatus(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func DownloadUserInvoiceFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	request, err := model.GetUserInvoiceRequestDetail(c.GetInt("id"), id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	downloadInvoiceFile(c, request)
}

func DownloadInvoiceFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	request, err := model.GetInvoiceRequestDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	downloadInvoiceFile(c, request)
}

func parseInvoiceApprovePayload(c *gin.Context, id int) (invoiceApprovePayload, error) {
	payload := invoiceApprovePayload{SendEmail: true}
	contentType := c.GetHeader("Content-Type")
	if strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		detail, err := model.GetInvoiceRequestDetail(id)
		if err != nil {
			return payload, err
		}
		payload.InvoiceNo = strings.TrimSpace(c.PostForm("invoice_no"))
		payload.InvoiceUrl = strings.TrimSpace(c.PostForm("invoice_url"))
		payload.AdminRemark = strings.TrimSpace(c.PostForm("admin_remark"))
		payload.InvoiceSentTo = strings.TrimSpace(c.PostForm("invoice_sent_to"))
		if payload.InvoiceSentTo == "" {
			payload.InvoiceSentTo = detail.Email
		}
		if raw := strings.TrimSpace(c.PostForm("send_email")); raw != "" {
			payload.SendEmail = raw == "true" || raw == "1" || strings.EqualFold(raw, "on")
		}
		fileHeader, err := c.FormFile("invoice_file")
		if err != nil {
			return payload, fmt.Errorf("请上传发票 PDF")
		}
		payload.FileHeader = fileHeader
		if payload.SendEmail && payload.InvoiceSentTo == "" {
			return payload, fmt.Errorf("发票接收邮箱不能为空")
		}
		return payload, nil
	}

	var req reviewInvoiceRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		return payload, err
	}
	payload.InvoiceNo = req.InvoiceNo
	payload.InvoiceUrl = req.InvoiceUrl
	payload.AdminRemark = req.AdminRemark
	payload.SendEmail = false
	return payload, nil
}

func saveInvoicePDFFile(invoiceID int, fileHeader *multipart.FileHeader) (string, string, error) {
	if fileHeader == nil {
		return "", "", fmt.Errorf("请上传发票 PDF")
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxInvoicePDFSize {
		return "", "", fmt.Errorf("发票 PDF 大小不能超过 10MB")
	}
	if strings.ToLower(filepath.Ext(fileHeader.Filename)) != ".pdf" {
		return "", "", fmt.Errorf("仅支持上传 PDF 文件")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxInvoicePDFSize+1))
	if err != nil {
		return "", "", err
	}
	if len(data) == 0 || len(data) > maxInvoicePDFSize {
		return "", "", fmt.Errorf("发票 PDF 大小不能超过 10MB")
	}
	header := data
	if len(header) > 1024 {
		header = header[:1024]
	}
	if !bytes.Contains(header, []byte("%PDF-")) {
		return "", "", fmt.Errorf("仅支持上传有效 PDF 文件")
	}

	now := time.Now()
	dir := filepath.Join("data", "invoices", now.Format("2006"), now.Format("01"))
	if err := os.MkdirAll(dir, 0750); err != nil {
		return "", "", err
	}
	safeName := fmt.Sprintf("invoice_%d_%d.pdf", invoiceID, now.UnixNano())
	filePath, err := filepath.Abs(filepath.Join(dir, safeName))
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", "", err
	}
	return sanitizeInvoiceFilename(fileHeader.Filename), filePath, nil
}

func sanitizeInvoiceFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		return "invoice.pdf"
	}
	return filename
}

func sendInvoiceFileAndUpdateStatus(request *model.InvoiceRequest) (*model.InvoiceRequest, error) {
	if request == nil {
		return nil, model.ErrInvoiceRequestNotFound
	}
	err := sendInvoiceFileEmail(request)
	if err != nil {
		updated, updateErr := model.UpdateInvoiceSendStatus(request.Id, model.InvoiceSendStatusFailed, err.Error())
		if updateErr != nil {
			return nil, updateErr
		}
		return updated, nil
	}
	return model.UpdateInvoiceSendStatus(request.Id, model.InvoiceSendStatusSent, "")
}

func sendInvoiceFileEmail(request *model.InvoiceRequest) error {
	if strings.TrimSpace(request.InvoiceFilePath) == "" {
		return fmt.Errorf("发票 PDF 文件不存在")
	}
	receiver := strings.TrimSpace(request.InvoiceSentTo)
	if receiver == "" {
		receiver = strings.TrimSpace(request.Email)
	}
	if receiver == "" {
		return fmt.Errorf("发票接收邮箱不能为空")
	}
	data, err := os.ReadFile(request.InvoiceFilePath)
	if err != nil {
		return err
	}
	filename := strings.TrimSpace(request.InvoiceFileName)
	if filename == "" {
		filename = fmt.Sprintf("invoice-%d.pdf", request.Id)
	}
	subject := fmt.Sprintf("%s 发票已开具", common.SystemName)
	content := buildInvoiceEmailContent(request)
	return common.SendEmailWithAttachments(subject, receiver, content, []common.EmailAttachment{{
		Filename:    filename,
		ContentType: "application/pdf",
		Data:        data,
	}})
}

func buildInvoiceEmailContent(request *model.InvoiceRequest) string {
	title := html.EscapeString(request.Title)
	invoiceNo := html.EscapeString(request.InvoiceNo)
	totalMoney := fmt.Sprintf("%.2f", request.TotalMoney)
	return fmt.Sprintf(`<p>您好，您的发票申请已审核通过，发票 PDF 见附件。</p>
<p>发票抬头：%s</p>
<p>发票号/代码：%s</p>
<p>开票金额：¥%s</p>
<p>如有疑问，请联系平台管理员。</p>`, title, invoiceNo, totalMoney)
}

func downloadInvoiceFile(c *gin.Context, request *model.InvoiceRequest) {
	if request == nil || request.Status != model.InvoiceStatusInvoiced || strings.TrimSpace(request.InvoiceFilePath) == "" {
		common.ApiErrorMsg(c, "发票 PDF 不存在")
		return
	}
	if _, err := os.Stat(request.InvoiceFilePath); err != nil {
		common.ApiErrorMsg(c, "发票 PDF 不存在")
		return
	}
	filename := request.InvoiceFileName
	if filename == "" {
		filename = fmt.Sprintf("invoice-%d.pdf", request.Id)
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, strings.ReplaceAll(filename, `"`, "")))
	c.File(request.InvoiceFilePath)
}
