package service

import (
	"context"
	"testing"
	"time"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
)

type fakeShowRepo struct {
	shows     map[int64]*model.Show
	freeSeats map[int64]int
	nextID    int64
	listErr   error
	countErr  error
}

func newFakeShowRepo() *fakeShowRepo {
	return &fakeShowRepo{
		shows:     make(map[int64]*model.Show),
		freeSeats: make(map[int64]int),
	}
}

func (r *fakeShowRepo) Create(ctx context.Context, s *model.Show) (int64, error) {
	r.nextID++
	s.ID = r.nextID
	cp := *s
	r.shows[s.ID] = &cp
	return s.ID, nil
}

func (r *fakeShowRepo) GetByID(ctx context.Context, id int64) (*model.Show, error) {
	s, ok := r.shows[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (r *fakeShowRepo) List(ctx context.Context, f repository.ShowListFilter) ([]model.Show, int, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	var result []model.Show
	for _, s := range r.shows {
		if s.Status != f.Status {
			continue
		}
		if f.Search != "" && !contains(s.Title, f.Search) {
			continue
		}
		if f.Venue != "" && s.Venue != f.Venue {
			continue
		}
		result = append(result, *s)
	}
	total := len(result)

	start := (f.Page - 1) * f.PageSize
	if start > len(result) {
		start = len(result)
	}
	end := start + f.PageSize
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}

func (r *fakeShowRepo) UpdateStatus(ctx context.Context, id int64, status model.ShowStatus) error {
	s, ok := r.shows[id]
	if !ok {
		return model.ErrNotFound
	}
	s.Status = status
	return nil
}

func (r *fakeShowRepo) CountFreeSeats(ctx context.Context, showID int64) (int, error) {
	if r.countErr != nil {
		return 0, r.countErr
	}
	return r.freeSeats[showID], nil
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 ||
		(len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestShowService_List(t *testing.T) {
	repo := newFakeShowRepo()
	now := time.Now()

	_, _ = repo.Create(context.Background(), &model.Show{Title: "Дюна", Venue: "Зал 1", StartsAt: now, Status: model.ShowPublished})
	_, _ = repo.Create(context.Background(), &model.Show{Title: "Оппенгеймер", Venue: "Зал 2", StartsAt: now, Status: model.ShowPublished})
	_, _ = repo.Create(context.Background(), &model.Show{Title: "Черновик", Venue: "Зал 1", StartsAt: now, Status: model.ShowDraft})

	svc := NewShowService(repo)

	tests := []struct {
		name      string
		input     ListShowsInput
		wantCount int
		wantTotal int
	}{
		{
			name:      "показывает только published",
			input:     ListShowsInput{},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:      "фильтр по залу",
			input:     ListShowsInput{Venue: "Зал 1"},
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "поиск по названию",
			input:     ListShowsInput{Search: "Дюна"},
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:      "пагинация — некорректная страница нормализуется",
			input:     ListShowsInput{Page: 0, PageSize: 0},
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:      "page_size больше максимума ограничивается",
			input:     ListShowsInput{PageSize: 1000},
			wantCount: 2,
			wantTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.List(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if len(resp.Items) != tt.wantCount {
				t.Errorf("количество items = %d, ожидалось %d", len(resp.Items), tt.wantCount)
			}
			if resp.Total != tt.wantTotal {
				t.Errorf("Total = %d, ожидалось %d", resp.Total, tt.wantTotal)
			}
		})
	}
}

func TestShowService_GetByID(t *testing.T) {
	repo := newFakeShowRepo()
	now := time.Now()

	publishedID, _ := repo.Create(context.Background(), &model.Show{Title: "Дюна", Status: model.ShowPublished, StartsAt: now})
	draftID, _ := repo.Create(context.Background(), &model.Show{Title: "Черновик", Status: model.ShowDraft, StartsAt: now})
	repo.freeSeats[publishedID] = 42

	svc := NewShowService(repo)

	tests := []struct {
		name          string
		id            int64
		wantErr       error
		wantFreeSeats int
	}{
		{
			name:          "published сеанс доступен",
			id:            publishedID,
			wantFreeSeats: 42,
		},
		{
			name:    "draft сеанс недоступен (как будто не существует)",
			id:      draftID,
			wantErr: model.ErrNotFound,
		},
		{
			name:    "несуществующий id",
			id:      999,
			wantErr: model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detail, err := svc.GetByID(context.Background(), tt.id)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("ожидалась ошибка %v, получено %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if detail.FreeSeats != tt.wantFreeSeats {
				t.Errorf("FreeSeats = %d, ожидалось %d", detail.FreeSeats, tt.wantFreeSeats)
			}
		})
	}
}
