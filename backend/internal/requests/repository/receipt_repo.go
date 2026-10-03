package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/Firakef1/settle/backend/internal/requests/model"
)

type ReceiptRepository interface {
	CreateReceipt(ctx context.Context, receipt *model.Receipt) error
	GetByRequestID(ctx context.Context, requestID string) ([]model.Receipt, error)
	CopyReceipts(ctx context.Context, oldRequestID, newRequestID string) error
	CreateReceiptTx(ctx context.Context, tx *sql.Tx, receipt *model.Receipt) error
	CopyReceiptsTx(ctx context.Context, tx *sql.Tx, oldRequestID, newRequestID string) error
	GetReceiptByID(ctx context.Context, receiptID string) (*model.Receipt, error)
	UpdateOCRStatus(ctx context.Context, receiptID, status string, results *string) error
}

type ReceiptRepo struct {
	db             *sql.DB
	memoryReceipts map[string]*model.Receipt // id -> receipt
	mu             sync.RWMutex
}

func NewReceiptRepo(db *sql.DB) *ReceiptRepo {
	return &ReceiptRepo{
		db:             db,
		memoryReceipts: make(map[string]*model.Receipt),
	}
}

func (r *ReceiptRepo) CreateReceipt(ctx context.Context, receipt *model.Receipt) error {
	if r.db != nil {
		query := `
			INSERT INTO receipts (id, request_id, file_path, ocr_status, ocr_results, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := r.db.ExecContext(ctx, query,
			receipt.ID,
			receipt.RequestID,
			receipt.FilePath,
			receipt.OCRStatus,
			receipt.OCRResults,
			receipt.CreatedAt,
		)
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	c := *receipt
	r.memoryReceipts[receipt.ID] = &c
	return nil
}

func (r *ReceiptRepo) CreateReceiptTx(ctx context.Context, tx *sql.Tx, receipt *model.Receipt) error {
	if tx != nil {
		query := `
			INSERT INTO receipts (id, request_id, file_path, ocr_status, ocr_results, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := tx.ExecContext(ctx, query,
			receipt.ID,
			receipt.RequestID,
			receipt.FilePath,
			receipt.OCRStatus,
			receipt.OCRResults,
			receipt.CreatedAt,
		)
		return err
	}
	return r.CreateReceipt(ctx, receipt)
}

func (r *ReceiptRepo) GetByRequestID(ctx context.Context, requestID string) ([]model.Receipt, error) {
	if r.db != nil {
		query := `
			SELECT id, request_id, file_path, ocr_status, ocr_results, created_at
			FROM receipts
			WHERE request_id = $1
			ORDER BY created_at ASC`
		rows, err := r.db.QueryContext(ctx, query, requestID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var receipts []model.Receipt
		for rows.Next() {
			var rec model.Receipt
			if err := rows.Scan(&rec.ID, &rec.RequestID, &rec.FilePath, &rec.OCRStatus, &rec.OCRResults, &rec.CreatedAt); err != nil {
				return nil, err
			}
			receipts = append(receipts, rec)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return receipts, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	var receipts []model.Receipt
	for _, rec := range r.memoryReceipts {
		if rec.RequestID == requestID {
			receipts = append(receipts, *rec)
		}
	}
	return receipts, nil
}

func (r *ReceiptRepo) CopyReceipts(ctx context.Context, oldRequestID, newRequestID string) error {
	if r.db != nil {
		query := `
			INSERT INTO receipts (id, request_id, file_path, ocr_status, ocr_results, created_at)
			SELECT gen_random_uuid(), $2, file_path, ocr_status, ocr_results, $3
			FROM receipts
			WHERE request_id = $1`
		_, err := r.db.ExecContext(ctx, query, oldRequestID, newRequestID, time.Now())
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rec := range r.memoryReceipts {
		if rec.RequestID == oldRequestID {
			newRec := *rec
			newRec.ID = "mem-uuid-" + rec.ID // pseudo uuid for memory
			newRec.RequestID = newRequestID
			newRec.CreatedAt = time.Now()
			r.memoryReceipts[newRec.ID] = &newRec
		}
	}
	return nil
}

func (r *ReceiptRepo) CopyReceiptsTx(ctx context.Context, tx *sql.Tx, oldRequestID, newRequestID string) error {
	if tx != nil {
		query := `
			INSERT INTO receipts (id, request_id, file_path, ocr_status, ocr_results, created_at)
			SELECT gen_random_uuid(), $2, file_path, ocr_status, ocr_results, $3
			FROM receipts
			WHERE request_id = $1`
		_, err := tx.ExecContext(ctx, query, oldRequestID, newRequestID, time.Now())
		return err
	}
	return r.CopyReceipts(ctx, oldRequestID, newRequestID)
}

func (r *ReceiptRepo) GetReceiptByID(ctx context.Context, receiptID string) (*model.Receipt, error) {
	if r.db != nil {
		query := `SELECT id, request_id, file_path, ocr_status, ocr_results, created_at FROM receipts WHERE id = $1`
		var rec model.Receipt
		err := r.db.QueryRowContext(ctx, query, receiptID).Scan(&rec.ID, &rec.RequestID, &rec.FilePath, &rec.OCRStatus, &rec.OCRResults, &rec.CreatedAt)
		if err != nil {
			return nil, err
		}
		return &rec, nil
	}
	return nil, errors.New("not found")
}

func (r *ReceiptRepo) UpdateOCRStatus(ctx context.Context, receiptID, status string, results *string) error {
	if r.db != nil {
		query := `UPDATE receipts SET ocr_status = $1, ocr_results = $2 WHERE id = $3`
		_, err := r.db.ExecContext(ctx, query, status, results, receiptID)
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, exists := r.memoryReceipts[receiptID]; exists {
		rec.OCRStatus = status
		rec.OCRResults = results
		return nil
	}
	return errors.New("not found")
}
