package interceptor

type Chain struct {
	interceptors []Interceptor
}

func CreateChain(interceptors []Interceptor) *Chain {
	return &Chain{interceptors: interceptors}
}

func (chain *Chain) BindIncoming(reader IncomingReader) IncomingReader {
	for _, interceptor := range chain.interceptors {
		interceptor.BindIncoming(reader)
	}

	return reader
}

func (chain *Chain) BindOutgoing(writer OutgoingWriter) OutgoingWriter {
	for _, interceptor := range chain.interceptors {
		interceptor.BindOutgoing(writer)
	}

	return writer
}

func (chain *Chain) BindConnection(connection Connection) Connection {
	for _, interceptor := range chain.interceptors {
		interceptor.BindConnection(connection)
	}

	return connection
}

func (chain *Chain) Close() error {
	var errs []error
	for _, interceptor := range chain.interceptors {
		errs = append(errs, interceptor.Close())
	}

	return flattenErrs(errs)
}
