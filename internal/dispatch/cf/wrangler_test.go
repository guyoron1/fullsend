package cf

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePreviewAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
		errMsg  string
	}{
		{"valid lowercase", "pr-42", false, ""},
		{"valid single char", "a", false, ""},
		{"valid max length 63", "a12345678901234567890123456789012345678901234567890123456789012", false, ""},
		{"empty", "", true, "empty string"},
		{"too long 64", "a123456789012345678901234567890123456789012345678901234567890123", true, "1-63 chars"},
		{"uppercase", "PR-42", true, "lowercase"},
		{"leading hyphen", "-pr42", true, "lowercase"},
		{"trailing hyphen", "pr42-", true, "lowercase"},
		{"double hyphen", "pr--42", false, ""},
		{"with underscore", "pr_42", true, "lowercase"},
		{"with period", "pr.42", true, "lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePreviewAlias(tt.alias)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateWorkerName(t *testing.T) {
	tests := []struct {
		name       string
		workerName string
		wantErr    bool
		errMsg     string
	}{
		{"valid lowercase", "mint-test", false, ""},
		{"valid single char", "w", false, ""},
		{"empty", "", true, "empty string"},
		{"too long 64", "a123456789012345678901234567890123456789012345678901234567890123", true, "1-63 chars"},
		{"uppercase", "MINT-TEST", true, "lowercase"},
		{"leading hyphen", "-mint", true, "lowercase"},
		{"trailing hyphen", "mint-", true, "lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkerName(tt.workerName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPreviewURL(t *testing.T) {
	tests := []struct {
		name       string
		alias      string
		workerName string
		subdomain  string
		want       string
	}{
		{
			name:       "standard preview",
			alias:      "pr-42",
			workerName: "mint-test",
			subdomain:  "myaccount",
			want:       "https://pr-42-mint-test.myaccount.workers.dev",
		},
		{
			name:       "single letter",
			alias:      "a",
			workerName: "w",
			subdomain:  "sub",
			want:       "https://a-w.sub.workers.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreviewURL(tt.alias, tt.workerName, tt.subdomain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProductionURL(t *testing.T) {
	tests := []struct {
		name       string
		workerName string
		subdomain  string
		want       string
	}{
		{
			name:       "standard production",
			workerName: "mint-test",
			subdomain:  "myaccount",
			want:       "https://mint-test.myaccount.workers.dev",
		},
		{
			name:       "single letter",
			workerName: "w",
			subdomain:  "sub",
			want:       "https://w.sub.workers.dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProductionURL(tt.workerName, tt.subdomain)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPreviewTeardown_NoOp(t *testing.T) {
	runner := &LiveRunner{}
	err := runner.PreviewTeardown(context.Background(), "mint-test", "pr-42")
	assert.NoError(t, err)
}

// Compile-time interface check.
var _ Runner = (*LiveRunner)(nil)
