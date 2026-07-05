// Package captcha 提供图形验证码的生成与校验能力，与具体业务解耦。
// 基于 base64Captcha 库，默认使用数字验证码 + 内存存储。
package captcha

import (
	"github.com/mojocn/base64Captcha"
)

// Generator 验证码生成与校验接口。
type Generator interface {
	// Generate 生成验证码，返回 id 和 Base64 编码的图片。
	Generate() (id string, base64Image string, err error)
	// Verify 校验验证码，answer 大小写不敏感。
	Verify(id, answer string) bool
}

// Config 验证码配置，零值可用。
type Config struct {
	Width  int // 图片宽度，默认 240
	Height int // 图片高度，默认 80
	Length int // 验证码长度，默认 6
}

func (c *Config) normalized() (width, height, length int) {
	width, height, length = c.Width, c.Height, c.Length
	if width <= 0 {
		width = 240
	}
	if height <= 0 {
		height = 80
	}
	if length <= 0 {
		length = 6
	}
	return
}

// New 创建验证码生成器，使用内存存储 + 数字驱动。
func New(cfg Config) Generator {
	return NewWithStore(cfg, base64Captcha.NewMemoryStore(1024, 5*60))
}

// NewWithStore 创建验证码生成器，使用自定义存储（如 Redis）。
func NewWithStore(cfg Config, store base64Captcha.Store) Generator {
	width, height, length := cfg.normalized()
	driver := base64Captcha.NewDriverDigit(height, width, length, 0.7, 80)
	return &generator{
		captcha: base64Captcha.NewCaptcha(driver, store),
		store:   store,
	}
}

type generator struct {
	captcha *base64Captcha.Captcha
	store   base64Captcha.Store
}

func (g *generator) Generate() (string, string, error) {
	id, b64s, _, err := g.captcha.Generate()
	return id, b64s, err
}

func (g *generator) Verify(id, answer string) bool {
	return g.store.Verify(id, answer, true)
}
