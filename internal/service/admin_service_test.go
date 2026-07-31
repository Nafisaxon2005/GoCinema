package service

import (
	"context"
	"testing"
	"time"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

// --- моки ---

type mockAdminRepo struct {
	getStatsFn  func(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error)
	listShowsFn func(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error)
}

func (m *mockAdminRepo) GetStatsByPeriod(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
	return m.getStatsFn(ctx, from, to)
}

func (m *mockAdminRepo) ListShows(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
	return m.listShowsFn(ctx, f)
}

// mockShowRepo реализует весь repository.ShowRepo, но тестам нужны только
// GetByID/UpdateStatus — остальные методы просто не должны использоваться.
type mockShowRepo struct {
	getByIDFn      func(ctx context.Context, id int64) (*model.Show, error)
	updateStatusFn func(ctx context.Context, id int64, status model.ShowStatus) error
}

func (m *mockShowRepo) Create(ctx context.Context, s *model.Show) (int64, error) { return 0, nil }
func (m *mockShowRepo) GetByID(ctx context.Context, id int64) (*model.Show, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockShowRepo) List(ctx context.Context, f repository.ShowListFilter) ([]model.Show, int, error) {
	return nil, 0, nil
}
func (m *mockShowRepo) UpdateStatus(ctx context.Context, id int64, status model.ShowStatus) error {
	return m.updateStatusFn(ctx, id, status)
}
func (m *mockShowRepo) Update(ctx context.Context, s *model.Show) error { return nil }
func (m *mockShowRepo) Delete(ctx context.Context, id int64) error      { return nil }
func (m *mockShowRepo) UpdatePoster(ctx context.Context, showID int64, path string) error {
	return nil
}
func (m *mockShowRepo) GetStats(ctx context.Context, showID int64) (*model.ShowStats, error) {
	return nil, nil
}
func (m *mockShowRepo) GenerateSeatMap(ctx context.Context, showID int64, seats []model.Seat) error {
	return nil
}
func (m *mockShowRepo) CancelShow(ctx context.Context, showID int64) error { return nil }
func (m *mockShowRepo) CountFreeSeats(ctx context.Context, showID int64) (int, error) {
	return 0, nil
}

// --- тесты GetStats ---

func TestAdminService_GetStats(t *testing.T) {
	tests := []struct {
		name    string
		from    time.Time
		to      time.Time
		mockErr error
		wantErr error
	}{
		{
			name:    "to before from returns ErrInvalid",
			from:    time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
			to:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			wantErr: model.ErrInvalid,
		},
		{
			name: "valid period delegates to repo",
			from: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			to:   time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockAdminRepo{
				getStatsFn: func(ctx context.Context, from, to time.Time) ([]model.ShowSalesStat, error) {
					return []model.ShowSalesStat{{ShowID: 1, Sold: 5, Total: 10, Revenue: 1000}}, tt.mockErr
				},
			}
			svc := NewAdminService(repo, &mockShowRepo{})

			_, err := svc.GetStats(context.Background(), tt.from, tt.to)
			if tt.wantErr != nil {
				if err != tt.wantErr {
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

// --- тесты ListShows ---

func TestAdminService_ListShows_Defaults(t *testing.T) {
	var gotFilter repository.AdminShowFilter

	repo := &mockAdminRepo{
		listShowsFn: func(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
			gotFilter = f
			return []model.Show{}, 0, nil
		},
	}
	svc := NewAdminService(repo, &mockShowRepo{})

	_, _, err := svc.ListShows(context.Background(), repository.AdminShowFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFilter.Page != 1 {
		t.Errorf("expected default Page=1, got %d", gotFilter.Page)
	}
	if gotFilter.PageSize != 20 {
		t.Errorf("expected default PageSize=20, got %d", gotFilter.PageSize)
	}
}

func TestAdminService_ListShows_PageSizeCap(t *testing.T) {
	var gotFilter repository.AdminShowFilter

	repo := &mockAdminRepo{
		listShowsFn: func(ctx context.Context, f repository.AdminShowFilter) ([]model.Show, int, error) {
			gotFilter = f
			return []model.Show{}, 0, nil
		},
	}
	svc := NewAdminService(repo, &mockShowRepo{})

	_, _, _ = svc.ListShows(context.Background(), repository.AdminShowFilter{Page: 2, PageSize: 500})
	if gotFilter.PageSize != 20 {
		t.Errorf("expected PageSize capped to 20, got %d", gotFilter.PageSize)
	}
	if gotFilter.Page != 2 {
		t.Errorf("expected Page=2 preserved, got %d", gotFilter.Page)
	}
}

// --- тесты ModerateShow ---

func TestAdminService_ModerateShow(t *testing.T) {
	tests := []struct {
		name       string
		showStatus model.ShowStatus
		getByIDErr error
		action     string
		wantErr    error
		wantStatus model.ShowStatus
		expectCall bool
	}{
		{
			name:       "approve draft show succeeds",
			showStatus: model.ShowDraft,
			action:     "approve",
			wantStatus: model.ShowPublished,
			expectCall: true,
		},
		{
			name:       "reject draft show succeeds",
			showStatus: model.ShowDraft,
			action:     "reject",
			wantStatus: model.ShowCancelled,
			expectCall: true,
		},
		{
			name:       "approve already published show fails",
			showStatus: model.ShowPublished,
			action:     "approve",
			wantErr:    model.ErrInvalid,
		},
		{
			name:       "unknown action fails",
			showStatus: model.ShowDraft,
			action:     "delete",
			wantErr:    model.ErrInvalid,
		},
		{
			name:       "show not found propagates error",
			getByIDErr: model.ErrNotFound,
			action:     "approve",
			wantErr:    model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calledWith model.ShowStatus
			called := false

			showRepo := &mockShowRepo{
				getByIDFn: func(ctx context.Context, id int64) (*model.Show, error) {
					if tt.getByIDErr != nil {
						return nil, tt.getByIDErr
					}
					return &model.Show{ID: id, Status: tt.showStatus}, nil
				},
				updateStatusFn: func(ctx context.Context, id int64, status model.ShowStatus) error {
					called = true
					calledWith = status
					return nil
				},
			}
			svc := NewAdminService(&mockAdminRepo{}, showRepo)

			err := svc.ModerateShow(context.Background(), 1, tt.action)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.expectCall && !called {
				t.Fatal("expected UpdateStatus to be called, but it wasn't")
			}
			if tt.expectCall && calledWith != tt.wantStatus {
				t.Errorf("expected status %s, got %s", tt.wantStatus, calledWith)
			}
		})
	}
}
