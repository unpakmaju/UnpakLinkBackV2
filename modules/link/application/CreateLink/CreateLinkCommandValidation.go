package application

import (
	"errors"
	"net/url"
	"strings"

	helper "UnpakSiamida/common/helper"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

func isSafeHttpURLRule() validation.RuleFunc {
	return func(value interface{}) error {
		s, _ := value.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		u, err := url.ParseRequestURI(s)
		if err != nil || u.Host == "" {
			return errors.New("LongUrl must be a valid URL")
		}
		scheme := strings.ToLower(u.Scheme)
		if scheme != "http" && scheme != "https" {
			return errors.New("LongUrl scheme must be http or https")
		}
		return nil
	}
}

func CreateLinkCommandValidation(cmd CreateLinkCommand) error {
	return validation.ValidateStruct(&cmd,
		validation.Field(&cmd.ShortUrl,
			validation.Required.Error("ShortUrl cannot be blank"),
			validation.By(helper.NoXSSFullScanWithDecode()),
		),
		validation.Field(&cmd.LongUrl,
			validation.Required.Error("LongUrl cannot be blank"),
			validation.By(isSafeHttpURLRule()),
			validation.By(helper.NoXSSFullScanWithDecode()),
		),
		validation.Field(&cmd.Creator,
			validation.Required.Error("Creator cannot be blank"),
			validation.By(helper.NoXSSFullScanWithDecode()),
		),
	)
}
