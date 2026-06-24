package errs

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       int    `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code int, message string, httpStatus int) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus}
}

func Wrap(code int, message string, httpStatus int, err error) *AppError {
	return &AppError{Code: code, Message: message, HTTPStatus: httpStatus, Err: err}
}

func FromError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return &AppError{
		Code:       CodeInternal,
		Message:    "internal server error",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

var (
	ErrBadRequest = New(CodeValidation, "请求参数错误", http.StatusBadRequest)
	ErrNotFound   = New(CodeNotFound, "资源未找到", http.StatusNotFound)
	ErrInternal   = New(CodeInternal, "服务器内部错误", http.StatusInternalServerError)
)
