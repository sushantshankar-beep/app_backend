package service

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"app_backend/internal/domain"

	pdf "github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

func generateInvoicePDF(inv *domain.Invoice) {
	tpl, err := template.ParseFiles("internal/templates/invoice.html")
	if err != nil {
		fmt.Println("template error:", err)
		return
	}
	data := map[string]any{
		"InvoiceNumber": inv.InvoiceNumber,
		"Date":          time.Now().Format("02 Jan 2006"),
		"Items":         inv.Items,
		"SubTotal":      inv.SubTotal,
		"TaxAmount":     inv.TaxAmount,
		"TotalAmount":   inv.TotalAmount,
	}

	var html bytes.Buffer
	if err := tpl.Execute(&html, data); err != nil {
		fmt.Println("html exec error:", err)
		return
	}
	pdfg, err := pdf.NewPDFGenerator()
	if err != nil {
		fmt.Println("pdf gen error:", err)
		return
	}
	page := pdf.NewPageReader(&html)
	pdfg.AddPage(page)
	pdfg.PageSize.Set(pdf.PageSizeA4)
	pdfg.Dpi.Set(300)
	if err := pdfg.Create(); err != nil {
		fmt.Println("pdf create error:", err)
		return
	}
	fileName := fmt.Sprintf("invoice_%s.pdf", inv.InvoiceNumber)
	filePath := filepath.Join("uploads/invoices", fileName)
	_ = os.MkdirAll("uploads/invoices", 0755)
	if err := pdfg.WriteFile(filePath); err != nil {
		fmt.Println("pdf write error:", err)
		return
	}
	inv.PDFUrl = "https://your-cdn.com/invoices/" + fileName
	fmt.Println("Invoice PDF generated:", inv.PDFUrl)
}
