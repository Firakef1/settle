package repository

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/Firakef1/settle/backend/internal/auth/model"
)

var ErrOTPNotFound = errors.New("password reset otp not found")

// PasswordResetOTPRepository defines operations for password reset OTP storage.
type PasswordResetOTPRepository interface {
	UpsertOTP(ctx context.Context, otp *model.PasswordResetOTP) error
	FindByUserID(ctx context.Context, userID string) (*model.PasswordResetOTP, error)
	IncrementAttemptCount(ctx context.Context, userID string) (int, error)
	DeleteByUserID(ctx context.Context, userID string) error
}

// PasswordResetOTPRepo implements PasswordResetOTPRepository with SQL & memory fallback.
type PasswordResetOTPRepo struct {
	db      *sql.DB
	mu      sync.RWMutex
	memOTPs map[string]*model.PasswordResetOTP // userID -> OTP
}

// NewPasswordResetOTPRepo creates a new PasswordResetOTPRepo.
func NewPasswordResetOTPRepo(db *sql.DB) *PasswordResetOTPRepo {
	return &PasswordResetOTPRepo{
		db:      db,
		memOTPs: make(map[string]*model.PasswordResetOTP),
	}
}

func (r *PasswordResetOTPRepo) UpsertOTP(ctx context.Context, otp *model.PasswordResetOTP) error {
	if r.db != nil {
		query := `
			INSERT INTO password_reset_otps (id, user_id, otp_hash, expires_at, attempt_count, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (user_id) DO UPDATE SET
				otp_hash = EXCLUDED.otp_hash,
				expires_at = EXCLUDED.expires_at,
				attempt_count = 0,
				updated_at = EXCLUDED.updated_at`
		_, err := r.db.ExecContext(ctx, query,
			otp.ID,
			otp.UserID,
			otp.OTPHash,
			otp.ExpiresAt,
			otp.AttemptCount,
			otp.CreatedAt,
			otp.UpdatedAt,
		)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	oCopy := *otp
	oCopy.AttemptCount = 0
	r.memOTPs[otp.UserID] = &oCopy
	return nil
}

func (r *PasswordResetOTPRepo) FindByUserID(ctx context.Context, userID string) (*model.PasswordResetOTP, error) {
	if r.db != nil {
		query := `
			SELECT id, user_id, otp_hash, expires_at, attempt_count, created_at, updated_at
			FROM password_reset_otps
			WHERE user_id = $1`
		row := r.db.QueryRowContext(ctx, query, userID)

		var o model.PasswordResetOTP
		err := row.Scan(&o.ID, &o.UserID, &o.OTPHash, &o.ExpiresAt, &o.AttemptCount, &o.CreatedAt, &o.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, ErrOTPNotFound
			}
			return nil, err
		}
		return &o, nil
	}

	// Memory fallback
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, exists := r.memOTPs[userID]
	if !exists {
		return nil, ErrOTPNotFound
	}
	oCopy := *o
	return &oCopy, nil
}

func (r *PasswordResetOTPRepo) IncrementAttemptCount(ctx context.Context, userID string) (int, error) {
	if r.db != nil {
		query := `
			UPDATE password_reset_otps
			SET attempt_count = attempt_count + 1, updated_at = NOW()
			WHERE user_id = $1
			RETURNING attempt_count`
		var newCount int
		err := r.db.QueryRowContext(ctx, query, userID).Scan(&newCount)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, ErrOTPNotFound
			}
			return 0, err
		}
		return newCount, nil
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	o, exists := r.memOTPs[userID]
	if !exists {
		return 0, ErrOTPNotFound
	}
	o.AttemptCount++
	o.UpdatedAt = time.Now()
	return o.AttemptCount, nil
}

func (r *PasswordResetOTPRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if r.db != nil {
		_, err := r.db.ExecContext(ctx, `DELETE FROM password_reset_otps WHERE user_id = $1`, userID)
		return err
	}

	// Memory fallback
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.memOTPs, userID)
	return nil
}
