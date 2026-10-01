package model

import "time"

type Receipt struct {
	ID         string    `json:"id" db:"id"`
	RequestID  string    `json:"request_id" db:"request_id"`
	FilePath   string    `json:"file_path" db:"file_path"`
	OCRStatus  string    `json:"ocr_status" db:"ocr_status"`
	OCRResults *string   `json:"ocr_results,omitempty" db:"ocr_results"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
