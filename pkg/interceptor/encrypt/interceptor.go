package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/coder/websocket"

	"github.com/harshabose/skyline_sonata/serve/pkg/interceptor"
)

type collection struct {
	encryptor cipher.AEAD // For encrypting outgoing messages
	decryptor cipher.AEAD // For decrypting incoming messages
}

type Interceptor struct {
	interceptor.NoOpInterceptor
	collection map[*websocket.Conn]*collection
	mux        sync.Mutex
}

func (encrypt *Interceptor) BindSocketConnection(connection *websocket.Conn) error {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	// Generate a key for this connection
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode encryptor
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM mode: %w", err)
	}
	_ = gcm

	// TODO: Exchange keys with the peer using a key exchange protocol like Diffie-Hellman
	// TODO: Store different keys for encryption and decryption

	encrypt.collection[connection] = &collection{
		encryptor: gcm,
		decryptor: gcm,
	}
	return nil
}

func (encrypt *Interceptor) BindSocketWriter(writer interceptor.Writer) interceptor.Writer {
	return interceptor.WriterFunc(func(conn *websocket.Conn, messageType websocket.MessageType, data []byte) error {
		encrypt.mux.Lock()
		connState, exists := encrypt.collection[conn]
		encrypt.mux.Unlock()

		if !exists || connState.encryptor == nil {
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
		encryptor := connState.encryptor
		encryptedData := encryptor.Seal(nil, nonce, data, nil)

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
		collection, exists := encrypt.collection[conn]
		encrypt.mux.Unlock()

		if !exists || collection.decryptor == nil || len(encryptedData) < 2 {
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
		decryptor := collection.decryptor
		plaintext, err := decryptor.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return messageType, encryptedData, fmt.Errorf("decryption failed: %w", err)
		}

		return messageType, plaintext, nil
	})
}

func (encrypt *Interceptor) UnBindSocketConnection(connection *websocket.Conn) {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	delete(encrypt.collection, connection)
}

func (encrypt *Interceptor) UnBindSocketWriter(_ interceptor.Writer) {}

func (encrypt *Interceptor) UnBindSocketReader(_ interceptor.Reader) {}

func (encrypt *Interceptor) Close() error {
	encrypt.mux.Lock()
	defer encrypt.mux.Unlock()

	encrypt.collection = make(map[*websocket.Conn]*collection)

	return nil
}
