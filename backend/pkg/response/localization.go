package response

import (
	"errors"
	"github.com/CIPFZ/video-cms/pkg/errcode"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

type Localizer func(string, map[string]interface{}) (string, bool)

const localizerKey = "cms.response.localizer"

func SetLocalizer(c *gin.Context, localizer Localizer) { c.Set(localizerKey, localizer) }

func lookup(c *gin.Context, id string, data map[string]interface{}) (string, bool) {
	value, exists := c.Get(localizerKey)
	if !exists {
		return "", false
	}
	localizer, ok := value.(Localizer)
	if !ok {
		return "", false
	}
	return localizer(id, data)
}

func translateKnown(c *gin.Context, message string) (string, bool) {
	if key, exists := messageKeys[message]; exists {
		return lookup(c, key, nil)
	}
	if value, ok := lookup(c, message, nil); ok {
		return value, true
	}
	// Context such as "update failed: role does not exist" retains both messages.
	if prefix, detail, ok := strings.Cut(message, ": "); ok {
		if translated, found := translateKnown(c, prefix); found {
			if explanation, known := translateKnown(c, detail); known {
				return translated + ": " + explanation, true
			}
			return translated, true
		}
	}
	if strings.HasPrefix(message, "file extension ") {
		return lookup(c, "validation.fileType", nil)
	}
	if strings.HasPrefix(message, "file size ") {
		return lookup(c, "validation.fileSize", nil)
	}
	return "", false
}

func LocalizeMessage(c *gin.Context, message string) string {
	if _, enabled := c.Get(localizerKey); !enabled {
		return message
	}
	if translated, ok := translateKnown(c, message); ok {
		return translated
	}
	value, _ := lookup(c, "api.internalError", nil)
	return value
}

// FailWithValidation translates field errors instead of exposing Gin's English
// validator dump. JSON syntax/type errors receive a localized general message.
func FailWithValidation(err error, c *gin.Context) {
	var fields validator.ValidationErrors
	if errors.As(err, &fields) && len(fields) > 0 {
		field := fields[0]
		label, ok := lookup(c, "field."+field.StructField(), nil)
		if !ok {
			label = field.Field()
		}
		key := "validation." + field.Tag()
		if field.Kind() == reflect.String && (field.Tag() == "min" || field.Tag() == "max") {
			key += "Length"
		}
		data := map[string]interface{}{"Field": label, "Param": field.Param()}
		message, ok := lookup(c, key, data)
		if !ok {
			message, ok = lookup(c, "validation.field", data)
		}
		if ok {
			c.JSON(200, Response{Code: errcode.InvalidParams.Code, Msg: message})
			return
		}
	}
	message, ok := lookup(c, "validation.invalid", nil)
	if !ok {
		message = errcode.InvalidParams.Msg
	}
	c.JSON(200, Response{Code: errcode.InvalidParams.Code, Msg: message})
}
