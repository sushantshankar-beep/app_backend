package service

import (
	"app_backend/internal/domain"
	"html/template"
	"log"
	"os"

	"path/filepath"
)

func RenderInvoiceHTML(inv *domain.Invoice) (string, error) {

	log.Println("iinvoiceeee jksdvbjhdsbjhvd")
	tmpl, err := template.ParseFiles("internal/template/invoice.html")
	if err != nil {
		return "", err
	}

	filename := inv.InvoiceNumber + ".html"
	log.Println("jdcsnbjdsjhbc",filename)
	htmlPath := filepath.Join("internal/storage/invoices", filename)
    log.Println("jdcsnbjdsjhbc",htmlPath)
	f, err := os.Create(htmlPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = tmpl.Execute(f, map[string]interface{}{
		"InvoiceNumber": inv.InvoiceNumber,
		"InvoiceDate":   inv.InvoiceDate.Format("02 Jan 2006"),
		"Company":       inv.CompanyInfo,
		"Customer":      inv.CustomerInfo,
		"Vehicle": map[string]interface{}{
			"Type":   inv.VehicleDetails.VehicleType,
			"Number": inv.VehicleDetails.VehicleNumber,
			"Brand":  inv.VehicleDetails.Brand,
			"Model":  inv.VehicleDetails.Model,
			"Year":   inv.VehicleDetails.Year,
			"Fuel":   inv.VehicleDetails.FuelType,
		},
		"Service": map[string]interface{}{
			"ID":            inv.ServiceID,
			"Type":          inv.ServiceInfo.Type,
			"Status":        inv.ServiceInfo.Status,
			"PaymentStatus": inv.ServiceInfo.PaymentStatus,
		},
		"Pricing": inv.PricingDeatils,
	})

	return htmlPath, err
}
