// Package response provides a consistent JSON envelope for fiber v3
// handlers: {data} on success and {error:{code,appCode,message|messages}}
// on failure, with the HTTP status resolved from apperror codes.
package response

import (
	"net/http"

	"github.com/chi07/apperror/v2"
	"github.com/gofiber/fiber/v3"
)

// Response is the standard JSON envelope.
type Response struct {
	Error *ErrorResponse `json:"error,omitempty"`
	Data  any            `json:"data,omitempty"`
}

// ErrorResponse describes a failure. Code mirrors the HTTP status;
// AppCode carries the stable application code (e.g. "V4004") when the
// error is an *apperror.AppError.
type ErrorResponse struct {
	Code     int           `json:"code"`
	AppCode  apperror.Code `json:"appCode,omitempty"`
	Message  string        `json:"message,omitempty"`
	Messages []string      `json:"messages,omitempty"`
}

// Error writes a single-error envelope. For *apperror.AppError the public
// ae.Message is sent (the wrapped cause is never leaked to clients); other
// errors fall back to err.Error() with status 500. A non-empty override
// replaces the message.
func Error(c fiber.Ctx, err error, override ...string) error {
	if err == nil {
		return Success(c, http.StatusOK, nil)
	}

	status := GetHTTPStatus(err)
	resp := ErrorResponse{Code: status}

	if ae, ok := apperror.AsAppError(err); ok {
		resp.AppCode = ae.Code
		resp.Message = string(ae.Message)
	} else {
		resp.Message = err.Error()
	}
	if len(override) > 0 && override[0] != "" {
		resp.Message = override[0]
	}

	return c.Status(status).JSON(Response{Error: &resp})
}

// Errors writes a multi-error envelope. A single non-nil error delegates to
// Error; the status is taken from the first *apperror.AppError found
// (default 400). An empty (or all-nil) list produces a 500 envelope.
func Errors(c fiber.Ctx, errList []error) error {
	errs := make([]error, 0, len(errList))
	for _, e := range errList {
		if e != nil {
			errs = append(errs, e)
		}
	}

	switch len(errs) {
	case 0:
		return c.Status(http.StatusInternalServerError).JSON(Response{
			Error: &ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: http.StatusText(http.StatusInternalServerError),
			},
		})
	case 1:
		return Error(c, errs[0])
	}

	status := http.StatusBadRequest
	resp := ErrorResponse{Code: status, Messages: make([]string, 0, len(errs))}
	for _, e := range errs {
		if ae, ok := apperror.AsAppError(e); ok {
			if resp.AppCode == "" && ae.HTTPCode != 0 {
				resp.AppCode = ae.Code
				status = ae.HTTPCode
				resp.Code = status
			}
			resp.Messages = append(resp.Messages, string(ae.Message))
		} else {
			resp.Messages = append(resp.Messages, e.Error())
		}
	}
	return c.Status(status).JSON(Response{Error: &resp})
}

// Success writes a data envelope with the given status.
func Success(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(Response{Data: data})
}

// GetHTTPStatus resolves the HTTP status for err: 200 for nil, the
// AppError's own HTTPCode when present, otherwise 500.
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if ae, ok := apperror.AsAppError(err); ok && ae.HTTPCode != 0 {
		return ae.HTTPCode
	}
	return http.StatusInternalServerError
}
