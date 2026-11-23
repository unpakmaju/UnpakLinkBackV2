package infrastructure

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"UnpakSiamida/common/domain"

	"github.com/gofiber/fiber/v2"
)

// ResponseError representasi error yang dikirim ke client
type ResponseError struct {
	Code    string      `json:"code"`
	Message interface{} `json:"message"` // string atau object/map
	Trace   interface{} `json:"trace,omitempty"`
}

// implementasi interface error
func (e *ResponseError) Error() string {
	switch m := e.Message.(type) {
	case string:
		return m
	default:
		return fmt.Sprintf("%s: %v", e.Code, m)
	}
}

// NewResponseError membuat ResponseError baru dengan trace otomatis (hanya saat APP_DEBUG=true)
func NewResponseError(code string, message interface{}) *ResponseError {
	return &ResponseError{
		Code:    code,
		Message: message,
		Trace:   getTrace(),
	}
}

// NewInternalError untuk error unknown/wrapped
func NewInternalError(err error) *ResponseError {
	if err == nil {
		return nil
	}

	log.Printf("[INTERNAL_ERROR] %v", err)

	msg := "Internal server error"
	if os.Getenv("APP_DEBUG") == "true" {
		msg = err.Error()
	}

	return &ResponseError{
		Code:    "INTERNAL_SERVER_ERROR",
		Message: msg,
		Trace:   getTrace(),
	}
}

// mapCodeToStatus mapping code string ke http status
func mapCodeToStatus(code string) int {
	switch {
	case strings.HasSuffix(code, ".Validation"):
		return 400
	case strings.HasSuffix(code, ".InvalidPassword"), strings.HasSuffix(code, ".Unauthorized"):
		return 401
	case strings.HasSuffix(code, ".Forbidden"):
		return 403
	case strings.HasSuffix(code, ".NotFound"):
		return 404
	case strings.HasSuffix(code, ".Conflict"):
		return 409
	default:
		return 400
	}
}

// mapDomainErrorToStatus mapping domain.ErrorType ke http status
func mapDomainErrorToStatus(errType domain.ErrorType) int {
	switch errType {
	case domain.Validation:
		return 400
	case domain.NotFound:
		return 404
	case domain.Conflict:
		return 409
	default:
		return 500
	}
}

// HandleError mengeksekusi error ke response JSON fiber
func HandleError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	trace := getTrace()

	// 1) ResponseError
	var respErr *ResponseError
	if errors.As(err, &respErr) {
		re := &ResponseError{
			Code:    respErr.Code,
			Message: respErr.Message,
			Trace:   trace,
		}
		status := mapCodeToStatus(respErr.Code)
		return c.Status(status).JSON(re)
	}

	// 2) domain.Error
	var derr domain.Error
	if errors.As(err, &derr) {
		re := &ResponseError{
			Code:    derr.Code,
			Message: derr.Description,
			Trace:   trace,
		}
		status := mapDomainErrorToStatus(derr.Type)
		return c.Status(status).JSON(re)
	}

	// 3) fallback: unknown/wrapped error -> internal server error
	return c.Status(500).JSON(NewInternalError(err))
}

// helper ambil trace (hanya dikembalikan ke response jika APP_DEBUG=true)
func getTrace() interface{} {
	if os.Getenv("APP_DEBUG") != "true" {
		return nil
	}
	const maxDepth = 10
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])

	var traceLines []string
	for {
		frame, more := frames.Next()
		traceLines = append(traceLines, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return strings.Join(traceLines, "\n")
}
