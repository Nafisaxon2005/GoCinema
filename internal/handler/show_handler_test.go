package handler

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
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

func TestShowHandler_OrganizerEndpoints(t *testing.T) {
	repo := newFakeShowRepo()
	svc := service.NewShowService(repo)
	h := NewShowHandler(svc)

	showID, _ := repo.Create(context.Background(), &model.Show{
		OrganizerID: 10,
		Title:       "Фильм",
		Venue:       "Зал 1",
		StartsAt:    time.Now().Add(24 * time.Hour),
		Status:      model.ShowDraft,
	})

	doPostJSON := func(handler gin.HandlerFunc, routePath, target string, body string, userID int64) *httptest.ResponseRecorder {
		r := gin.New()
		r.POST(routePath, func(c *gin.Context) {
			if userID > 0 {
				c.Set("userID", userID)
			}
			handler(c)
		})
		req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	doPutJSON := func(handler gin.HandlerFunc, routePath, target string, body string, userID int64) *httptest.ResponseRecorder {
		r := gin.New()
		r.PUT(routePath, func(c *gin.Context) {
			if userID > 0 {
				c.Set("userID", userID)
			}
			handler(c)
		})
		req := httptest.NewRequest(http.MethodPut, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	doDeleteReq := func(handler gin.HandlerFunc, routePath, target string, userID int64) *httptest.ResponseRecorder {
		r := gin.New()
		r.DELETE(routePath, func(c *gin.Context) {
			if userID > 0 {
				c.Set("userID", userID)
			}
			handler(c)
		})
		req := httptest.NewRequest(http.MethodDelete, target, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("Create - 401 without auth", func(t *testing.T) {
		w := doPostJSON(h.Create, "/shows", "/shows", `{"title":"A","venue":"B"}`, 0)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Create - 201 Created", func(t *testing.T) {
		future := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
		body := `{"title":"Новый","venue":"Зал 2","starts_at":"` + future + `"}`
		w := doPostJSON(h.Create, "/shows", "/shows", body, 10)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d, body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Update - 200 OK", func(t *testing.T) {
		body := `{"title":"Обновлённый"}`
		w := doPutJSON(h.Update, "/shows/:id", "/shows/"+itoa(showID), body, 10)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Update - 403 Forbidden for non-owner", func(t *testing.T) {
		body := `{"title":"Обновлённый"}`
		w := doPutJSON(h.Update, "/shows/:id", "/shows/"+itoa(showID), body, 999)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("Delete - 204 No Content", func(t *testing.T) {
		w := doDeleteReq(h.Delete, "/shows/:id", "/shows/"+itoa(showID), 10)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
	})

	t.Run("GenerateSeatMap - 200 OK", func(t *testing.T) {
		newID, _ := repo.Create(context.Background(), &model.Show{OrganizerID: 10, StartsAt: time.Now().Add(time.Hour)})
		body := `{"rows":5,"seats_per_row":10,"price":500}`
		w := doPostJSON(h.GenerateSeatMap, "/shows/:id/seatmap", "/shows/"+itoa(newID)+"/seatmap", body, 10)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GetStats - 200 OK", func(t *testing.T) {
		newID, _ := repo.Create(context.Background(), &model.Show{OrganizerID: 10, StartsAt: time.Now().Add(time.Hour)})
		r := gin.New()
		r.GET("/shows/:id/stats", func(c *gin.Context) {
			c.Set("userID", int64(10))
			h.GetStats(c)
		})
		req := httptest.NewRequest(http.MethodGet, "/shows/"+itoa(newID)+"/stats", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("Cancel - 200 OK", func(t *testing.T) {
		newID, _ := repo.Create(context.Background(), &model.Show{OrganizerID: 10, StartsAt: time.Now().Add(time.Hour)})
		w := doPutJSON(h.Cancel, "/shows/:id/cancel", "/shows/"+itoa(newID)+"/cancel", "", 10)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("UploadPoster - multipart upload & GetPoster", func(t *testing.T) {
		newID, _ := repo.Create(context.Background(), &model.Show{OrganizerID: 10, StartsAt: time.Now().Add(time.Hour)})

		var body bytes.Buffer
		mw := multipart.NewWriter(&body)
		hHeader := make(textproto.MIMEHeader)
		hHeader.Set("Content-Disposition", `form-data; name="poster"; filename="test.png"`)
		hHeader.Set("Content-Type", "image/png")
		part, _ := mw.CreatePart(hHeader)
		part.Write([]byte("fake-image-bytes"))
		mw.Close()

		r := gin.New()
		r.POST("/shows/:id/poster", func(c *gin.Context) {
			c.Set("userID", int64(10))
			h.UploadPoster(c)
		})

		req := httptest.NewRequest(http.MethodPost, "/shows/"+itoa(newID)+"/poster", &body)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d, body: %s", w.Code, w.Body.String())
		}

		// Test GetPoster
		rGet := gin.New()
		rGet.GET("/shows/:id/poster", h.GetPoster)
		reqGet := httptest.NewRequest(http.MethodGet, "/shows/"+itoa(newID)+"/poster", nil)
		wGet := httptest.NewRecorder()
		rGet.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Errorf("expected 200 for GetPoster, got %d", wGet.Code)
		}
	})

	t.Run("Invalid ID parameters -> 400 Bad Request", func(t *testing.T) {
		if w := doPutJSON(h.Update, "/shows/:id", "/shows/abc", `{}`, 10); w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		if w := doDeleteReq(h.Delete, "/shows/:id", "/shows/abc", 10); w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		if w := doPostJSON(h.GenerateSeatMap, "/shows/:id/seatmap", "/shows/abc/seatmap", `{}`, 10); w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
		if w := doPutJSON(h.Cancel, "/shows/:id/cancel", "/shows/abc/cancel", `{}`, 10); w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
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

