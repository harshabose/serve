package interceptor

type NoInterceptor struct{}

func (interceptor *NoInterceptor) BindIncoming(reader IncomingReader) IncomingReader {
	return reader
}

func (interceptor *NoInterceptor) BindOutgoing(writer OutgoingWriter) OutgoingWriter {
	return writer
}

func (interceptor *NoInterceptor) BindConnection(connection Connection) Connection {
	return connection
}

func (interceptor *NoInterceptor) Close() error {
	return nil
}
