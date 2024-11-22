package domain

import "mime/multipart"

type File struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}
