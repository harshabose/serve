package room

import "github.com/harshabose/skyline_sonata/serve/pkg/interceptor"

type state struct {
	id     string
	writer interceptor.Writer
	reader interceptor.Reader
}
