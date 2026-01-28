package service

import (
	"fmt"
	"time"
    "path/filepath"
	"app_backend/internal/domain"
	"github.com/jung-kurt/gofpdf"
)

type PDFService struct {
	pdfDir string
}

func NewPDFService(pdfDir string) *PDFService {
	return &PDFService{
		pdfDir: pdfDir,
	}
}

func (s *PDFService) GenerateInvoicePDF(inv *domain.Invoice) (string, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	s.addHeader(pdf, inv)
	s.addCompanyCustomerInfo(pdf, inv)
	s.addVehicleInfo(pdf, inv)
	s.addServiceDetails(pdf, inv)
	s.addPricingSummary(pdf, inv)
	s.addFooter(pdf)

	filename := fmt.Sprintf("%s_%d.pdf", inv.InvoiceNumber, time.Now().Unix())
	pdfPath := filepath.Join(s.pdfDir, filename)

	err := pdf.OutputFileAndClose(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return pdfPath, nil
}

func (s *PDFService) addHeader(pdf *gofpdf.Fpdf, inv *domain.Invoice) {
	pdf.SetFillColor(44, 62, 80)
	pdf.Rect(0, 0, 210, 40, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 28)
	pdf.SetY(15)
	pdf.CellFormat(0, 10, "INVOICE", "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetY(28)
	pdf.CellFormat(0, 5, 
		fmt.Sprintf("Invoice No: %s | Date: %s", 
			inv.InvoiceNumber, 
			inv.InvoiceDate.Format("02-Jan-2006")), 
		"", 1, "C", false, 0, "")
}

func (s *PDFService) addCompanyCustomerInfo(pdf *gofpdf.Fpdf, inv *domain.Invoice) {
	pdf.SetY(50)
	pdf.SetTextColor(0, 0, 0)

	pdf.SetFillColor(248, 249, 250)
	pdf.Rect(10, 50, 90, 40, "F")
	
	pdf.SetXY(10, 50)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(44, 62, 80)
	pdf.CellFormat(90, 8, "Service Provider", "", 1, "L", false, 0, "")

	pdf.SetX(10)
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(90, 6, fmt.Sprintf("Company: %s", inv.ProviderInfo.Name), "", 1, "L", false, 0, "")
	pdf.SetX(10)
	pdf.CellFormat(90, 6, fmt.Sprintf("Phone: %s", inv.ProviderInfo.Phone), "", 1, "L", false, 0, "")
	pdf.SetX(10)
	pdf.MultiCell(90, 6, fmt.Sprintf("Address: %s", inv.ProviderInfo.Address), "", "L", false)
	pdf.SetX(10)
	pdf.MultiCell(90, 6, fmt.Sprintf("Email %s", inv.ProviderInfo.Email), "", "L", false)

	pdf.SetFillColor(248, 249, 250)
	pdf.Rect(110, 50, 90, 40, "F")
	
	pdf.SetXY(110, 50)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(44, 62, 80)
	pdf.CellFormat(90, 8, "Customer Details", "", 1, "L", false, 0, "")

	pdf.SetX(110)
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(90, 6, fmt.Sprintf("Name: %s", inv.CustomerInfo.Name), "", 1, "L", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(90, 6, fmt.Sprintf("Phone: %s", inv.CustomerInfo.Phone), "", 1, "L", false, 0, "")
	pdf.SetX(110)
	pdf.CellFormat(90, 6, fmt.Sprintf("Email: %s", inv.CustomerInfo.Email), "", 1, "L", false, 0, "")
	if inv.ServiceDate != nil {
		pdf.SetX(110)
		pdf.CellFormat(90, 6, fmt.Sprintf("Service Date: %s", inv.ServiceDate.Format("02-Jan-2006")), "", 1, "L", false, 0, "")
	}
}

func (s *PDFService) addVehicleInfo(pdf *gofpdf.Fpdf, inv *domain.Invoice) {
	pdf.SetY(100)
	pdf.SetFillColor(236, 240, 241)
	pdf.Rect(10, 100, 190, 35, "F")

	pdf.SetXY(10, 100)
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(44, 62, 80)
	pdf.CellFormat(190, 8, "Vehicle Information", "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)

	pdf.SetXY(10, 110)
	pdf.CellFormat(63, 6, fmt.Sprintf("Brand: %s", inv.VehicleDetails.Brand), "", 0, "L", false, 0, "")
	pdf.CellFormat(63, 6, fmt.Sprintf("Model: %s", inv.VehicleDetails.Model), "", 0, "L", false, 0, "")
	pdf.CellFormat(64, 6, fmt.Sprintf("Year: %d", inv.VehicleDetails.Year), "", 1, "L", false, 0, "")

	pdf.SetX(10)
	pdf.CellFormat(63, 6, fmt.Sprintf("Type: %s", inv.VehicleDetails.VehicleType), "", 0, "L", false, 0, "")
	pdf.CellFormat(63, 6, fmt.Sprintf("Number: %s", inv.VehicleDetails.VehicleNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(64, 6, fmt.Sprintf("Fuel: %s", inv.VehicleDetails.FuelType), "", 1, "L", false, 0, "")
}

func (s *PDFService) addServiceDetails(pdf *gofpdf.Fpdf, inv *domain.Invoice) {
	yPos := pdf.GetY() + 10
	pdf.SetY(yPos)
	
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(52, 73, 94)
	pdf.SetTextColor(255, 255, 255)

	pdf.CellFormat(70, 10, "Service Type", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, "Status", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, "Payment", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, "Amount", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFillColor(248, 249, 250)

	pdf.CellFormat(70, 10, inv.ServiceInfo.Type, "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 10, string(inv.ServiceInfo.Status), "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, string(inv.ServiceInfo.PaymentStatus), "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 10, fmt.Sprintf("Rs %.2f", inv.PricingDeatils.ServiceCharge), "1", 1, "R", true, 0, "")
}

func (s *PDFService) addPricingSummary(pdf *gofpdf.Fpdf, inv *domain.Invoice) {
	pdf.SetY(pdf.GetY() + 10)

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(150, 6, "Subtotal:", "", 0, "R", false, 0, "")
	pdf.CellFormat(40, 6, fmt.Sprintf("Rs %.2f", inv.PricingDeatils.SubTotal), "", 1, "R", false, 0, "")

	pdf.CellFormat(150, 6, "GST (18%):", "", 0, "R", false, 0, "")
	pdf.CellFormat(40, 6, fmt.Sprintf("Rs %.2f", inv.PricingDeatils.GST), "", 1, "R", false, 0, "")

	if inv.PricingDeatils.Discount > 0 {
		pdf.CellFormat(150, 6, "Discount:", "", 0, "R", false, 0, "")
		pdf.CellFormat(40, 6, fmt.Sprintf("- Rs %.2f", inv.PricingDeatils.Discount), "", 1, "R", false, 0, "")
	}

	pdf.SetFont("Arial", "B", 12)
	pdf.SetDrawColor(44, 62, 80)
	pdf.SetLineWidth(0.5)
	pdf.Line(10, pdf.GetY()+2, 200, pdf.GetY()+2)
	pdf.Ln(5)

	pdf.CellFormat(150, 8, "Total Amount:", "", 0, "R", false, 0, "")
	pdf.CellFormat(40, 8, fmt.Sprintf("Rs %.2f", inv.PricingDeatils.Total), "", 1, "R", false, 0, "")
}

func (s *PDFService) addFooter(pdf *gofpdf.Fpdf) {
	pdf.SetY(270)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(127, 140, 141)
	pdf.CellFormat(0, 5, "Thank you for your business!", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, "This is a computer-generated invoice and does not require a signature.", "", 1, "C", false, 0, "")
}