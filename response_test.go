package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/chi07/apperror/v2"

	response "github.com/chi07/fiber-response/v2"
)

type respEnvelope struct {
	Error *struct {
		Code     int      `json:"code"`
		AppCode  string   `json:"appCode,omitempty"`
		Message  string   `json:"message,omitempty"`
		Messages []string `json:"messages,omitempty"`
	} `json:"error,omitempty"`
	Data any `json:"data,omitempty"`
}

func newCtx(t *testing.T) (fiber.Ctx, func()) {
	t.Helper()
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	return ctx, func() { app.ReleaseCtx(ctx) }
}

func decodeBody(t *testing.T, ctx fiber.Ctx) respEnvelope {
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
	ctx, release := newCtx(t)
	defer release()

	err := response.Error(ctx, errors.New("some error"), "test error")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusInternalServerError, env.Error.Code)
		assert.Equal(t, "test error", env.Error.Message)
	}
}

func TestError_AppError_NoOverride(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	err := apperror.NewErrDuplicatedValue("email")
	assert.NoError(t, response.Error(ctx, err))

	assert.Equal(t, http.StatusConflict, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusConflict, env.Error.Code)
		assert.Equal(t, string(apperror.ErrDuplicatedRecord), env.Error.AppCode)
		assert.True(t, strings.Contains(env.Error.Message, "email"), env.Error.Message)
		assert.True(t, strings.Contains(env.Error.Message, "already used"), env.Error.Message)
	}
}

func TestError_AppError_DoesNotLeakCause(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	err := apperror.NewErrInternalServer("processing", errors.New("db: connection refused"))
	assert.NoError(t, response.Error(ctx, err))

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.NotContains(t, env.Error.Message, "db: connection refused")
	}
}

func TestError_Nil(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	assert.NoError(t, response.Error(ctx, nil))
	assert.Equal(t, http.StatusOK, ctx.Response().StatusCode())
}

func TestErrors_EmptySlice_Returns500(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

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

func TestErrors_AllNil_Returns500(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	err := response.Errors(ctx, []error{nil, nil})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())
}

func TestErrors_SingleDelegatesToError(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	e := errors.New("only one")
	err := response.Errors(ctx, []error{e})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusInternalServerError, env.Error.Code)
		assert.Equal(t, "only one", env.Error.Message)
	}
}

func TestErrors_Multiple_UsesFirstAppErrorStatus(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	errList := []error{
		errors.New("plain"),
		apperror.NewErrInvalidType("field"),
		apperror.NewErrMissingField("name"),
	}

	err := response.Errors(ctx, errList)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.NotNil(t, env.Error) {
		assert.Equal(t, http.StatusBadRequest, env.Error.Code)
		assert.Equal(t, string(apperror.ErrInvalidFieldType), env.Error.AppCode)
		assert.Empty(t, env.Error.Message)
		if assert.Len(t, env.Error.Messages, 3) {
			assert.True(t, strings.Contains(env.Error.Messages[1], "field"))
		}
	}
}

func TestSuccess_StatusAndBody(t *testing.T) {
	ctx, release := newCtx(t)
	defer release()

	payload := struct {
		Message string `json:"message"`
	}{Message: "success"}

	err := response.Success(ctx, http.StatusCreated, payload)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, ctx.Response().StatusCode())

	env := decodeBody(t, ctx)
	if assert.Nil(t, env.Error) {
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
		{"Suspended", apperror.NewErrSuspended("account"), http.StatusBadRequest},
		{"NotFound", apperror.NewErrRecordNotfound("id"), http.StatusNotFound},
		{"CustomAppError", apperror.NewAppError(422, "C422", "custom"), 422},
		{"PlainError", errors.New("some error"), http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedStatus, response.GetHTTPStatus(tc.err))
		})
	}
}
