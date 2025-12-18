package handler

import (
	"GH-Server/internal/database"
	"GH-Server/pkg/utils"
)

type Handler struct {
	database.Service
	UserHandler
	*utils.StructValidator
}
