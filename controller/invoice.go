package controller

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	maxInvoicePDFSize        = 10 * 1024 * 1024
	maxInvoiceDetailBillSize = 10 * 1024 * 1024
)

type createInvoiceRequestPayload struct {
	InvoiceType       string                  `json:"invoice_type"`
	TitleType         string                  `json:"title_type"`
	Title             string                  `json:"title"`
	TaxNumber         string                  `json:"tax_number"`
	RegisteredAddress string                  `json:"registered_address"`
	RegisteredPhone   string                  `json:"registered_phone"`
	BankName          string                  `json:"bank_name"`
	BankAccount       string                  `json:"bank_account"`
	Email             string                  `json:"email"`
	Phone             string                  `json:"phone"`
	Remark            string                  `json:"remark"`
	Orders            []model.InvoiceOrderRef `json:"orders"`
}

type reviewInvoiceRequestPayload struct {
	InvoiceNo   string `json:"invoice_no"`
	InvoiceUrl  string `json:"invoice_url"`
	AdminRemark string `json:"admin_remark"`
}

type invoiceApprovePayload struct {
	InvoiceNo            string
	InvoiceUrl           string
	AdminRemark          string
	InvoiceSentTo        string
	SendEmail            bool
	SendDetailBill       bool
	FileHeader           *multipart.FileHeader
	DetailBillFileHeader *multipart.FileHeader
}

type invoiceEmailPayload struct {
	SendDetailBill       bool
	DetailBillFileHeader *multipart.FileHeader
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
		InvoiceType:       req.InvoiceType,
		TitleType:         req.TitleType,
		Title:             req.Title,
		TaxNumber:         req.TaxNumber,
		RegisteredAddress: req.RegisteredAddress,
		RegisteredPhone:   req.RegisteredPhone,
		BankName:          req.BankName,
		BankAccount:       req.BankAccount,
		Email:             req.Email,
		Phone:             req.Phone,
		Remark:            req.Remark,
		Orders:            req.Orders,
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
		var detailBillAttachment *common.EmailAttachment
		if req.SendEmail && req.SendDetailBill && req.DetailBillFileHeader != nil {
			detailBillAttachment, err = readInvoiceDetailBillAttachment(req.DetailBillFileHeader)
			if err != nil {
				common.ApiError(c, err)
				return
			}
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
			if req.SendDetailBill && detailBillAttachment == nil {
				err = fmt.Errorf("请上传明细账单 PDF")
			} else {
				request, err = sendInvoiceFileAndUpdateStatus(request, detailBillAttachment)
			}
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
	payload, err := parseInvoiceEmailPayload(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var detailBillAttachment *common.EmailAttachment
	if payload.SendDetailBill {
		if payload.DetailBillFileHeader != nil {
			detailBillAttachment, err = readInvoiceDetailBillAttachment(payload.DetailBillFileHeader)
			if err != nil {
				common.ApiError(c, err)
				return
			}
		} else {
			common.ApiError(c, fmt.Errorf("请上传明细账单 PDF"))
			return
		}
	}
	request, err = sendInvoiceFileAndUpdateStatus(request, detailBillAttachment)
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
			payload.SendEmail = parseInvoiceBool(raw, payload.SendEmail)
		}
		if raw := strings.TrimSpace(c.PostForm("send_detail_bill")); raw != "" {
			payload.SendDetailBill = parseInvoiceBool(raw, false)
		}
		fileHeader, err := c.FormFile("invoice_file")
		if err != nil {
			return payload, fmt.Errorf("请上传发票 PDF")
		}
		payload.FileHeader = fileHeader
		if payload.SendDetailBill {
			if detailBillFileHeader, err := c.FormFile("detail_bill_file"); err == nil {
				payload.DetailBillFileHeader = detailBillFileHeader
			} else if payload.SendEmail {
				return payload, fmt.Errorf("请上传明细账单 PDF")
			}
		}
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

func parseInvoiceEmailPayload(c *gin.Context) (invoiceEmailPayload, error) {
	payload := invoiceEmailPayload{}
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if raw := strings.TrimSpace(c.PostForm("send_detail_bill")); raw != "" {
			payload.SendDetailBill = parseInvoiceBool(raw, false)
		}
		if payload.SendDetailBill {
			if fileHeader, err := c.FormFile("detail_bill_file"); err == nil {
				payload.DetailBillFileHeader = fileHeader
			} else {
				return payload, fmt.Errorf("请上传明细账单 PDF")
			}
		}
		return payload, nil
	}
	if c.Request.ContentLength == 0 {
		return payload, nil
	}
	var req struct {
		SendDetailBill bool `json:"send_detail_bill"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		return payload, err
	}
	if req.SendDetailBill {
		return payload, fmt.Errorf("请上传明细账单 PDF")
	}
	payload.SendDetailBill = req.SendDetailBill
	return payload, nil
}

func parseInvoiceBool(raw string, defaultValue bool) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue
	}
	return raw == "true" || raw == "1" || strings.EqualFold(raw, "on") || strings.EqualFold(raw, "yes")
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

func readInvoiceDetailBillAttachment(fileHeader *multipart.FileHeader) (*common.EmailAttachment, error) {
	if fileHeader == nil {
		return nil, nil
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxInvoiceDetailBillSize {
		return nil, fmt.Errorf("明细账单 PDF 大小不能超过 10MB")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" {
		return nil, fmt.Errorf("明细账单附件仅支持 PDF 文件")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxInvoiceDetailBillSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > maxInvoiceDetailBillSize {
		return nil, fmt.Errorf("明细账单 PDF 大小不能超过 10MB")
	}
	return &common.EmailAttachment{
		Filename:    sanitizeInvoiceDetailBillFilename(fileHeader.Filename),
		ContentType: "application/pdf",
		Data:        data,
	}, nil
}

func sanitizeInvoiceDetailBillFilename(filename string) string {
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		return "明细账单.pdf"
	}
	if strings.ToLower(filepath.Ext(filename)) != ".pdf" {
		filename += ".pdf"
	}
	return filename
}

func sendInvoiceFileAndUpdateStatus(request *model.InvoiceRequest, detailBillAttachment *common.EmailAttachment) (*model.InvoiceRequest, error) {
	if request == nil {
		return nil, model.ErrInvoiceRequestNotFound
	}
	err := sendInvoiceFileEmail(request, detailBillAttachment)
	if err != nil {
		updated, updateErr := model.UpdateInvoiceSendStatus(request.Id, model.InvoiceSendStatusFailed, err.Error())
		if updateErr != nil {
			return nil, updateErr
		}
		return updated, nil
	}
	return model.UpdateInvoiceSendStatus(request.Id, model.InvoiceSendStatusSent, "")
}

func sendInvoiceFileEmail(request *model.InvoiceRequest, detailBillAttachment *common.EmailAttachment) error {
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
	attachments := []common.EmailAttachment{{
		Filename:    filename,
		ContentType: "application/pdf",
		Data:        data,
	}}
	if detailBillAttachment != nil && len(detailBillAttachment.Data) > 0 {
		attachments = append(attachments, *detailBillAttachment)
	}
	return common.SendEmailWithAttachments(subject, receiver, content, attachments)
}

func buildInvoiceEmailContent(request *model.InvoiceRequest) string {
	title := html.EscapeString(request.Title)
	invoiceNo := html.EscapeString(request.InvoiceNo)
	totalMoney := fmt.Sprintf("%.2f", request.TotalMoney)
	invoiceType := "普票"
	titleLabel := "发票抬头"
	if request.InvoiceType == model.InvoiceTypeSpecial {
		invoiceType = "专票"
		titleLabel = "单位名称"
	}
	content := fmt.Sprintf(`<p>您好，您的发票申请已审核通过，发票 PDF 见附件。</p>
<p>发票类型：%s</p>
<p>%s：%s</p>
<p>发票号/代码：%s</p>
<p>开票金额：¥%s</p>`, invoiceType, titleLabel, title, invoiceNo, totalMoney)
	if request.InvoiceType == model.InvoiceTypeSpecial {
		content += fmt.Sprintf(`<p>税号：%s</p>`, html.EscapeString(request.TaxNumber))
		if strings.TrimSpace(request.RegisteredAddress) != "" {
			content += fmt.Sprintf(`<p>注册地址：%s</p>`, html.EscapeString(request.RegisteredAddress))
		}
		if strings.TrimSpace(request.RegisteredPhone) != "" {
			content += fmt.Sprintf(`<p>注册电话：%s</p>`, html.EscapeString(request.RegisteredPhone))
		}
		if strings.TrimSpace(request.BankName) != "" {
			content += fmt.Sprintf(`<p>开户银行：%s</p>`, html.EscapeString(request.BankName))
		}
		if strings.TrimSpace(request.BankAccount) != "" {
			content += fmt.Sprintf(`<p>银行账号：%s</p>`, html.EscapeString(request.BankAccount))
		}
	}
	return content + `<p>如有疑问，请联系平台管理员。</p>`
}

type invoiceDetailBillTypeSummary struct {
	Count int
	Money float64
}

func buildInvoiceDetailBillHTML(request *model.InvoiceRequest) string {
	items := request.Items
	count := len(items)
	totalMoney := request.TotalMoney
	startTime := int64(0)
	endTime := int64(0)
	byType := make(map[string]invoiceDetailBillTypeSummary)
	for _, item := range items {
		stats := byType[item.OrderType]
		stats.Count++
		stats.Money += item.Money
		byType[item.OrderType] = stats
		orderTime := item.CompleteTime
		if orderTime == 0 {
			orderTime = item.CreateTime
		}
		if orderTime > 0 {
			if startTime == 0 || orderTime < startTime {
				startTime = orderTime
			}
			if orderTime > endTime {
				endTime = orderTime
			}
		}
	}
	cell := func(value interface{}) string {
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" {
			text = "-"
		}
		return html.EscapeString(text)
	}
	orderRows := strings.Builder{}
	if len(items) == 0 {
		orderRows.WriteString(`<tr><td colspan="8" class="empty">暂无订单明细</td></tr>`)
	} else {
		for index, item := range items {
			fmt.Fprintf(&orderRows, `<tr>
<td class="center">%d</td>
<td class="center">%s</td>
<td class="center">%d</td>
<td>%s</td>
<td class="center">%s</td>
<td class="center">%s</td>
<td class="money">%s</td>
<td class="center">%s</td>
</tr>`,
				index+1,
				cell(invoiceDetailBillOrderTypeLabel(item.OrderType)),
				item.OrderId,
				cell(invoiceDetailBillCode(item)),
				cell(item.ProductName),
				cell(invoiceDetailBillPaymentLabel(item.PaymentMethod)),
				cell(invoiceDetailBillPaymentAmount(item)),
				cell(invoiceDetailBillTime(invoiceDetailBillOrderTime(item))),
			)
		}
	}
	typeSummary := invoiceDetailBillTypeSummaryText(byType)
	timeRange := "-"
	if startTime > 0 {
		timeRange = fmt.Sprintf("%s 至 %s", invoiceDetailBillTime(startTime), invoiceDetailBillTime(endTime))
	}
	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <title>曜算平台交易明细证明 #%s</title>
  <style>
    @page { size: A4 landscape; margin: 14mm; }
    * { box-sizing: border-box; }
    body { margin: 0; color: #243244; background: #fff; font: 14px/1.55 "Songti SC", "SimSun", "Noto Serif CJK SC", serif; }
    .certificate { position: relative; min-height: 176mm; padding: 4mm 2mm 34mm; }
    h1 { margin: 0; text-align: center; font: 700 25px/1.25 "PingFang SC", "Microsoft YaHei", sans-serif; letter-spacing: 1px; color: #23364a; }
    .title-line { height: 2px; margin: 14px 0 8px; background: #2d5f86; }
    .intro { margin: 0 26px 16px; text-indent: 2em; font-size: 14px; }
    h2 { margin: 0 0 8px; font: 700 16px/1.2 "PingFang SC", "Microsoft YaHei", sans-serif; color: #1f3447; }
    table { width: calc(100%% - 40px); border-collapse: collapse; table-layout: fixed; margin-left: 20px; }
    th { background: #22364b; color: #fff; font-weight: 700; }
    th, td { border: 1px solid #cfd8e3; padding: 5px 7px; vertical-align: middle; word-break: break-all; }
    tbody tr:nth-child(even) { background: #f7f9fb; }
    .center { text-align: center; }
    .money { text-align: right; white-space: nowrap; }
    .summary { margin: 16px 20px 10px; padding: 9px 12px; border: 1px solid #b9c7d7; background: #f7f9fc; font-size: 14px; }
    .summary div { margin: 2px 0; }
    .notes { margin: 10px 20px 0; }
    .notes p { margin: 5px 0; text-indent: 2em; }
    .sign { position: absolute; right: 88px; bottom: 4px; min-width: 250px; min-height: 104px; font-size: 14px; }
    .sign p { margin: 8px 0; }
    .empty { padding: 18px; text-align: center; color: #697586; }
    @media screen { body { padding: 18px; background: #eef2f7; } .certificate { max-width: 1120px; margin: 0 auto; padding: 28px 32px 52px; background: #fff; box-shadow: 0 10px 34px rgba(15, 23, 42, 0.12); } }
  </style>
</head>
<body>
  <main class="certificate">
    <h1>曜算平台交易明细证明</h1>
    <div class="title-line"></div>
    <p class="intro">兹证明：用户 %s 于曜算平台存在相关交易记录。根据该用户申请时所选择的交易类型及时间范围，平台系统记录的交易明细如下：</p>

    <h2>交易明细表</h2>
    <table>
      <thead>
        <tr>
          <th style="width: 44px;">序号</th>
          <th style="width: 90px;">订单类型</th>
          <th style="width: 96px;">平台订单 ID</th>
          <th>订单编码/交易号</th>
          <th style="width: 110px;">商品/套餐</th>
          <th style="width: 96px;">支付渠道</th>
          <th style="width: 120px;">支付金额</th>
          <th style="width: 160px;">支付时间</th>
        </tr>
      </thead>
      <tbody>%s</tbody>
    </table>

    <section class="summary">
      <div>合计：共 %d 笔交易，支付金额合计人民币 %s。</div>
      <div>其中：%s。</div>
      <div>交易时间范围：%s。</div>
    </section>

    <section class="notes">
      <h2>说明</h2>
      <p>本《曜算平台交易明细证明》仅用于证明用户在其申请范围内，于曜算平台产生的相关支付订单记录。</p>
      <p>本证明所列订单明细依据用户申请时选择的订单及平台系统记录生成，具体筛选条件以用户申请页面选择内容为准。</p>
      <p>本证明仅限用于证明用户在曜算平台的相关交易记录，不作为其他权利义务认定依据。</p>
      <p>本证明不得擅自修改、涂改、拆分或用于与申请目的不一致的其他用途。</p>
      <p>本证明中所列时间均为北京时间（UTC+08:00）。</p>
      <p>本证明经上海曜算智能科技有限公司加盖公章后生效。</p>
    </section>

    <section class="sign">
      <p>上海曜算智能科技有限公司</p>
      <p>盖章：</p>
      <p>出具日期：%s</p>
    </section>
  </main>
</body>
</html>`,
		cell(request.Id),
		cell(invoiceDetailBillUser(request)),
		orderRows.String(),
		count,
		cell(invoiceDetailBillMoney(totalMoney)),
		cell(typeSummary),
		cell(timeRange),
		cell(invoiceDetailBillDate(common.GetTimestamp())),
	)
}

func invoiceDetailBillOrderTypeLabel(orderType string) string {
	switch orderType {
	case model.PaymentRecordTypeTopUp:
		return "在线充值"
	case model.PaymentOrderTypeSubscription:
		return "订阅订单"
	case model.PaymentRecordTypeSellableTokenPurchase:
		return "钱包购买令牌"
	default:
		if strings.TrimSpace(orderType) == "" {
			return "-"
		}
		return orderType
	}
}

func invoiceDetailBillPaymentLabel(paymentMethod string) string {
	switch paymentMethod {
	case "alipay":
		return "支付宝"
	case "wxpay":
		return "微信"
	case "stripe":
		return "Stripe"
	case "creem":
		return "Creem"
	case model.PaymentMethodWallet:
		return "钱包余额"
	default:
		if strings.TrimSpace(paymentMethod) == "" {
			return "-"
		}
		return paymentMethod
	}
}

func invoiceDetailBillCode(item model.InvoiceRequestItem) string {
	if strings.TrimSpace(item.TradeNo) != "" {
		return item.TradeNo
	}
	return fmt.Sprintf("%s-%d", item.OrderType, item.OrderId)
}

func invoiceDetailBillOrderTime(item model.InvoiceRequestItem) int64 {
	if item.CompleteTime > 0 {
		return item.CompleteTime
	}
	return item.CreateTime
}

func invoiceDetailBillPaymentAmount(item model.InvoiceRequestItem) string {
	if item.Money > 0 {
		return invoiceDetailBillMoney(item.Money)
	}
	return invoiceDetailBillMoney(0)
}

func invoiceDetailBillMoney(value float64) string {
	return fmt.Sprintf("¥%.2f", value)
}

func invoiceDetailBillTime(value int64) string {
	if value <= 0 {
		return "-"
	}
	return time.Unix(value, 0).In(time.FixedZone("CST", 8*3600)).Format("2006-01-02 15:04:05")
}

func invoiceDetailBillDate(value int64) string {
	if value <= 0 {
		return "-"
	}
	return time.Unix(value, 0).In(time.FixedZone("CST", 8*3600)).Format("2006年1月2日")
}

func invoiceDetailBillUser(request *model.InvoiceRequest) string {
	if strings.TrimSpace(request.Username) != "" {
		displayName := ""
		if strings.TrimSpace(request.DisplayName) != "" && request.DisplayName != request.Username {
			displayName = " / " + strings.TrimSpace(request.DisplayName)
		}
		return fmt.Sprintf("%s%s（用户 ID：%d）", request.Username, displayName, request.UserId)
	}
	if request.UserId > 0 {
		return fmt.Sprintf("用户 ID：%d", request.UserId)
	}
	return "-"
}

func invoiceDetailBillTypeSummaryText(byType map[string]invoiceDetailBillTypeSummary) string {
	if len(byType) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(byType))
	for orderType := range byType {
		keys = append(keys, orderType)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(byType))
	for _, orderType := range keys {
		stats := byType[orderType]
		if stats.Count <= 0 {
			continue
		}
		part := fmt.Sprintf("%s %d 笔", invoiceDetailBillOrderTypeLabel(orderType), stats.Count)
		if stats.Money > 0 {
			part += fmt.Sprintf("，金额合计 %s", invoiceDetailBillMoney(stats.Money))
		}
		parts = append(parts, part)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, "；")
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
