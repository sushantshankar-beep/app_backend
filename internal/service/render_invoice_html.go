package service

import (
	"app_backend/internal/domain"
	"html/template"
	"log"
	"os"

	"path/filepath"
	"fmt"
)

func RenderInvoiceHTML(inv *domain.Invoice) (string, error) {

	baseDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	tmplPath := filepath.Join(baseDir, "internal/template/invoice.html")
	outDir := filepath.Join(baseDir, "internal/storage/invoices")

	// ensure directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	filename := inv.InvoiceNumber + ".html"
	htmlPath := filepath.Join(outDir, filename)

	log.Println("📄 invoice html path:", htmlPath)

	f, err := os.Create(htmlPath)
	if err != nil {
		return "", fmt.Errorf("create html failed: %w", err)
	}
	defer f.Close()

	err = tmpl.Execute(f, map[string]any{
		"InvoiceNumber": inv.InvoiceNumber,
		"InvoiceDate":   inv.InvoiceDate.Format("02 Jan 2006"),
		"Company":       inv.CompanyInfo,
		"Customer":      inv.CustomerInfo,
		"Vehicle": map[string]any{
			"Type":   inv.VehicleDetails.VehicleType,
			"Number": inv.VehicleDetails.VehicleNumber,
			"Brand":  inv.VehicleDetails.Brand,
			"Model":  inv.VehicleDetails.Model,
			"Year":   inv.VehicleDetails.Year,
			"Fuel":   inv.VehicleDetails.FuelType,
		},
		"Service": map[string]interface{}{
			"ID":            inv.ServiceNumber,
			"Type":          inv.ServiceInfo.Type,
			"Status":        inv.ServiceInfo.Status,
			"PaymentStatus": inv.ServiceInfo.PaymentStatus,
			"PaymentMode": inv.Transaction.PaymentMode,
		},
		"Pricing": inv.PricingDeatils,
	})

	if err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

	return htmlPath, nil
}

