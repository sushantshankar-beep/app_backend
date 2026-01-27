package pdf

import (
	"fmt"
	"time"
     "app_backend/internal/domain" 
	"github.com/jung-kurt/gofpdf"
)

type PDFGenerator struct{}

func NewPDFGenerator() *PDFGenerator {
	return &PDFGenerator{}
}

func (g *PDFGenerator) GenerateInvoicePDF(invoice *domain.Invoice) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(15, 15, 15)

	pdf.SetFont("Arial", "B", 24)
	pdf.CellFormat(180, 12, "INVOICE", "", 1, "C", false, 0, "")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(90, 6, fmt.Sprintf("Invoice Number: %s", invoice.InvoiceNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, fmt.Sprintf("Invoice Date: %s", formatDate(invoice.CreatedAt)), "", 1, "L", false, 0, "")
	
	if invoice.ServiceDate != nil {
		pdf.CellFormat(90, 6, fmt.Sprintf("Service Date: %s", formatDate(*invoice.ServiceDate)), "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(90, 8, "Provider Details", "1", 0, "L", true, 0, "")
	pdf.CellFormat(90, 8, "Customer Details", "1", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(90, 6, fmt.Sprintf("Name: %s", invoice.ProviderName), "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, fmt.Sprintf("Name: %s", invoice.BillToName), "1", 1, "L", false, 0, "")

	pdf.CellFormat(90, 6, fmt.Sprintf("Phone: %s", invoice.ProviderPhone), "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, fmt.Sprintf("Phone: %s", invoice.BillToPhone), "1", 1, "L", false, 0, "")

	pdf.CellFormat(90, 6, fmt.Sprintf("GST: %s", invoice.ProviderGST), "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, truncateText(fmt.Sprintf("Address: %s", invoice.BillToAddress), 40), "1", 1, "L", false, 0, "")

	pdf.CellFormat(90, 6, truncateText(fmt.Sprintf("Address: %s", invoice.ProviderAddress), 40), "1", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, "", "1", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(180, 8, "Vehicle Details", "1", 1, "L", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(45, 6, fmt.Sprintf("Brand: %s", invoice.VehicleBrand), "1", 0, "L", false, 0, "")
	pdf.CellFormat(45, 6, fmt.Sprintf("Model: %s", invoice.VehicleModel), "1", 0, "L", false, 0, "")
	pdf.CellFormat(45, 6, fmt.Sprintf("Year: %d", invoice.VehicleYear), "1", 0, "L", false, 0, "")
	pdf.CellFormat(45, 6, fmt.Sprintf("Type: %s", invoice.VehicleType), "1", 1, "L", false, 0, "")

	pdf.CellFormat(60, 6, fmt.Sprintf("Number: %s", invoice.VehicleNumber), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, fmt.Sprintf("Fuel: %s", invoice.FuelType), "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 6, fmt.Sprintf("Service: %s", invoice.ServiceType), "1", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 12)
	pdf.SetFillColor(51, 51, 51)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(70, 8, "Item", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 8, "Qty", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Price", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "GST %", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Amount", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Arial", "", 10)

	for i, item := range invoice.Items {
		if i%2 == 0 {
			pdf.SetFillColor(249, 249, 249)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		pdf.CellFormat(70, 7, truncateText(item.Name, 30), "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 7, fmt.Sprintf("%d", item.Qty), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("Rs. %.2f", item.Price), "1", 0, "R", true, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("%.1f%%", item.GSTPercent), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, fmt.Sprintf("Rs. %.2f", item.GrossAmount), "1", 1, "R", true, 0, "")
	}
	pdf.Ln(3)

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(150, 7, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, fmt.Sprintf("Rs. %.2f", invoice.SubTotal), "", 1, "R", false, 0, "")

	pdf.CellFormat(150, 7, "Tax (GST):", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 7, fmt.Sprintf("Rs. %.2f", invoice.TaxAmount), "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "B", 13)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(2)
	pdf.CellFormat(150, 8, "Total Amount:", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("Rs. %.2f", invoice.TotalAmount), "", 1, "R", false, 0, "")

	var buf []byte
	w := &bytesWriter{buf: &buf}
	err := pdf.Output(w)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

type bytesWriter struct {
	buf *[]byte
}

func (w *bytesWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}