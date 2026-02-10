package service

import (
	"context"
	// "fmt"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func (s *InvoiceService) convertHTMLToPDFAndUpload(
	htmlPath string,
) (string, error) {

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath

	var pdfBuf []byte
	var err error

	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			pdfBuf, _, err = page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).
				WithPaperHeight(11.69).
				WithMarginTop(0).
				WithMarginBottom(0).
				WithMarginLeft(0).
				WithMarginRight(0).
				Do(ctx)
			return err
		}),
	)

	if err != nil {
		return "", err
	}

	fileName := filepath.Base(htmlPath)
	fileName = fileName[:len(fileName)-5] + ".pdf"
// 
	// ✅ USE YOUR WRAPPER — NOT AWS SDK
	return s.s3Uploader.UploadInvoicePDF(
		pdfBuf,
		fileName, // or better: inv.ServiceID when you call this
	)
}
