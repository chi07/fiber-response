package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/chi07/apperror"

	response "github.com/chi07/fiber-response"
)

type respEnvelope struct {
	Error *struct {
		Code     int      `json:"code"`
		Message  string   `json:"message,omitempty"`
		Messages []string `json:"messages,omitempty"`
	} `json:"error,omitempty"`
	Data any `json:"data,omitempty"`
}

func decodeBody(t *testing.T, ctx *fiber.Ctx) respEnvelope {
	t.Helper()
	var env respEnvelope
	body := ctx.Response().Body()
	if len(body) == 0 {
		t.Fatalf("empty response body")
	}
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody=%s", err, string(body))
	}
	return env
}

func TestError_WithOverrideMessage(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	err := response.Error(ctx, errors.New("some error"), "test error")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusInternalServerError, env.Error.Code)
		assert.Equal(t, "test error", env.Error.Message)
	}
}

func TestError_AppError_NoOverride_UsesAppMessageAndHttpCode(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	// AppError có HttpCode 409 & message tuỳ theo apperror
	err := apperror.NewErrDuplicatedValue("email")
	assert.NoError(t, response.Error(ctx, err))

	assert.Equal(t, http.StatusConflict, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusConflict, env.Error.Code)
		assert.True(t, strings.Contains(env.Error.Message, "email"), env.Error.Message)
		assert.True(t, strings.Contains(env.Error.Message, "already used"), env.Error.Message)
	}
}

func TestErrors_EmptySlice_Returns500(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	err := response.Errors(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusInternalServerError, env.Error.Code)
		assert.NotEmpty(t, env.Error.Message)
		assert.Empty(t, env.Error.Messages)
	}
}

func TestErrors_SingleDelegatesToError(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	e := errors.New("only one")
	err := response.Errors(ctx, []error{e})
	assert.NoError(t, err)
	// Errors() với single error sẽ gọi Error() → 500 by default với error thường
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusInternalServerError, env.Error.Code)
		assert.Equal(t, "only one", env.Error.Message)
	}
}

func TestErrors_Multiple_ReturnsMessages400(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	errList := []error{
		apperror.NewErrInvalidType("field"),
		errors.New("plain 2"),
		errors.New("plain 3"),
	}

	err := response.Errors(ctx, errList)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusBadRequest, env.Error.Code)
		assert.Empty(t, env.Error.Message) // dùng messages thay vì message
		if assert.Len(t, env.Error.Messages, 3) {
			assert.True(t, strings.Contains(env.Error.Messages[0], "field"))
		}
	}
}

func TestSuccess_StatusAndBody(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	payload := struct {
		Message string `json:"message"`
	}{Message: "success"}

	err := response.Success(ctx, http.StatusCreated, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.Nil(t, env.Error) {
		// Data là object có field message
		m, ok := env.Data.(map[string]any)
		if assert.True(t, ok) {
			assert.Equal(t, "success", m["message"])
		}
	}
}

func TestGetHTTPStatus_Table(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{"NilErrorReturns200", nil, http.StatusOK},
		{"Internal", apperror.NewErrInternalServer("boom"), http.StatusInternalServerError},
		{"Duplicated", apperror.NewErrDuplicatedValue("email"), http.StatusConflict},
		{"InvalidValue", apperror.NewErrInvalidValue("field"), http.StatusBadRequest},
		{"InvalidType", apperror.NewErrInvalidType("field"), http.StatusBadRequest},
		{"Required", apperror.NewErrMissingField("field"), http.StatusBadRequest},
		{"NotMatched", apperror.NewErrNotMatched("a", "b"), http.StatusBadRequest},
		{"Unauthorized", apperror.NewErrUnauthorized("token"), http.StatusUnauthorized},
		{"ForbiddenPermission", apperror.NewErrStatusForbidden("no access"), http.StatusForbidden},
		{"ForbiddenNotActivated", apperror.NewErrNotActivated("account"), http.StatusForbidden},
		{"ForbiddenSuspended", apperror.NewErrSuspended("account"), http.StatusBadRequest /* or 403 if your mapCodeToStatus maps it so */},
		{"NotFound", apperror.NewErrRecordNotfound("id"), http.StatusNotFound},
		{"PlainError", errors.New("some error"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status := response.GetHTTPStatus(tc.err)
			assert.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestGetHTTPStatus_AppError_NoHttpCode_MapFallback(t *testing.T) {
	ae := &apperror.AppError{
		Code:     apperror.ErrUnauthorized,
		Message:  apperror.NewMessage("unauth"),
		HTTPCode: 0,
	}
	status := response.GetHTTPStatus(ae)
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestGetHTTPStatus_AppError_NoHttpCode_UnknownCode_Default500(t *testing.T) {
	ae := &apperror.AppError{
		Code:     apperror.Code("UNKNOWN_XYZ"),
		Message:  apperror.NewMessage("unknown"),
		HTTPCode: 0,
	}
	status := response.GetHTTPStatus(ae)
	assert.Equal(t, http.StatusInternalServerError, status)
}
