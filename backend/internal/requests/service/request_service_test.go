package service_test

import (
	"context"
	"testing"

	"github.com/Firakef1/settle/backend/internal/requests/dto"
	"github.com/Firakef1/settle/backend/internal/requests/repository"
	"github.com/Firakef1/settle/backend/internal/requests/service"
)

// mockUserProvider is a simple mock for tests.
type mockUserProvider struct{}

func (m *mockUserProvider) GetUserBasicInfo(ctx context.Context, userID string) (dto.RequesterResponse, error) {
	return dto.RequesterResponse{ID: userID, Name: "Test User", Email: "test@example.com"}, nil
}

func TestRequestService_Create(t *testing.T) {
	reqRepo := repository.NewRequestRepo(nil)
	recRepo := repository.NewReceiptRepo(nil)
	comRepo := repository.NewCommentRepo(nil)

	srv := service.NewRequestService(reqRepo, recRepo, comRepo, &mockUserProvider{})

	ctx := context.Background()
	orgID := "org-1"
	userID := "user-1"

	// Test case 1: Create reimbursement (starts as draft)
	req1, err := srv.Create(ctx, orgID, userID, dto.CreateRequestDTO{
		Type:    "reimbursement",
		Amount:  100.50,
		Purpose: "Travel expense",
		Urgency: "routine",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req1.Status != "draft" {
		t.Errorf("expected status 'draft', got '%s'", req1.Status)
	}

	// Test case 2: Create advance (starts as pending)
	req2, err := srv.Create(ctx, orgID, userID, dto.CreateRequestDTO{
		Type:    "advance",
		Amount:  500.00,
		Purpose: "Upcoming conference",
		Urgency: "urgent",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req2.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", req2.Status)
	}
	if req2.SubmittedAt == nil {
		t.Errorf("expected submitted_at to be set for pending advance")
	}
}

func TestRequestService_Submit(t *testing.T) {
	reqRepo := repository.NewRequestRepo(nil)
	recRepo := repository.NewReceiptRepo(nil)
	comRepo := repository.NewCommentRepo(nil)

	srv := service.NewRequestService(reqRepo, recRepo, comRepo, &mockUserProvider{})

	ctx := context.Background()
	orgID := "org-1"
	userID := "user-1"

	// Create a reimbursement request
	req1, _ := srv.Create(ctx, orgID, userID, dto.CreateRequestDTO{
		Type:    "reimbursement",
		Amount:  100.50,
		Purpose: "Travel expense",
		Urgency: "routine",
	})

	// Try to submit without receipt
	_, err := srv.Submit(ctx, orgID, userID, req1.ID)
	if err != service.ErrReceiptRequired {
		t.Errorf("expected ErrReceiptRequired, got %v", err)
	}

	// Attach a receipt
	_, err = srv.UploadReceipt(ctx, orgID, userID, req1.ID, "/path/to/receipt.jpg")
	if err != nil {
		t.Fatalf("failed to attach receipt: %v", err)
	}

	// Submit again
	req1Submitted, err := srv.Submit(ctx, orgID, userID, req1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req1Submitted.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", req1Submitted.Status)
	}
}
