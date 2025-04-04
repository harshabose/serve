# WebSocket Interceptor Framework

A flexible, extensible WebSocket middleware framework for building secure, scalable real-time communication systems.

## Overview

This framework provides an interceptor pattern implementation for WebSocket connections, allowing you to add middleware-like functionality to your WebSocket applications. Rather than the traditional approach of building communication mechanisms first and adding middleware later, this framework inverts that pattern by starting with interceptors that can work with any underlying communication stack.

## Key Features

- **Middleware for WebSockets**: Add cross-cutting concerns like encryption, authentication, logging, and compression to WebSocket connections
- **Protocol-Based Message Routing**: Nest messages with protocol identifiers for flexible routing
- **Connection Lifecycle Management**: Proper binding and cleanup of resources
- **Transport Agnostic**: Works with any communication system that supports read/write operations
- **Composable Architecture**: Chain interceptors together to build complex functionality
- **Extensible Design**: Easy to add new interceptors without modifying existing code

## Use Cases

- **Secure Communications**: Add encryption layers to WebSocket traffic
- **Real-Time Monitoring**: Log and analyze message patterns
- **Protocol Translation**: Adapt between different message formats
- **Access Control**: Implement authentication and authorization
- **Rate Limiting**: Protect your system from excessive traffic
- **Message Transformation**: Compress, validate, or transform messages in transit

## Architecture

The framework is built around several key interfaces:

- **Interceptor**: The core interface that all interceptors implement
- **Connection**: Represents a WebSocket connection
- **Writer**: Handles outgoing messages
- **Reader**: Handles incoming messages

Interceptors can be chained together to form a processing pipeline for messages. Each interceptor can examine, modify, or route messages based on their protocol and content.

## Message Structure

Messages use a flexible nested structure:

```go
type BaseMessage struct {
    Header
    Payload json.RawMessage `json:"payload"`
}

type Header struct {
    SenderID   string   `json:"source_id"`
    ReceiverID string   `json:"destination_id"`
    Protocol   Protocol `json:"protocol"`
}
```

The Protocol field identifies the type of message and determines how the Payload should be processed. Messages can be nested to arbitrary depth, with each layer having its own protocol identifier.

## Example: Encryption Interceptor

An encryption interceptor can seamlessly add security to your WebSocket communications:

```go
func (i *EncryptionInterceptor) InterceptSocketWriter(writer interceptor.Writer) interceptor.Writer {
    return interceptor.WriterFunc(func(conn interceptor.Connection, messageType websocket.MessageType, m message.Message) error {
        // Check if this connection has encryption enabled
        state, exists := i.getState(conn)
        if !exists {
            return writer.Write(conn, messageType, m)
        }

        // Encrypt the message
        encrypted, err := state.encryptor.Encrypt(m.Message().SenderID, m.Message().ReceiverID, m)
        if err != nil {
            return writer.Write(conn, messageType, m)
        }
        
        // Send the encrypted message
        return writer.Write(conn, messageType, encrypted)
    })
}
```

## Benefits Over Traditional Approaches

This "interceptors-first" approach offers several advantages:

1. **Separation of Concerns**: Clean separation between communication mechanics and business logic
2. **Framework Agnosticism**: Swap underlying communication technology without changing application code
3. **Easier Testing**: Test each interceptor in isolation
4. **Adaptability**: Add new functionality (encryption, logging, etc.) without modifying existing code
5. **Protocol Evolution**: Change protocols without widespread codebase changes
6. **Reduced Technical Debt**: Keep cross-cutting concerns localized to specific interceptors

## Getting Started

[Installation and basic usage instructions would go here]

## Example Usage

[Code examples showing how to set up and use interceptors would go here]

## License

[License information would go here]