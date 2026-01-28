package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func ConvertHTMLToPDF(htmlPath string) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if _, err := os.ReadFile(htmlPath); err != nil {
		return "", fmt.Errorf("failed to read HTML: %w", err)
	}

	pdfName := filepath.Base(htmlPath)
	pdfName = pdfName[:len(pdfName)-5] + ".pdf"
	pdfPath := filepath.Join("internal/storage/invoices", pdfName)

	os.MkdirAll(filepath.Dir(pdfPath), 0755)

	absPath, _ := filepath.Abs(htmlPath)
	fileURL := "file://" + absPath

	var buf []byte
	var err error

	err = chromedp.Run(ctx,
		chromedp.Navigate(fileURL),
		chromedp.Sleep(2*time.Second),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err = page.PrintToPDF().
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
		return "", fmt.Errorf("chromedp failed: %w", err)
	}

	if err := os.WriteFile(pdfPath, buf, 0644); err != nil {
		return "", fmt.Errorf("failed to write PDF: %w", err)
	}

	return pdfName, nil
}
