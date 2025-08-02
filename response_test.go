package response_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	response "github.com/chi07/fiber-response"
)

func setupApp() *fiber.App {
	return fiber.New()
}

func TestOK(t *testing.T) {
	app := setupApp()

	app.Get("/ok", func(c *fiber.Ctx) error {
		return response.OK(c, fiber.Map{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/ok", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body response.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.True(t, body.Success)
	assert.Equal(t, "success", body.Data.(map[string]interface{})["message"])
}

func TestCreated(t *testing.T) {
	app := setupApp()

	app.Post("/created", func(c *fiber.Ctx) error {
		return response.Created(c, fiber.Map{"id": "abc123"})
	})

	req := httptest.NewRequest("POST", "/created", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var body response.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.True(t, body.Success)
	assert.Equal(t, "abc123", body.Data.(map[string]interface{})["id"])
}

func TestNoContent(t *testing.T) {
	app := setupApp()

	app.Delete("/nocontent", func(c *fiber.Ctx) error {
		return response.NoContent(c)
	})

	req := httptest.NewRequest("DELETE", "/nocontent", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestError(t *testing.T) {
	app := setupApp()

	app.Get("/error", func(c *fiber.Ctx) error {
		return response.Error(c, fiber.StatusBadRequest, "Bad Request")
	})

	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var body response.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.False(t, body.Success)
	assert.Equal(t, "Bad Request", body.Error)
}

func TestValidationError(t *testing.T) {
	app := setupApp()

	app.Post("/validate", func(c *fiber.Ctx) error {
		return response.ValidationError(c, []response.FieldError{
			{Field: "title", Message: "required"},
		})
	})

	req := httptest.NewRequest("POST", "/validate", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)

	var body response.ValidationErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.False(t, body.Success)
	assert.Equal(t, "Validation failed", body.Error)
	assert.Equal(t, "title", body.Fields[0].Field)
	assert.Equal(t, "required", body.Fields[0].Message)
}

func TestPaginate(t *testing.T) {
	app := setupApp()

	app.Get("/paginate", func(c *fiber.Ctx) error {
		return response.Paginate(c, 1, 10, 25, []string{"lesson1", "lesson2"})
	})

	req := httptest.NewRequest("GET", "/paginate", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body response.SuccessResponse
	err = json.NewDecoder(resp.Body).Decode(&body)
	assert.NoError(t, err)
	assert.True(t, body.Success)

	data := body.Data.(map[string]interface{})
	assert.Equal(t, float64(1), data["page"])
	assert.Equal(t, float64(10), data["pageSize"])
	assert.Equal(t, float64(25), data["total"])
}
