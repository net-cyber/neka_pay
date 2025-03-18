package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/net-cyber/neka_pay/util"
)

var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
      if currency, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	  }
	  return false
}