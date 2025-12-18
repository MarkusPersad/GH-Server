package utils

import (
	"GH-Server/pkg/zaplog"
	"fmt"

	"github.com/go-playground/validator/v10"
)

type StructValidator struct {
	Validator *validator.Validate
}

func (sv *StructValidator) Validate(body any) error {
	if err := sv.Validator.Struct(body); err != nil {
		zaplog.Zap.Error(fmt.Sprintf("validate failed: %v", err))
		return err
	}
	return nil
}
