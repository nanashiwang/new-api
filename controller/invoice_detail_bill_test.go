package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestBuildInvoiceDetailBillHTMLUsesTransactionProofWording(t *testing.T) {
	html := buildInvoiceDetailBillHTML(&model.InvoiceRequest{
		Id:         7,
		UserId:     29,
		Username:   "zhuhongjun823",
		TotalMoney: 414,
		TotalQuota: 1,
		Title:      "南京信息工程大学",
		TaxNumber:  "12320000466006762K",
		Email:      "csx20020320@126.com",
		Items: []model.InvoiceRequestItem{{
			OrderType:     model.PaymentRecordTypeTopUp,
			OrderId:       23,
			TradeNo:       "2026-04-09",
			PaymentMethod: "alipay",
			Money:         414,
			Amount:        1,
			CompleteTime:  1775702400,
		}},
	})

	for _, want := range []string{
		"曜算平台交易明细证明",
		"用户 zhuhongjun823（用户 ID：29）",
		"于曜算平台存在相关交易记录",
		"支付金额合计人民币 ¥414.00",
		"本《曜算平台交易明细证明》",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("detail bill html missing %q:\n%s", want, html)
		}
	}
	for _, unexpected := range []string{
		"发票申请明细证明",
		"额度合计",
		"发票抬头",
		"南京信息工程大学",
		"12320000466006762K",
		"csx20020320@126.com",
	} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("detail bill html should not contain %q:\n%s", unexpected, html)
		}
	}
}

func TestBuildInvoiceDetailBillHTMLUsesManualTransferWordingAndNoFakeOrderID(t *testing.T) {
	html := buildInvoiceDetailBillHTML(&model.InvoiceRequest{
		Id:         8,
		UserId:     30,
		Username:   "manual-user",
		SourceType: model.InvoiceSourceTypeManualTransfer,
		TotalMoney: 1000,
		Items: []model.InvoiceRequestItem{{
			SourceType:       model.InvoiceSourceTypeManualTransfer,
			OrderType:        model.InvoiceOrderTypeManualTransfer,
			TradeNo:          "BANK-001",
			PaymentMethod:    model.InvoicePaymentMethodBankTransfer,
			Money:            1000,
			PayerName:        "付款测试公司",
			PayeeName:        "收款测试公司",
			TransferBankName: "测试银行",
			CompleteTime:     1775702400,
		}},
	})

	for _, want := range []string{
		"转账",
		"申请人提交的银行转账凭证及平台审核结果",
		"付款测试公司",
		"收款测试公司",
		"付款测试公司 → 收款测试公司",
		"银行转账（测试银行）",
		"BANK-001",
		"不会据此自动增加或扣减用户钱包余额",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("manual detail bill html missing %q:\n%s", want, html)
		}
	}
	for _, unexpected := range []string{
		"平台订单 ID",
		">0</td>",
		"平台系统记录的交易明细如下",
		"公对公",
	} {
		if strings.Contains(html, unexpected) {
			t.Fatalf("manual detail bill html should not contain %q:\n%s", unexpected, html)
		}
	}
}
