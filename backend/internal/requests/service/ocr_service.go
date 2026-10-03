package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/Firakef1/settle/backend/internal/shared/config"
)

type OCRResponse struct {
	ParsedResults []struct {
		ParsedText   string `json:"ParsedText"`
		ErrorMessage string `json:"ErrorMessage"`
	} `json:"ParsedResults"`
	OCRExitCode             int    `json:"OCRExitCode"`
	IsErroredOnProcessing   bool   `json:"IsErroredOnProcessing"`
	ErrorMessage            string `json:"ErrorMessage"`
}

type ExtractedData struct {
	Amount   string `json:"amount"`
	Date     string `json:"date"`
	Merchant string `json:"merchant"`
}

// ProcessOCRBackground reads the file and calls the OCR.Space API, then updates the receipt.
func (s *RequestService) ProcessOCRBackground(receiptID, filePath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Printf("[OCR] Starting background job for receipt %s", receiptID)

	apiKey := config.AppConfig.OCRApiKey
	if apiKey == "" {
		apiKey = "helloworld" // Free fallback key
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[OCR] Failed to open file: %v", err)
		s.markOCRFailed(ctx, receiptID, "File unreadable")
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		log.Printf("[OCR] Failed to create form file: %v", err)
		s.markOCRFailed(ctx, receiptID, "Multipart error")
		return
	}
	_, err = io.Copy(part, file)
	if err != nil {
		log.Printf("[OCR] Failed to copy file: %v", err)
		s.markOCRFailed(ctx, receiptID, "File read error")
		return
	}

	writer.WriteField("apikey", apiKey)
	writer.WriteField("language", "eng")
	writer.WriteField("isOverlayRequired", "false")
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.ocr.space/parse/image", body)
	if err != nil {
		log.Printf("[OCR] Request creation failed: %v", err)
		s.markOCRFailed(ctx, receiptID, "Request error")
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[OCR] API call failed: %v", err)
		s.markOCRFailed(ctx, receiptID, "API network error")
		return
	}
	defer resp.Body.Close()

	var ocrResp OCRResponse
	if err := json.NewDecoder(resp.Body).Decode(&ocrResp); err != nil {
		log.Printf("[OCR] Decode failed: %v", err)
		s.markOCRFailed(ctx, receiptID, "API decode error")
		return
	}

	if ocrResp.IsErroredOnProcessing {
		log.Printf("[OCR] API error: %s", ocrResp.ErrorMessage)
		s.markOCRFailed(ctx, receiptID, "API processing error")
		return
	}

	var parsedText string
	if len(ocrResp.ParsedResults) > 0 {
		parsedText = ocrResp.ParsedResults[0].ParsedText
	}

	// Simple heuristic extraction
	amountRe := regexp.MustCompile(`(?i)(?:total|amount)[\s:]*[\$£€]?\s*(\d+\.\d{2})`)
	dateRe := regexp.MustCompile(`(\d{1,2}[-/]\d{1,2}[-/]\d{2,4})`)

	amount := ""
	if m := amountRe.FindStringSubmatch(parsedText); len(m) > 1 {
		amount = m[1]
	}

	date := ""
	if m := dateRe.FindStringSubmatch(parsedText); len(m) > 1 {
		date = m[1]
	}

	merchant := "Unknown Merchant"
	// Extremely naive merchant grab (first line usually)
	lines := regexp.MustCompile(`\r?\n`).Split(parsedText, -1)
	for _, l := range lines {
		if l != "" && len(l) > 3 {
			merchant = l
			break
		}
	}

	extracted := ExtractedData{
		Amount:   amount,
		Date:     date,
		Merchant: merchant,
	}
	resultsBytes, _ := json.Marshal(extracted)
	resultsStr := string(resultsBytes)

	// Update DB using raw SQL via the recRepo's db if we don't have a specific method.
	// We'll just call an update method on recRepo.
	if err := s.recRepo.UpdateOCRStatus(ctx, receiptID, "completed", &resultsStr); err != nil {
		log.Printf("[OCR] DB update failed: %v", err)
	} else {
		log.Printf("[OCR] Successfully processed receipt %s", receiptID)
	}
}

func (s *RequestService) markOCRFailed(ctx context.Context, receiptID, reason string) {
	errBytes, _ := json.Marshal(map[string]string{"error": reason})
	errStr := string(errBytes)
	s.recRepo.UpdateOCRStatus(ctx, receiptID, "failed", &errStr)
}
