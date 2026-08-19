package http

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/LerianStudio/fetcher/v2/pkg"
	"github.com/LerianStudio/fetcher/v2/pkg/datasource/hostsafety"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	cn "github.com/LerianStudio/fetcher/v2/pkg/constant"
	en2 "github.com/go-playground/validator/v10/translations/en"
)

var (
	cachedValidator   *validator.Validate
	cachedTranslator  ut.Translator
	validatorInitOnce sync.Once
	validatorInitErr  error
)

// getValidator returns the cached validator and translator, initializing them on first call.
func getValidator() (*validator.Validate, ut.Translator, error) {
	validatorInitOnce.Do(func() {
		cachedValidator, cachedTranslator, validatorInitErr = newValidator()
	})

	return cachedValidator, cachedTranslator, validatorInitErr
}

// ErrValidatorInit is returned when the request validator cannot be initialized.
// Callers should map this to a 5xx response (server-side failure), not 400.
var ErrValidatorInit = errors.New("validator initialization failed")

// ValidateStruct validates a struct against defined validation rules, using the validator package.
func ValidateStruct(s any) error {
	v, trans, err := getValidator()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrValidatorInit, err)
	}

	k := reflect.ValueOf(s).Kind()
	if k == reflect.Pointer {
		k = reflect.ValueOf(s).Elem().Kind()
	}

	if k != reflect.Struct {
		return nil
	}

	err = v.Struct(s)
	if err != nil {
		for _, fieldError := range err.(validator.ValidationErrors) {
			switch fieldError.Tag() {
			case "keymax":
				return pkg.ValidateBusinessError(cn.ErrMetadataKeyLengthExceeded, "", fieldError.Translate(trans), fieldError.Param())
			case "valuemax":
				return pkg.ValidateBusinessError(cn.ErrMetadataValueLengthExceeded, "", fieldError.Translate(trans), fieldError.Param())
			case "nonested":
				return pkg.ValidateBusinessError(cn.ErrInvalidMetadataNesting, "", fieldError.Translate(trans))
			case "safe_host":
				// SSRF host safety guard rejected an IP literal at DTO layer.
				// Generic-message contract: do NOT echo the host or reveal
				// which range matched (see docs/PROJECT_RULES.md § "Error Surface").
				return pkg.ValidateBusinessError(cn.ErrForbiddenHost, "connection")
			}
		}

		// Return the value (not a pointer): WithError downstream uses
		// errors.As with a value-type target (pkg.ValidationKnownFieldsError),
		// which requires type identity — a *pkg.ValidationKnownFieldsError
		// would fall through to the InternalServerError default branch and
		// render as HTTP 500 instead of 400.
		return malformedRequestErr(err.(validator.ValidationErrors), trans)
	}

	return nil
}

func fields(errs validator.ValidationErrors, trans ut.Translator) pkg.FieldValidations {
	l := len(errs)
	if l > 0 {
		fields := make(pkg.FieldValidations, l)
		for _, e := range errs {
			fields[e.Field()] = e.Translate(trans)
		}

		return fields
	}

	return nil
}

func fieldsRequired(myMap pkg.FieldValidations) pkg.FieldValidations {
	result := make(pkg.FieldValidations)

	for key, value := range myMap {
		if strings.Contains(value, "required") {
			result[key] = value
		}
	}

	return result
}

func malformedRequestErr(err validator.ValidationErrors, trans ut.Translator) pkg.ValidationKnownFieldsError {
	invalidFieldsMap := fields(err, trans)

	requiredFields := fieldsRequired(invalidFieldsMap)

	var vErr pkg.ValidationKnownFieldsError

	if !errors.As(pkg.ValidateBadRequestFieldsError(requiredFields, invalidFieldsMap, "", make(map[string]any)), &vErr) {
		return pkg.ValidationKnownFieldsError{
			Code:    "VALIDATION_ERROR",
			Title:   "Validation Error",
			Message: "request validation failed",
		}
	}

	return vErr
}

//nolint:ireturn
func newValidator() (*validator.Validate, ut.Translator, error) {
	locale := en.New()
	uni := ut.New(locale, locale)

	trans, _ := uni.GetTranslator("en")

	v := validator.New()

	if err := en2.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, nil, fmt.Errorf("failed to register default translations: %w", err)
	}

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}

		return name
	})

	_ = v.RegisterValidation("keymax", validateMetadataKeyMaxLength)
	_ = v.RegisterValidation("nonested", validateMetadataNestedValues)
	_ = v.RegisterValidation("valuemax", validateMetadataValueMaxLength)
	_ = v.RegisterValidation("safe_host", validateSafeHost)

	_ = v.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} is a required field", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", formatErrorFieldName(fe.Namespace()))

		return t
	})

	_ = v.RegisterTranslation("gte", trans, func(ut ut.Translator) error {
		return ut.Add("gte", "{0} must be {1} or greater", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("gte", formatErrorFieldName(fe.Namespace()), fe.Param())

		return t
	})

	_ = v.RegisterTranslation("eq", trans, func(ut ut.Translator) error {
		return ut.Add("eq", "{0} is not equal to {1}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("eq", formatErrorFieldName(fe.Namespace()), fe.Param())

		return t
	})

	_ = v.RegisterTranslation("keymax", trans, func(ut ut.Translator) error {
		return ut.Add("keymax", "{0}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("keymax", formatErrorFieldName(fe.Namespace()))

		return t
	})

	_ = v.RegisterTranslation("valuemax", trans, func(ut ut.Translator) error {
		return ut.Add("valuemax", "{0}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("valuemax", formatErrorFieldName(fe.Namespace()))

		return t
	})

	_ = v.RegisterTranslation("nonested", trans, func(ut ut.Translator) error {
		return ut.Add("nonested", "{0}", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("nonested", formatErrorFieldName(fe.Namespace()))

		return t
	})

	return v, trans, nil
}

// validateMetadataNestedValues checks if there are nested metadata structures
func validateMetadataNestedValues(fl validator.FieldLevel) bool {
	return fl.Field().Kind() != reflect.Map
}

// validateSafeHost rejects connection hosts whose literal IP falls in a
// denylisted CIDR range when the SSRF host safety guard is enabled. Hostnames
// always pass at this layer — DNS resolution and CIDR matching for hostnames
// is the responsibility of the factory-level guard (hostsafety.ValidateHostForConnection),
// which runs just before the database dial.
func validateSafeHost(fl validator.FieldLevel) bool {
	return hostsafety.ValidateSafeHostString(fl.Field().String())
}

// validateMetadataKeyMaxLength checks if metadata key (always a string) length is allowed
func validateMetadataKeyMaxLength(fl validator.FieldLevel) bool {
	limitParam := fl.Param()

	limit := 100 // default limit if no param configured

	if limitParam != "" {
		if parsedParam, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedParam
		}
	}

	return len(fl.Field().String()) <= limit
}

// validateMetadataValueMaxLength checks metadata value max length
func validateMetadataValueMaxLength(fl validator.FieldLevel) bool {
	limitParam := fl.Param()

	limit := 2000 // default limit if no param configured

	if limitParam != "" {
		if parsedParam, err := strconv.Atoi(limitParam); err == nil {
			limit = parsedParam
		}
	}

	var value string

	switch fl.Field().Kind() {
	case reflect.Int:
		value = strconv.Itoa(int(fl.Field().Int()))
	case reflect.Float64:
		value = strconv.FormatFloat(fl.Field().Float(), 'f', -1, 64)
	case reflect.String:
		value = fl.Field().String()
	case reflect.Bool:
		value = strconv.FormatBool(fl.Field().Bool())
	default:
		return false
	}

	return len(value) <= limit
}

// formatErrorFieldNameRegex is a pre-compiled regex for extracting field names from validator namespaces.
var formatErrorFieldNameRegex = regexp.MustCompile(`\.(.+)$`)

// formatErrorFieldName extracts the field name from a validator namespace string (e.g., "SomeStruct.field" -> "field").
func formatErrorFieldName(text string) string {
	matches := formatErrorFieldNameRegex.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	return text
}
