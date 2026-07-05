package captcha

type captchaResponse struct {
	CaptchaID string `json:"captcha_id"`
	Image     string `json:"image"`
}
