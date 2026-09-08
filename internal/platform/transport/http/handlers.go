package http

type IHandler interface{}

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}
