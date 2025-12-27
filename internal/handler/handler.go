package handler

import (
	"GH-Server/internal/database"
	"GH-Server/internal/fileServer"
	"GH-Server/pkg/utils"
)

type Handler struct {
	database.Service
	fileServer.RustFSService
	UserHandler
	*utils.StructValidator
}
