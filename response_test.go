package response_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/chi07/apperror"
	response "github.com/chi07/fiber-response"
)

func TestError(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	// Set the status code
	ctx.Status(http.StatusBadRequest)

	err := response.Error(ctx, errors.New("some error"), "test error")
	assert.NoError(t, err)
}

func TestErrors(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Status(http.StatusBadRequest)
	defer app.ReleaseCtx(ctx)

	errList := []error{
		apperror.NewErrInvalidType("field"),
	}

	err := response.Errors(ctx, errList)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ctx.Response().StatusCode())

	errList2 := []error{
		errors.New("error 1"),
		errors.New("error 2"),
	}

	err = response.Errors(ctx, errList2)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, ctx.Response().StatusCode())
}

func TestSuccess(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	ctx.Status(http.StatusBadRequest)
	defer app.ReleaseCtx(ctx)

	data := struct {
		Message string `json:"message"`
	}{
		Message: "success",
	}

	err := response.Success(ctx, http.StatusOK, data)

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, ctx.Response().StatusCode())
}

func TestGetHttpCode(t *testing.T) {
	tests := []struct {
		err            error
		expectedStatus int
	}{
		{
			err:            apperror.NewErrInternalServer("internal server error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			err:            apperror.NewErrDuplicatedValue("field"),
			expectedStatus: http.StatusConflict,
		},
		{
			err:            apperror.NewErrInvalidValue("field"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			err:            apperror.NewErrRecordNotfound("field"),
			expectedStatus: http.StatusNotFound,
		},
		{
			err:            errors.New("some error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		status := response.GetHttpCode(test.err)
		assert.Equal(t, test.expectedStatus, status)
	}
}
