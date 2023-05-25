package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"net/url"
	"reflect"
)

type BinderWithURLDecoding struct {
	binder *echo.DefaultBinder
}

func (b *BinderWithURLDecoding) Bind(i interface{}, c echo.Context) error {
	err := b.binder.Bind(i, c)
	if err != nil {
		return err
	}

	return b.DecodeURLEncodedParams(i)
}

func (b *BinderWithURLDecoding) DecodeURLEncodedParams(i any) error {
	value := reflect.ValueOf(i)
	structValue := value.Elem()
	structType := structValue.Type()

	for i := 0; i < structValue.NumField(); i++ {
		field := structValue.Field(i)
		fieldType := structType.Field(i)

		if field.Kind() != reflect.String {
			//skip non string fields, because they can't be url encoded
			continue
		}

		tagValue := fieldType.Tag.Get("param")
		if tagValue == "" {
			tagValue = fieldType.Tag.Get("query")
		}

		if tagValue == "" || tagValue == "-" || !field.CanSet() {
			//skip empty or other tags
			continue
		}

		val := field.String()
		unescapedVal, err := url.QueryUnescape(val)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid escape sequence").SetInternal(err)
		}

		field.SetString(unescapedVal)
	}

	return nil
}
