package utils

import (
	"app_backend/internal/domain"
	"bytes"
	"fmt"
	"strings"
    "time"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

func GeneratePDF(html string) ([]byte, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	pdfg.AddPage(wkhtmltopdf.NewPageReader(strings.NewReader(html)))

	var buf bytes.Buffer
	pdfg.SetOutput(&buf)

	if err := pdfg.Create(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GenerateAgreementPDF(provider *domain.Provider, agreement *domain.Agreement) ([]byte, error) {
	var htmlBuilder strings.Builder

	styles := `
		<style>
			body { font-family: 'Arial', sans-serif; font-size: 12px; line-height: 1.5; color: #333; margin: 0; padding: 20px; }
			h1, h2 { text-align: center; font-weight: bold; margin-bottom: 20px; }
			h1 { font-size: 18px; text-transform: uppercase; }
			h2 { font-size: 16px; margin-top: 30px; text-decoration: underline; }
			p { margin-bottom: 15px; text-align: justify; }
			.section-title { font-weight: bold; margin-top: 15px; margin-bottom: 5px; }
			.paragraph-container { margin-bottom: 15px; }
			.paragraph-content { text-align: justify; }
			.bold { font-weight: bold; }
			.input-field { border-bottom: 1px solid #000; padding: 0 5px; display: inline-block; min-width: 150px; }
			ul { padding-left: 20px; margin-bottom: 15px; }
			li { margin-bottom: 5px; }
			table { width: 100%; border-collapse: collapse; margin-top: 10px; }
			td { padding: 5px; vertical-align: top; }
			.commercial-term-box { background-color: #f9f9f9; padding: 15px; border: 1px solid #ddd; margin-top: 20px; border-radius: 5px; }
			.term-label { font-weight: bold; display: block; margin-bottom: 5px; }
			.term-value { background-color: #fff; border: 1px solid #ccc; padding: 8px; border-radius: 4px; display: block; width: 100%; box-sizing: border-box; }
			.note { font-size: 11px; margin-top: 10px; font-style: italic; }
		</style>
	`

	htmlBuilder.WriteString("<html><head>")
	htmlBuilder.WriteString(styles)
	htmlBuilder.WriteString("</head><body>")

	htmlBuilder.WriteString(fmt.Sprintf("<h1>%s</h1>", agreement.Title))

	htmlBuilder.WriteString(fmt.Sprintf("<p><strong>BETWEEN:</strong></p>"))
	htmlBuilder.WriteString(fmt.Sprintf("<p>%s</p>", agreement.AgreementOf))
	htmlBuilder.WriteString(fmt.Sprintf("<p><strong>AND</strong></p>"))

	agreementDate := time.Now().Format("02-01-2006 15:04:05")
	if provider.AgreementSubmittedAt != nil {
		agreementDate = provider.AgreementSubmittedAt.Format("02-01-2006 15:04:05")
	}

	providerDetails := fmt.Sprintf(`
		<p><strong>henceforth known as the Service Provider:</strong></p>
		<div style="background-color: #f0f0f0; padding: 15px; border-radius: 5px; margin-bottom: 20px;">
			<table style="width: 100%%;">
				<tr>
					<td style="width: 150px;"><strong>Provider Name:</strong></td>
					<td>%s</td>
				</tr>
				<tr>
					<td><strong>Company Name:</strong></td>
					<td>%s</td>
				</tr>
				<tr>
					<td><strong>City/Place:</strong></td>
					<td>%s</td>
				</tr>
				<tr>
					<td><strong>Date & Time:</strong></td>
					<td>%s</td>
				</tr>
			</table>
		</div>
	`, provider.Name, provider.CompanyName, provider.City, agreementDate)

	htmlBuilder.WriteString(providerDetails)

	for _, p := range agreement.Paragraphs {
		content := p.Content

		content = strings.ReplaceAll(content, "{provider.name}", provider.Name)
		content = strings.ReplaceAll(content, "{provider.companyName}", provider.CompanyName)
		content = strings.ReplaceAll(content, "{provider.city}", provider.City)
		content = strings.ReplaceAll(content, "{provider.dateTime}",agreementDate)

		htmlBuilder.WriteString("<div class='paragraph-container'>")

		titlePart := ""
		if p.Title != "" {
			titlePart = fmt.Sprintf("<strong>%s</strong> ", p.Title)
		}

		numberPart := ""
		if p.Number != "" {
			numberPart = fmt.Sprintf("<strong>%s</strong> ", p.Number)
		}

		htmlBuilder.WriteString(fmt.Sprintf("<div class='paragraph-content'>%s%s%s</div>", numberPart, titlePart, content))
		htmlBuilder.WriteString("</div>")
	}

	if agreement.Annexures != "" {
		htmlBuilder.WriteString(fmt.Sprintf("<h2>%s</h2>", "Annexure - A"))
		htmlBuilder.WriteString("<div class='commercial-term-box'>")
		htmlBuilder.WriteString("<div class='section-title'>COMMERCIAL TERMS</div>")

		htmlBuilder.WriteString("<div style='margin-bottom: 15px;'>")
		htmlBuilder.WriteString("<span class='term-label'>Marketplace Fee Per booking received from Vahanwire</span>")
		commissionStr := fmt.Sprintf("%.2f%%", provider.CommissionPercentage)
		if provider.CommissionPercentage == 0 {
			commissionStr = "To be determined"
		}
		htmlBuilder.WriteString(fmt.Sprintf("<span class='term-value'>%s</span>", commissionStr))
		htmlBuilder.WriteString("</div>")

		htmlBuilder.WriteString("<div style='margin-bottom: 15px;'>")
		htmlBuilder.WriteString("<span class='term-label'>Payment Gateway Charges</span>")
		htmlBuilder.WriteString("<span class='term-label'>As Applicable</span>")
		htmlBuilder.WriteString("<span class='note'>This will be charged back to back</span>")
		htmlBuilder.WriteString("</div>")

		if agreement.AdditionalNotes != "" {
			htmlBuilder.WriteString(fmt.Sprintf("<p class='note'><strong>Please Note:</strong> %s</p>", agreement.AdditionalNotes))
		}
		htmlBuilder.WriteString("</div>")
	}

	htmlBuilder.WriteString("</body></html>")

	return GeneratePDF(htmlBuilder.String())
}
