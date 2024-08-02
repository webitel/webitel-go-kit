package model

type AppError interface {
	GetStatusCode() int
	GetDetailedError() string
	GetId() string
	Error() string
}
