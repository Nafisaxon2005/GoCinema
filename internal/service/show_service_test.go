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
	result := make([]model.Show, 0)
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

func (r *fakeShowRepo) Update(ctx context.Context, s *model.Show) error {
	if _, ok := r.shows[s.ID]; !ok {
		return model.ErrNotFound
	}
	cp := *s
	r.shows[s.ID] = &cp
	return nil
}

func (r *fakeShowRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := r.shows[id]; !ok {
		return model.ErrNotFound
	}
	delete(r.shows, id)
	return nil
}

func (r *fakeShowRepo) UpdatePoster(ctx context.Context, showID int64, posterPath string) error {
	s, ok := r.shows[showID]
	if !ok {
		return model.ErrNotFound
	}
	s.PosterPath = posterPath
	return nil
}

func (r *fakeShowRepo) GetStats(ctx context.Context, showID int64) (*model.ShowStats, error) {
	if _, ok := r.shows[showID]; !ok {
		return nil, model.ErrNotFound
	}
	return &model.ShowStats{ShowID: showID, TotalSeats: 10, SoldSeats: 5, Revenue: 2500, OccupancyRate: 50.0}, nil
}

func (r *fakeShowRepo) GenerateSeatMap(ctx context.Context, showID int64, seats []model.Seat) error {
	if _, ok := r.shows[showID]; !ok {
		return model.ErrNotFound
	}
	return nil
}

func (r *fakeShowRepo) CancelShow(ctx context.Context, showID int64) error {
	s, ok := r.shows[showID]
	if !ok {
		return model.ErrNotFound
	}
	s.Status = model.ShowCancelled
	return nil
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

func TestShowService_Create(t *testing.T) {
	repo := newFakeShowRepo()
	svc := NewShowService(repo)

	t.Run("успешное создание сеанса в будущем", func(t *testing.T) {
		s, err := svc.Create(context.Background(), 10, model.CreateShowInput{
			Title:    "Интерстеллар",
			Venue:    "Зал 1",
			StartsAt: time.Now().Add(48 * time.Hour),
		})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if s.Status != model.ShowDraft {
			t.Errorf("ожидался статус draft, получено %s", s.Status)
		}
		if s.OrganizerID != 10 {
			t.Errorf("ожидался organizer_id = 10, получено %d", s.OrganizerID)
		}
	})

	t.Run("ошибка при прошедшем времени", func(t *testing.T) {
		_, err := svc.Create(context.Background(), 10, model.CreateShowInput{
			Title:    "Интерстеллар",
			Venue:    "Зал 1",
			StartsAt: time.Now().Add(-1 * time.Hour),
		})
		if err != model.ErrInvalid {
			t.Errorf("ожидалась ошибка ErrInvalid, получено %v", err)
		}
	})
}

func TestShowService_Update_StatusTransitions(t *testing.T) {
	repo := newFakeShowRepo()
	svc := NewShowService(repo)

	show, _ := svc.Create(context.Background(), 1, model.CreateShowInput{
		Title:    "Тест",
		Venue:    "Зал 1",
		StartsAt: time.Now().Add(24 * time.Hour),
	})

	t.Run("чужой пользователь -> 403 Forbidden", func(t *testing.T) {
		status := model.ShowPublished
		_, err := svc.Update(context.Background(), 99, show.ID, model.UpdateShowInput{Status: &status})
		if err != model.ErrForbidden {
			t.Errorf("ожидалась ошибка ErrForbidden, получено %v", err)
		}
	})

	t.Run("draft -> published (разрешено)", func(t *testing.T) {
		status := model.ShowPublished
		updated, err := svc.Update(context.Background(), 1, show.ID, model.UpdateShowInput{Status: &status})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if updated.Status != model.ShowPublished {
			t.Errorf("статус должен быть published, получено %s", updated.Status)
		}
	})

	t.Run("published -> draft (запрещено)", func(t *testing.T) {
		status := model.ShowDraft
		_, err := svc.Update(context.Background(), 1, show.ID, model.UpdateShowInput{Status: &status})
		if err != model.ErrInvalid {
			t.Errorf("ожидалась ошибка ErrInvalid, получено %v", err)
		}
	})

	t.Run("published -> cancelled (разрешено)", func(t *testing.T) {
		status := model.ShowCancelled
		updated, err := svc.Update(context.Background(), 1, show.ID, model.UpdateShowInput{Status: &status})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if updated.Status != model.ShowCancelled {
			t.Errorf("статус должен быть cancelled, получено %s", updated.Status)
		}
	})
}

func TestShowService_Delete_Poster_SeatMap_Stats_Cancel(t *testing.T) {
	repo := newFakeShowRepo()
	svc := NewShowService(repo)

	show, _ := svc.Create(context.Background(), 10, model.CreateShowInput{
		Title:    "Фильм 1",
		Venue:    "Зал A",
		StartsAt: time.Now().Add(24 * time.Hour),
	})

	t.Run("UploadPoster и GetPosterPath", func(t *testing.T) {
		_, err := svc.UploadPoster(context.Background(), 99, show.ID, "poster.jpg")
		if err != model.ErrForbidden {
			t.Errorf("ожидалось 403 Forbidden, получено %v", err)
		}

		updated, err := svc.UploadPoster(context.Background(), 10, show.ID, "poster.jpg")
		if err != nil || updated.PosterPath != "poster.jpg" {
			t.Errorf("не удалось обновить постер: %v", err)
		}

		path, err := svc.GetPosterPath(context.Background(), show.ID)
		if err != nil || path != "poster.jpg" {
			t.Errorf("ошибка получения постера: %v", err)
		}
	})

	t.Run("GenerateSeatMap", func(t *testing.T) {
		err := svc.GenerateSeatMap(context.Background(), 10, show.ID, model.GenerateSeatMapInput{
			Rows:        5,
			SeatsPerRow: 10,
			Price:       500,
			Zones: []model.SeatMapZone{
				{FromRow: 1, ToRow: 2, Price: 1000},
			},
		})
		if err != nil {
			t.Errorf("неожиданная ошибка генерации схемы зала: %v", err)
		}
	})

	t.Run("GetStats", func(t *testing.T) {
		stats, err := svc.GetStats(context.Background(), 10, show.ID)
		if err != nil || stats == nil {
			t.Errorf("неожиданная ошибка получения статистики: %v", err)
		}
	})

	t.Run("CancelShow", func(t *testing.T) {
		err := svc.CancelShow(context.Background(), 10, show.ID)
		if err != nil {
			t.Errorf("неожиданная ошибка отмены сеанса: %v", err)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := svc.Delete(context.Background(), 10, show.ID)
		if err != nil {
			t.Errorf("неожиданная ошибка удаления: %v", err)
		}
	})
}


