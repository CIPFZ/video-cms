package middleware

import (
	corei18n "github.com/CIPFZ/video-cms/internal/core/i18n"
	"github.com/CIPFZ/video-cms/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/text/language"
)

// Supported languages are intentionally limited to Simplified Chinese and English.
func RequestLanguage(header string) string {
	tags, _, err := language.ParseAcceptLanguage(header)
	if err != nil || len(tags) == 0 {
		return "zh-CN"
	}
	_, index, confidence := language.NewMatcher([]language.Tag{language.SimplifiedChinese, language.AmericanEnglish}).Match(tags...)
	if confidence == language.No || index == 0 {
		return "zh-CN"
	}
	return "en-US"
}

func I18nMiddleware(service *corei18n.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := RequestLanguage(c.GetHeader("Accept-Language"))
		c.Header("Content-Language", lang)
		c.Header("Vary", "Accept-Language")
		if service != nil {
			response.SetLocalizer(c, func(id string, data map[string]interface{}) (string, bool) {
				return service.Lookup(lang, id, data)
			})
		}
		c.Next()
	}
}
