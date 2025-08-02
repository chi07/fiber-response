package response

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/chi07/apperror"
)

type Response struct {
	Error *ErrorResponse `json:"error,omitempty"`
	Data  interface{}    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
}

func Error(ctx *fiber.Ctx, err error, message ...string) error {
	httpCode := GetHttpCode(err)

	msg := err.Error()
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	ctx.Status(httpCode)
	return ctx.JSON(&fiber.Map{
		"error": &ErrorResponse{
			Code:    httpCode,
			Message: msg,
		},
	})
}

func Errors(ctx *fiber.Ctx, errList []error) error {
	httpCode := fiber.StatusBadRequest
	ctx.Status(httpCode)
	var errMessage []string

	if len(errList) == 1 {
		return Error(ctx, errList[0], errList[0].Error())
	}

	for _, e := range errList {
		errMessage = append(errMessage, e.Error())
	}

	return ctx.JSON(&fiber.Map{
		"error": map[string]interface{}{
			"errors": errList,
		},
	})
}

func Success(ctx *fiber.Ctx, code int, data interface{}) error {
	ctx.Status(code)
	return ctx.JSON(&fiber.Map{
		"data": data,
	})
}

func GetHttpCode(err error) int {
	if err, ok := err.(apperror.AppError); ok {
		switch err.Code {
		case apperror.ErrInternalError:
			return http.StatusInternalServerError
		case apperror.ErrDuplicatedRecord:
			return http.StatusConflict
		case apperror.ErrInvalidFieldType, apperror.ErrInvalidFieldValue, apperror.ErrRequiredField, apperror.ErrNotMatched:
			return http.StatusBadRequest
		case apperror.ErrUnauthorized:
			return http.StatusUnauthorized
		case apperror.ErrPermissionDenied, apperror.ErrNotActivated:
			return http.StatusForbidden
		case apperror.ErrRecordNotFound:
			return http.StatusNotFound
		}
	}

	return http.StatusInternalServerError
}
