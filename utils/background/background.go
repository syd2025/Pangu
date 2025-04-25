package background

import "example.com/myapp/utils/jsonlog"

type Background struct {
}

func New(logger *jsonlog.Logger) *Background {
	return &Background{}
}
