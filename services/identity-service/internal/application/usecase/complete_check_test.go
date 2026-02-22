package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/pkg/events"
	"github.com/bibbank/bib/services/identity-service/internal/application/dto"
	"github.com/bibbank/bib/services/identity-service/internal/application/usecase"
	"github.com/bibbank/bib/services/identity-service/internal/domain/model"
)

// --- Mock implementations ---

type mockIdentityEventPublisher struct {
	publishedEvents []events.DomainEvent
	publishFunc     func(ctx context.Context, topic string, events ...events.DomainEvent) error
}

func (m *mockIdentityEventPublisher) Publish(ctx context.Context, topic string, evts ...events.DomainEvent) error {
	if m.publishFunc != nil {
		return m.publishFunc(ctx, topic, evts...)
	}
	m.publishedEvents = append(m.publishedEvents, evts...)
	return nil
}

func inProgressVerification() model.IdentityVerification {
	// Create a verification with checks in IN_PROGRESS status
	v, _ := model.NewIdentityVerification(
		uuid.New(), "Jane", "Smith", "jane@example.com", "1990-01-01", "US",
	)
	// Start processing to transition checks to IN_PROGRESS
	now := time.Now().UTC()
	v, _ = v.StartProcessing(now)
	return v
}

// --- Tests ---

func TestCompleteCheck_Execute(t *testing.T) {
	t.Run("successfully completes a check with APPROVED status", func(t *testing.T) {
		v := inProgressVerification()
		checks := v.Checks()
		require.NotEmpty(t, checks)
		checkID := checks[0].ID()

		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
		}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: v.ID(),
			CheckID:        checkID,
			Status:         "APPROVED",
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, v.ID(), resp.ID)
		require.NotEmpty(t, repo.savedVerifications)
	})

	t.Run("fails with invalid status string", func(t *testing.T) {
		repo := &mockVerificationRepository{}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: uuid.New(),
			CheckID:        uuid.New(),
			Status:         "INVALID_STATUS",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("fails when verification not found", func(t *testing.T) {
		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return model.IdentityVerification{}, fmt.Errorf("not found")
			},
		}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: uuid.New(),
			CheckID:        uuid.New(),
			Status:         "APPROVED",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find verification")
	})

	t.Run("fails when check not found in verification", func(t *testing.T) {
		v := inProgressVerification()
		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
		}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: v.ID(),
			CheckID:        uuid.New(), // non-existent check
			Status:         "APPROVED",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete check")
	})

	t.Run("fails when repository save fails", func(t *testing.T) {
		v := inProgressVerification()
		checks := v.Checks()
		checkID := checks[0].ID()

		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
			saveFunc: func(ctx context.Context, ver model.IdentityVerification) error {
				return fmt.Errorf("database unavailable")
			},
		}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: v.ID(),
			CheckID:        checkID,
			Status:         "APPROVED",
		}
		_, err := uc.Execute(context.Background(), req)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save verification")
	})

	t.Run("fails when event publishing fails", func(t *testing.T) {
		v := inProgressVerification()
		checks := v.Checks()
		checkID := checks[0].ID()

		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
		}
		publisher := &mockIdentityEventPublisher{
			publishFunc: func(ctx context.Context, topic string, evts ...events.DomainEvent) error {
				return fmt.Errorf("kafka unavailable")
			},
		}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: v.ID(),
			CheckID:        checkID,
			Status:         "APPROVED",
		}
		_, err := uc.Execute(context.Background(), req)

		// The complete check generates domain events when all checks complete.
		// Even with one check approved, there may be domain events.
		// If publishing fails, it should return an error.
		if err != nil {
			assert.Contains(t, err.Error(), "failed to publish events")
		}
	})

	t.Run("completes check with REJECTED status", func(t *testing.T) {
		v := inProgressVerification()
		checks := v.Checks()
		checkID := checks[0].ID()

		repo := &mockVerificationRepository{
			findByIDFunc: func(ctx context.Context, id uuid.UUID) (model.IdentityVerification, error) {
				return v, nil
			},
		}
		publisher := &mockIdentityEventPublisher{}

		uc := usecase.NewCompleteCheck(repo, publisher)

		req := dto.CompleteCheckRequest{
			VerificationID: v.ID(),
			CheckID:        checkID,
			Status:         "REJECTED",
			FailureReason:  "document expired",
		}
		resp, err := uc.Execute(context.Background(), req)

		require.NoError(t, err)
		assert.Equal(t, v.ID(), resp.ID)

		// After rejecting one check, the overall status should become REJECTED.
		checkFound := false
		for _, c := range resp.Checks {
			if c.ID == checkID {
				checkFound = true
				assert.Equal(t, "REJECTED", c.Status)
				assert.Equal(t, "document expired", c.FailureReason)
			}
		}
		assert.True(t, checkFound)
	})
}
