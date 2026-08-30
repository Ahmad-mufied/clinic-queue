package domain

import (
	"errors"
	"testing"
	"time"
)

func TestAuditLog_Validate(t *testing.T) {
	tests := []struct {
		name    string
		log     *AuditLog
		wantErr error
	}{
		{
			name:    "nil audit log returns ErrInvalidInput",
			log:     nil,
			wantErr: ErrInvalidInput,
		},
		{
			name: "empty action returns ErrInvalidAction",
			log: &AuditLog{
				Action:    "   ",
				ActorName: "Doctor A",
				Role:      "doctor",
			},
			wantErr: ErrInvalidAction,
		},
		{
			name: "empty actor name returns ErrInvalidInput",
			log: &AuditLog{
				Action:    ActionAuthLogin,
				ActorName: "",
				Role:      "doctor",
			},
			wantErr: ErrInvalidInput,
		},
		{
			name: "empty role returns ErrInvalidRole",
			log: &AuditLog{
				Action:    ActionAuthLogin,
				ActorName: "Doctor A",
				Role:      "",
			},
			wantErr: ErrInvalidRole,
		},
		{
			name: "valid audit log returns nil",
			log: &AuditLog{
				Action:    ActionAuthLogin,
				ActorName: "Doctor A",
				Role:      "doctor",
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.log.Validate()
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAuditLog_Normalize(t *testing.T) {
	tests := []struct {
		name          string
		log           *AuditLog
		wantActor     string
		wantRole      string
		wantIP        string
		checkNilFirst bool
	}{
		{
			name:          "nil audit log does not panic",
			log:           nil,
			checkNilFirst: true,
		},
		{
			name: "empty fields normalized with safe defaults",
			log: &AuditLog{
				Action: ActionAuthLogin,
			},
			wantActor: DefaultAnonymousActor,
			wantRole:  DefaultFallbackRole,
			wantIP:    DefaultFallbackIP,
		},
		{
			name: "existing fields are preserved",
			log: &AuditLog{
				ActorName: "dr_smith",
				Role:      "doctor",
				IPAddress: "192.168.1.100",
				Details:   map[string]any{"key": "value"},
			},
			wantActor: "dr_smith",
			wantRole:  "doctor",
			wantIP:    "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkNilFirst {
				var nilLog *AuditLog
				nilLog.Normalize()
				return
			}

			tt.log.Normalize()

			if tt.log.ActorName != tt.wantActor {
				t.Errorf("expected ActorName %q, got %q", tt.wantActor, tt.log.ActorName)
			}
			if tt.log.Role != tt.wantRole {
				t.Errorf("expected Role %q, got %q", tt.wantRole, tt.log.Role)
			}
			if tt.log.IPAddress != tt.wantIP {
				t.Errorf("expected IPAddress %q, got %q", tt.wantIP, tt.log.IPAddress)
			}
			if tt.log.Details == nil {
				t.Errorf("expected Details map to be non-nil")
			}
		})
	}
}

func TestAuditLogFilter_NormalizePagination(t *testing.T) {
	tests := []struct {
		name          string
		filter        *AuditLogFilter
		wantPage      int
		wantLimit     int
		checkNilFirst bool
	}{
		{
			name:          "nil filter does not panic",
			filter:        nil,
			checkNilFirst: true,
		},
		{
			name: "non-positive page and limit set to defaults",
			filter: &AuditLogFilter{
				Page:  0,
				Limit: -5,
			},
			wantPage:  DefaultPage,
			wantLimit: DefaultLimit,
		},
		{
			name: "limit exceeding max limit is capped",
			filter: &AuditLogFilter{
				Page:  2,
				Limit: 250,
			},
			wantPage:  2,
			wantLimit: MaxLimit,
		},
		{
			name: "valid pagination values preserved",
			filter: &AuditLogFilter{
				Page:  3,
				Limit: 50,
			},
			wantPage:  3,
			wantLimit: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkNilFirst {
				var nilFilter *AuditLogFilter
				nilFilter.NormalizePagination()
				return
			}

			tt.filter.NormalizePagination()

			if tt.filter.Page != tt.wantPage {
				t.Errorf("expected Page %d, got %d", tt.wantPage, tt.filter.Page)
			}
			if tt.filter.Limit != tt.wantLimit {
				t.Errorf("expected Limit %d, got %d", tt.wantLimit, tt.filter.Limit)
			}
		})
	}
}

func TestAuditLogFilter_Offset(t *testing.T) {
	tests := []struct {
		name       string
		filter     *AuditLogFilter
		wantOffset int
	}{
		{
			name:       "nil filter returns 0",
			filter:     nil,
			wantOffset: 0,
		},
		{
			name: "page <= 1 returns 0",
			filter: &AuditLogFilter{
				Page:  1,
				Limit: 20,
			},
			wantOffset: 0,
		},
		{
			name: "page 0 returns 0",
			filter: &AuditLogFilter{
				Page:  0,
				Limit: 20,
			},
			wantOffset: 0,
		},
		{
			name: "page 3 with limit 20 returns 40",
			filter: &AuditLogFilter{
				Page:  3,
				Limit: 20,
			},
			wantOffset: 40,
		},
		{
			name: "page 5 with limit 15 returns 60",
			filter: &AuditLogFilter{
				Page:  5,
				Limit: 15,
			},
			wantOffset: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Offset()
			if got != tt.wantOffset {
				t.Errorf("expected offset %d, got %d", tt.wantOffset, got)
			}
		})
	}
}

func TestAuditLogFilter_NormalizeSort(t *testing.T) {
	tests := []struct {
		name      string
		filter    *AuditLogFilter
		wantOrder string
		checkNil  bool
	}{
		{
			name:     "nil filter does not panic",
			filter:   nil,
			checkNil: true,
		},
		{
			name: "empty sort order defaults to desc",
			filter: &AuditLogFilter{
				SortOrder: "",
			},
			wantOrder: "desc",
		},
		{
			name: "uppercase ASC normalized to asc",
			filter: &AuditLogFilter{
				SortOrder: "ASC",
			},
			wantOrder: "asc",
		},
		{
			name: "invalid sort order defaults to desc",
			filter: &AuditLogFilter{
				SortOrder: "random_string",
			},
			wantOrder: "desc",
		},
		{
			name: "valid desc order preserved",
			filter: &AuditLogFilter{
				SortOrder: "desc",
			},
			wantOrder: "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.checkNil {
				var nilFilter *AuditLogFilter
				nilFilter.NormalizeSort()
				return
			}

			tt.filter.NormalizeSort()
			if tt.filter.SortOrder != tt.wantOrder {
				t.Errorf("expected sort order %q, got %q", tt.wantOrder, tt.filter.SortOrder)
			}
		})
	}
}

func TestAuditLog_StructFields(t *testing.T) {
	userID := "01919df4-8e3b-7412-a1f9-90b567c9e203"
	now := time.Now().UTC()
	log := AuditLog{
		ID:        "01919df4-8e3b-7412-a1f9-90b567c9e301",
		UserID:    &userID,
		ActorName: "Dr. Who",
		Role:      "doctor",
		Action:    ActionConsultationStarted,
		Details:   map[string]any{"ticket_id": "01919df4-8e3b-7412-a1f9-90b567c9e401"},
		IPAddress: "10.0.0.1",
		CreatedAt: now,
	}

	if log.ID != "01919df4-8e3b-7412-a1f9-90b567c9e301" || *log.UserID != "01919df4-8e3b-7412-a1f9-90b567c9e203" || log.ActorName != "Dr. Who" || log.Role != "doctor" ||
		log.Action != ActionConsultationStarted || log.IPAddress != "10.0.0.1" || log.CreatedAt != now {
		t.Errorf("unexpected field values on AuditLog")
	}

	paginated := PaginatedAuditLogs{
		Page:         1,
		Limit:        20,
		TotalRecords: 1,
		Logs:         []AuditLog{log},
	}

	if paginated.Page != 1 || paginated.Limit != 20 || paginated.TotalRecords != 1 || len(paginated.Logs) != 1 {
		t.Errorf("unexpected field values on PaginatedAuditLogs")
	}
}
