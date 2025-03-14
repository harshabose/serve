package interceptor

import "github.com/coder/websocket"

type Chain struct {
	interceptors []Interceptor
}

func CreateChain(interceptors []Interceptor) *Chain {
	return &Chain{interceptors: interceptors}
}

func (chain *Chain) BindSocketConnection(connection *websocket.Conn) error {
	for _, interceptor := range chain.interceptors {
		if err := interceptor.BindSocketConnection(connection); err != nil {
			return err
		}
	}
	return nil
}

func (chain *Chain) BindSocketWriter(writer Writer) Writer {
	for _, interceptor := range chain.interceptors {
		writer = interceptor.BindSocketWriter(writer)
	}

	return writer
}

func (chain *Chain) BindSocketReader(reader Reader) Reader {
	for _, interceptor := range chain.interceptors {
		reader = interceptor.BindSocketReader(reader)
	}

	return reader
}

func (chain *Chain) UnBindSocketConnection(connection *websocket.Conn) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnBindSocketConnection(connection)
	}
}

func (chain *Chain) UnBindSocketWriter(writer Writer) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnBindSocketWriter(writer)
	}
}

func (chain *Chain) UnBindSocketReader(reader Reader) {
	for _, interceptor := range chain.interceptors {
		interceptor.UnBindSocketReader(reader)
	}
}

func (chain *Chain) Close() error {
	var errs []error
	for _, interceptor := range chain.interceptors {
		errs = append(errs, interceptor.Close())
	}

	return flattenErrs(errs)
}
