package encrypt

import "context"

type state struct {
	id        string
	encryptor encryptor
	cancel    context.CancelFunc
	ctx       context.Context
}
