package domain_test

import (
	"context"
	"testing"

	"clinic-queue/internal/core/domain"
)

func TestClientMetadata_Context(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		meta       *domain.ClientMetadata
		wantFound  bool
		expectedIP string
		expectedUA string
		expectedID string
	}{
		{
			name: "Successfully store and retrieve ClientMetadata",
			ctx:  context.Background(),
			meta: &domain.ClientMetadata{
				ClientIP:  "192.168.1.100",
				UserAgent: "Mozilla/5.0 Chrome/128.0",
				RequestID: "req-123-abc",
			},
			wantFound:  true,
			expectedIP: "192.168.1.100",
			expectedUA: "Mozilla/5.0 Chrome/128.0",
			expectedID: "req-123-abc",
		},
		{
			name:      "Retrieve from context without metadata",
			ctx:       context.Background(),
			meta:      nil,
			wantFound: false,
		},
		{
			name:      "GetClientMetadata with nil context",
			ctx:       nil,
			meta:      nil,
			wantFound: false,
		},
		{
			name: "WithClientMetadata with nil context creates background context",
			ctx:  nil,
			meta: &domain.ClientMetadata{
				ClientIP:  "10.0.0.1",
				UserAgent: "PostmanRuntime/7.39.0",
				RequestID: "01919df4-8e3b-7412-a1f9-90b567c9e101",
			},
			wantFound:  true,
			expectedIP: "10.0.0.1",
			expectedUA: "PostmanRuntime/7.39.0",
			expectedID: "01919df4-8e3b-7412-a1f9-90b567c9e101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testCtx context.Context = tt.ctx
			if tt.meta != nil {
				testCtx = domain.WithClientMetadata(tt.ctx, *tt.meta)
			}

			retrieved, ok := domain.GetClientMetadata(testCtx)
			if ok != tt.wantFound {
				t.Fatalf("expected found=%v, got %v", tt.wantFound, ok)
			}

			if tt.wantFound {
				if retrieved.ClientIP != tt.expectedIP {
					t.Errorf("expected ClientIP %q, got %q", tt.expectedIP, retrieved.ClientIP)
				}
				if retrieved.UserAgent != tt.expectedUA {
					t.Errorf("expected UserAgent %q, got %q", tt.expectedUA, retrieved.UserAgent)
				}
				if retrieved.RequestID != tt.expectedID {
					t.Errorf("expected RequestID %q, got %q", tt.expectedID, retrieved.RequestID)
				}
			}
		})
	}
}
