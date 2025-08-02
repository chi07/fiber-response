package response

import "github.com/gofiber/fiber/v2"

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrorResponse struct {
	Success bool         `json:"success"`
	Error   string       `json:"error"`
	Fields  []FieldError `json:"fields"`
}

type Pagination struct {
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Data     interface{} `json:"data"`
}

func Success(c *fiber.Ctx, status int, data interface{}) error {
	if status == fiber.StatusNoContent {
		return c.SendStatus(status)
	}
	return c.Status(status).JSON(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func OK(c *fiber.Ctx, data interface{}) error {
	return Success(c, fiber.StatusOK, data)
}

func Created(c *fiber.Ctx, data interface{}) error {
	return Success(c, fiber.StatusCreated, data)
}

func NoContent(c *fiber.Ctx) error {
	return Success(c, fiber.StatusNoContent, nil)
}

func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Success: false,
		Error:   message,
	})
}

func ValidationError(c *fiber.Ctx, fields []FieldError) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(ValidationErrorResponse{
		Success: false,
		Error:   "Validation failed",
		Fields:  fields,
	})
}

func Paginate(c *fiber.Ctx, page, pageSize, total int, data interface{}) error {
	return c.Status(fiber.StatusOK).JSON(SuccessResponse{
		Success: true,
		Data: Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
			Data:     data,
		},
	})
}
