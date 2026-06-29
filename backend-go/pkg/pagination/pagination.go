// Package pagination 实现分页逻辑。
//
// 参数: ?page=1&size=10（size 上限 200）。
// 返回结构由 pkg/response.PaginatedData 定义。
package pagination

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	DefaultPage = 1
	DefaultSize = 10
	MaxSize     = 200
)

// Params 从 query string 解析分页参数。
type Params struct {
	Page int `form:"page" binding:"omitempty,min=1"`
	Size int `form:"size" binding:"omitempty,min=1,max=200"`
}

// Parse 从 gin context 解析分页参数，缺省值 page=1, size=10。
func Parse(c *gin.Context) Params {
	p := Params{
		Page: DefaultPage,
		Size: DefaultSize,
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Page = n
		}
	}
	if v := c.Query("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > MaxSize {
				n = MaxSize
			}
			p.Size = n
		}
	}
	return p
}

// Offset 返回 GORM 查询的 offset。
func (p Params) Offset() int {
	return (p.Page - 1) * p.Size
}

// Paginate 对 *gorm.DB 施加分页（返回带 LIMIT/OFFSET 的新 session）。
// 用法: db.Scopes(pagination.Paginate(params)).Find(&items)
func Paginate(p Params) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(p.Offset()).Limit(p.Size)
	}
}

// NextURL 构造下一页的完整 URL。
// 无下一页返回空字符串。
func NextURL(c *gin.Context, page, size int, count int64) string {
	totalPages := int(math.Ceil(float64(count) / float64(size)))
	if page >= totalPages {
		return ""
	}
	return buildPageURL(c, page+1, size)
}

// PreviousURL 构造上一页的完整 URL。无上一页返回空字符串。
func PreviousURL(c *gin.Context, page, size int) string {
	if page <= 1 {
		return ""
	}
	return buildPageURL(c, page-1, size)
}

func buildPageURL(c *gin.Context, page, size int) string {
	scheme := "http"
	if c.Request.TLS != nil || strings.HasPrefix(c.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	// 复制 query 参数，覆盖 page/size
	q := c.Request.URL.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	return fmt.Sprintf("%s://%s%s?%s",
		scheme,
		c.Request.Host,
		c.Request.URL.Path,
		q.Encode(),
	)
}
