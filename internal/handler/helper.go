package handler

import "github.com/gin-gonic/gin"

// Response 标准 API 响应信封。
type Response struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
	Meta    *Meta      `json:"meta,omitempty"`
}

// ErrorInfo 错误详情。
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta 分页元数据，非分页接口省略。
type Meta struct {
	Page       int `json:"page,omitempty"`
	PerPage    int `json:"per_page,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// OK 返回成功响应（200）。
func OK(c *gin.Context, data any) {
	c.JSON(200, Response{
		Success: true,
		Data:    data,
	})
}

// OKPage 返回成功响应带分页元数据。
func OKPage(c *gin.Context, data any, meta *Meta) {
	c.JSON(200, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Fail 返回业务错误响应。
func Fail(c *gin.Context, code, message string) {
	c.JSON(200, Response{
		Success: false,
		Error:   &ErrorInfo{Code: code, Message: message},
	})
}
