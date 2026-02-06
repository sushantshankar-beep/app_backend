package service

import (
	"app_backend/internal/dto"
	"bytes"
	"fmt"
	"html/template"
	

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

type PDFService struct{}

func NewPDFService() *PDFService {
	return &PDFService{}
}


func (s *PDFService) GenerateAgreementPDF(
	agreement *dto.AgreementHTMLResponse,
) ([]byte, error) {

	htmlContent, err := s.renderHTMLTemplate(agreement)
	if err != nil {
		return nil, fmt.Errorf("failed to render HTML template: %w", err)
	}

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PDF generator: %w", err)
	}

	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.MarginTop.Set(20)
	pdfg.MarginBottom.Set(20)
	pdfg.MarginLeft.Set(15)
	pdfg.MarginRight.Set(15)

	page := wkhtmltopdf.NewPageReader(bytes.NewReader([]byte(htmlContent)))
	page.EnableLocalFileAccess.Set(true)

	pdfg.AddPage(page)

	if err := pdfg.Create(); err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	return pdfg.Bytes(), nil
}

func (s *PDFService) renderHTMLTemplate(
	agreement *dto.AgreementHTMLResponse,
) (string, error) {

	tmpl, err := template.New("agreement").Parse(agreementHTMLTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, agreement); err != nil {
		return "", err
	}

	return buf.String(), nil
}

var agreementHTMLTemplate = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
body {
	font-family: Georgia, serif;
	line-height: 1.8;
}
.paragraph {
	margin: 20px 0;
}
.paragraph-header {
	font-weight: bold;
}
.paragraph-content p {
	margin-bottom: 14px;
	text-align: justify;
}
</style>
</head>
<body>

<h2 style="text-align:center">{{.Title}}</h2>

<div>
<strong>Provider:</strong> {{.ProviderName}}<br>
<strong>Company:</strong> {{.CompanyName}}<br>
<strong>Date:</strong> {{.DateTime}}<br>
<strong>Place:</strong> {{.Place}}
</div>

<hr>

{{range .Paragraphs}}
<div class="paragraph">
	<div class="paragraph-header">{{.Number}}. {{.Title}}</div>
	<div class="paragraph-content">
		{{.Content}} <!-- ✅ HTML now renders -->
	</div>
</div>
{{end}}

</body>
</html>
`
