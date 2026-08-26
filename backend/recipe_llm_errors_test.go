package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteRecipeParseErrorMapping(t *testing.T) {
	tests := []struct {
		kind       RecipeLLMErrorKind
		status     int
		retryAfter int
	}{
		{LLMErrorAuth, http.StatusServiceUnavailable, 0},
		{LLMErrorQuota, http.StatusTooManyRequests, 17},
		{LLMErrorTimeout, http.StatusGatewayTimeout, 0},
		{LLMErrorUpstream, http.StatusBadGateway, 0},
		{LLMErrorInvalid, http.StatusUnprocessableEntity, 0},
	}
	for _, tc := range tests {
		t.Run(string(tc.kind), func(t *testing.T) {
			rr := httptest.NewRecorder()
			writeRecipeParseError(rr, &RecipeLLMError{Kind: tc.kind, RetryAfterSeconds: tc.retryAfter})
			require.Equal(t, tc.status, rr.Code)
			if tc.retryAfter > 0 {
				require.Equal(t, "17", rr.Header().Get("Retry-After"))
			}
			require.NotContains(t, rr.Body.String(), "token")
		})
	}
}
