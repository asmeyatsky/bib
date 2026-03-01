package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bibbank/bib/services/identity-service/internal/domain/port"
	"github.com/bibbank/bib/services/identity-service/internal/domain/valueobject"
	"github.com/bibbank/bib/services/identity-service/internal/infrastructure/provider"
)

func TestPersonaClient_InitiateCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/inquiries", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id": "inq_abc123",
				"attributes": map[string]interface{}{
					"status": "created",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := provider.NewPersonaClient("test-api-key", server.URL)

	ref, err := client.InitiateCheck(context.Background(), valueobject.CheckTypeDocument, port.ApplicantInfo{
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "john@example.com",
		DateOfBirth: "1990-01-01",
		Country:     "US",
	})

	require.NoError(t, err)
	assert.Equal(t, "inq_abc123", ref)
}

func TestPersonaClient_GetCheckResult_Approved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/inquiries/inq_abc123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"id": "inq_abc123",
				"attributes": map[string]interface{}{
					"status": "approved",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := provider.NewPersonaClient("test-api-key", server.URL)

	status, reason, err := client.GetCheckResult(context.Background(), "inq_abc123")

	require.NoError(t, err)
	assert.True(t, status.Equal(valueobject.StatusApproved))
	assert.Empty(t, reason)
}

func TestPersonaClient_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":[{"title":"Unauthorized"}]}`))
	}))
	defer server.Close()

	client := provider.NewPersonaClient("bad-key", server.URL)

	_, err := client.InitiateCheck(context.Background(), valueobject.CheckTypeDocument, port.ApplicantInfo{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "persona API error (status 401)")
}
