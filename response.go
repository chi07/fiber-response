package response

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/chi07/apperror"
)

type Response struct {
	Error *ErrorResponse `json:"error,omitempty"`
	Data  any            `json:"data,omitempty"`
}

type ErrorResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message,omitempty"`
	Messages []string `json:"messages,omitempty"` // dùng cho Errors()
}

func Error(ctx *fiber.Ctx, err error, override ...string) error {
	status := GetHTTPStatus(err)

	msg := err.Error()
	if len(override) > 0 && override[0] != "" {
		msg = override[0]
	}

	ctx.Status(status)
	return ctx.JSON(Response{
		Error: &ErrorResponse{
			Code:    status,
			Message: msg,
		},
	})
}

func Errors(ctx *fiber.Ctx, errList []error) error {
	if len(errList) == 0 {
		ctx.Status(http.StatusInternalServerError)
		return ctx.JSON(Response{
			Error: &ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			},
		})
	}

	if len(errList) == 1 {
		return Error(ctx, errList[0])
	}

	msgs := make([]string, 0, len(errList))
	for _, e := range errList {
		if e == nil {
			continue
		}
		msgs = append(msgs, e.Error())
	}

	ctx.Status(http.StatusBadRequest)
	return ctx.JSON(Response{
		Error: &ErrorResponse{
			Code:     http.StatusBadRequest,
			Messages: msgs,
		},
	})
}

func Success(ctx *fiber.Ctx, code int, data any) error {
	ctx.Status(code)
	return ctx.JSON(Response{
		Data: data,
	})
}

func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if ae, ok := apperror.AsAppError(err); ok {
		if ae.HTTPCode != 0 {
			return ae.HTTPCode
		}

		if s := mapCodeToStatus(ae.Code); s != 0 {
			return s
		}
		return http.StatusInternalServerError
	}

	return http.StatusInternalServerError
}

func mapCodeToStatus(code apperror.Code) int {
	switch code {
	case apperror.ErrInternalError:
		return http.StatusInternalServerError
	case apperror.ErrDuplicatedRecord:
		return http.StatusConflict
	case apperror.ErrInvalidFieldType,
		apperror.ErrInvalidFieldValue,
		apperror.ErrRequiredField,
		apperror.ErrNotMatched:
		return http.StatusBadRequest
	case apperror.ErrUnauthorized:
		return http.StatusUnauthorized
	case apperror.ErrPermissionDenied, apperror.ErrNotActivated, apperror.ErrSuspended:
		return http.StatusForbidden
	case apperror.ErrRecordNotFound:
		return http.StatusNotFound
	default:
		return 0
	}
}
