package validator

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.eagleview.com/engineering/assess-platform-library/util"
	"github.eagleview.com/engineering/symphony-service/commons/enums"
)

func initStructValidation() (*validator.Validate, ut.Translator) {
	translator := en.New()
	uni := ut.New(translator, translator)

	trans, _ := uni.GetTranslator("en")
	v := validator.New()

	//Register the default translator
	en_translations.RegisterDefaultTranslations(v, trans)

	//Register the tag name available in json
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	_ = v.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	return v, trans
}

func ValidateCallOutRequest(ctx context.Context, data interface{}) error {

	v, trans := initStructValidation()

	_ = v.RegisterTranslation("required_if", trans, func(ut ut.Translator) error {
		return ut.Add("required_if", "{0} is a required field based on the given input", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required_if", fe.Field())
		return t
	})

	_ = v.RegisterValidation("httpMethod", func(fl validator.FieldLevel) bool {
		_, ok := util.FindInStringArray(enums.RequestMethodList(), fl.Field().String(), true)
		return ok
	})

	_ = v.RegisterTranslation("httpMethod", trans, func(ut ut.Translator) error {
		return ut.Add("httpMethod", "invalid http request method", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("httpMethod", fe.Field())
		return t
	})

	_ = v.RegisterValidation("callTypes", func(fl validator.FieldLevel) bool {
		_, ok := util.FindInStringArray(enums.CallTypeList(), fl.Field().String(), true)
		return ok
	})

	_ = v.RegisterTranslation("callTypes", trans, func(ut ut.Translator) error {
		return ut.Add("callTypes", "unsupported calltype", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("callTypes", fe.Field())
		return t
	})

	_ = v.RegisterValidation("authType", func(fl validator.FieldLevel) bool {
		_, ok := util.FindInStringArray(enums.AuthTypeList(), fl.Field().String(), true)
		return ok
	})

	_ = v.RegisterTranslation("authType", trans, func(ut ut.Translator) error {
		return ut.Add("authType", "unsupported authentication type", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("authType", fe.Field())
		return t
	})

	err := v.Struct(data)
	errs := translateError(err, trans)
	return combinedError(errs)
}

func ValidateInvokeSfnRequest(ctx context.Context, data interface{}) error {
	v, trans := initStructValidation()

	_ = v.RegisterValidation("source", func(fl validator.FieldLevel) bool {
		_, ok := util.FindInStringArray(enums.SourcesList(), fl.Field().String(), true)
		return ok
	})

	_ = v.RegisterTranslation("source", trans, func(ut ut.Translator) error {
		return ut.Add("source", "unsupported authentication type", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("source", fe.Field())
		return t
	})

	err := v.Struct(data)
	errs := translateError(err, trans)
	return combinedError(errs)
}

func ValidateCallBackRequest(ctx context.Context, data interface{}) error {
	v, trans := initStructValidation()

	_ = v.RegisterValidation("taskStatus", func(fl validator.FieldLevel) bool {
		_, ok := util.FindInStringArray(enums.TaskStatusList(), fl.Field().String(), true)
		return ok
	})

	_ = v.RegisterTranslation("taskStatus", trans, func(ut ut.Translator) error {
		return ut.Add("taskStatus", "invalid status", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("taskStatus", fe.Field())
		return t
	})

	err := v.Struct(data)
	errs := translateError(err, trans)
	return combinedError(errs)
}

func ValidateSim2PDWRequest(ctx context.Context, data interface{}) error {
	v, trans := initStructValidation()
	err := v.Struct(data)
	errs := translateError(err, trans)
	return combinedError(errs)
}

//translateError Translate the error
func translateError(err error, trans ut.Translator) (errs []error) {
	if err == nil {
		return nil
	}
	validatorErrs := err.(validator.ValidationErrors)
	for _, e := range validatorErrs {
		translatedErr := fmt.Errorf(e.Translate(trans))
		errs = append(errs, translatedErr)
	}
	return errs
}

// combinedError combine all the error messages
func combinedError(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	msg := []string{}
	for _, e := range errs {
		msg = append(msg, e.Error())
	}
	return errors.New(strings.Join(msg, ","))
}
