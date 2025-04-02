package encrypt

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
	"github.com/harshabose/skyline_sonata/serve/pkg/message"
)

type Interceptor struct {
	interceptor.NoOpInterceptor
	states map[interceptor.Connection]*state
	mux    sync.Mutex
	ctx    context.Context
}

func (encrypt *Interceptor) BindSocketConnection(connection *websocket.Conn) error {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	// TODO: Exchange keys with the peer using a key exchange protocol like Diffie-Hellman
	// TODO: Store different keys for encryption and decryption

	go encrypt.loop(connection)

	return nil
}

func (encrypt *Interceptor) BindSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(connection interceptor.Connection, messageType websocket.MessageType, message message.Message) error {
		state, exists := encrypt.states[connection]
		if !exists {
			return
		}
	})
	return interceptor.WriterFunc(func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error {
		encrypt.mux.Lock()
		collection, exists := encrypt.states[conn]
		encrypt.mux.Unlock()

		if !exists || collection.encryptor == nil {
			// No encryption configured for this connection yet
			// Pass through unencrypted
			return writer.Write(conn, messageType, data)
		}

		// Generate a nonce for this message
		nonce := make([]byte, 12) // GCM typically uses a 12-byte nonce
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return fmt.Errorf("failed to generate nonce: %w", err)
		}

		// Encrypt the data
		encryptor := collection.encryptor
		sessionID := collection.sessionID
		encryptedData := encryptor.Seal(nil, nonce, data, sessionID)

		// Format the encrypted message:
		// [2-byte nonce length][nonce][encrypted data]
		finalData := make([]byte, 2+len(nonce)+len(encryptedData))
		binary.BigEndian.PutUint16(finalData[:2], uint16(len(nonce)))
		copy(finalData[2:], nonce)
		copy(finalData[2+len(nonce):], encryptedData)

		// Send the encrypted message
		return writer.Write(conn, messageType, finalData)
	})
}

func (encrypt *Interceptor) BindSocketReader(reader interceptor.Reader) interceptor.Reader {
	return interceptor.ReaderFunc(func(conn *websocket.Conn) (websocket.MessageType, []byte, error) {
		// Read the encrypted message
		messageType, encryptedData, err := reader.Read(conn)
		if err != nil {
			return messageType, encryptedData, err
		}

		encrypt.mux.Lock()
		collection, exists := encrypt.states[conn]
		encrypt.mux.Unlock()

		if !exists || collection.encryptor == nil || len(encryptedData) < 2 {
			// No decryption configured or data too short to be encrypted
			// Pass through as-is
			return messageType, encryptedData, nil
		}

		// Extract nonce length
		nonceLen := binary.BigEndian.Uint16(encryptedData[:2])

		// Ensure we have enough data for the nonce and at least some ciphertext
		if len(encryptedData) < int(2+nonceLen) {
			return messageType, encryptedData, fmt.Errorf("encrypted data too short")
		}

		// Extract nonce and ciphertext
		nonce := encryptedData[2 : 2+nonceLen]
		ciphertext := encryptedData[2+nonceLen:]

		// Decrypt the data
		decryptor := collection.encryptor
		sessionID := collection.sessionID
		plaintext, err := decryptor.Open(nil, nonce, ciphertext, sessionID)
		if err != nil {
			return messageType, encryptedData, fmt.Errorf("decryption failed: %w", err)
		}

		return messageType, plaintext, nil
	})
}

func (encrypt *Interceptor) UnBindSocketConnection(connection *websocket.Conn) {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	collection, exists := encrypt.states[connection]
	if !exists {
		fmt.Println("connection does not exists")
		return
	}

	collection.cancel()
	delete(encrypt.states, connection)
}

func (encrypt *Interceptor) UnBindSocketWriter(_ interceptor.Writer) {}

func (encrypt *Interceptor) UnBindSocketReader(_ interceptor.Reader) {}

func (encrypt *Interceptor) Close() error {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	encrypt.states = make(map[*websocket.Conn]*collection)

	return nil
}

func (encrypt *Interceptor) loop(connection *websocket.Conn) {

	encrypt.mux.Lock()
	collection, exists := encrypt.states[connection]
	if !exists {
		fmt.Println("connection does not exists")
		return
	}
	ctx := collection.ctx
	encrypt.mux.Unlock()

	timer := time.NewTicker(5 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			encrypt.mux.Lock()

			newSessionID := make([]byte, 16)
			if _, err := io.ReadFull(rand.Reader, newSessionID); err != nil {
				fmt.Println("Failed to generate new session messageID:", err)
				continue
			}

			if collection, exists := encrypt.states[connection]; exists {
				collection.sessionID = nil // Keep nil until sending to peer mechanism is set
			}

			// send the update sessionID to peer

			encrypt.mux.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
