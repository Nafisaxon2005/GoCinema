package httpx

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/raxima/seatpicker/internal/model"
)

type errorResponse struct {
	Error string `json:"error"`
}

func RespondJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}
func RespondCreated(c *gin.Context, payload any) {
	RespondJSON(c, http.StatusCreated, payload)
}
func RespondOK(c *gin.Context, payload any) {
	RespondJSON(c, http.StatusOK, payload)
}
func RespondError(c *gin.Context, err error) {
	status := http.StatusInternalServerError //непредвиденная ошибка  не утекала наружу текстом в JSON
	switch {
	case errors.Is(err, model.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, model.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, model.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, model.ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, model.ErrAlreadyExists), errors.Is(err, model.ErrSeatTaken):
		status = http.StatusConflict
	}

	message := err.Error()
	if status == http.StatusInternalServerError {
		log.Println(err)
		message = "internal error"
	}
	c.JSON(status, errorResponse{Error: message})
}
