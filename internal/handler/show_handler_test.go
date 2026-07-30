package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/raxima/seatpicker/internal/model"
	"github.com/raxima/seatpicker/internal/repository"
	"github.com/raxima/seatpicker/internal/service"
)

type fakeShowRepo struct {
	shows     map[int64]*model.Show
	freeSeats map[int64]int
	nextID    int64
}

func newFakeShowRepo() *fakeShowRepo {
	return &fakeShowRepo{shows: make(map[int64]*model.Show), freeSeats: make(map[int64]int)}
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
	var result []model.Show
	for _, s := range r.shows {
		if s.Status == f.Status {
			result = append(result, *s)
		}
	}
	return result, len(result), nil
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

func doGetRequest(handler gin.HandlerFunc, routePath, target string) *httptest.ResponseRecorder {
	r := gin.New()
	r.GET(routePath, handler)

	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestShowHandler_List(t *testing.T) {
	repo := newFakeShowRepo()
	now := time.Now()
	_, _ = repo.Create(context.Background(), &model.Show{Title: "Дюна", Status: model.ShowPublished, StartsAt: now})
	_, _ = repo.Create(context.Background(), &model.Show{Title: "Черновик", Status: model.ShowDraft, StartsAt: now})

	svc := service.NewShowService(repo)
	h := NewShowHandler(svc)

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "без фильтров", target: "/shows", wantStatus: http.StatusOK},
		{name: "невалидный date_from", target: "/shows?date_from=not-a-date", wantStatus: http.StatusBadRequest},
		{name: "невалидный date_to", target: "/shows?date_to=not-a-date", wantStatus: http.StatusBadRequest},
		{name: "валидный date_from", target: "/shows?date_from=2020-01-01", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doGetRequest(h.List, "/shows", tt.target)
			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func TestShowHandler_GetByID(t *testing.T) {
	repo := newFakeShowRepo()
	now := time.Now()
	publishedID, _ := repo.Create(context.Background(), &model.Show{Title: "Дюна", Status: model.ShowPublished, StartsAt: now})
	draftID, _ := repo.Create(context.Background(), &model.Show{Title: "Черновик", Status: model.ShowDraft, StartsAt: now})

	svc := service.NewShowService(repo)
	h := NewShowHandler(svc)

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "существующий published сеанс", target: "/shows/" + itoa(publishedID), wantStatus: http.StatusOK},
		{name: "draft сеанс — 404", target: "/shows/" + itoa(draftID), wantStatus: http.StatusNotFound},
		{name: "несуществующий id", target: "/shows/999999", wantStatus: http.StatusNotFound},
		{name: "невалидный id (не число)", target: "/shows/abc", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doGetRequest(h.GetByID, "/shows/:id", tt.target)
			if w.Code != tt.wantStatus {
				t.Errorf("статус = %d, ожидалось %d, тело: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
