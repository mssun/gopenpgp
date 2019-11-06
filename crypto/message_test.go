package crypto

import (
	"bytes"
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/openpgp/packet"
)

func TestTextMessageEncryptionWithSymmetricKey(t *testing.T) {
	var message = NewPlainMessageFromString("The secret code is... 1, 2, 3, 4, 5")

	// Encrypt data with password
	encrypted, err := testSymmetricKey.Encrypt(message)
	if err != nil {
		t.Fatal("Expected no error when encrypting, got:", err)
	}
	// Decrypt data with wrong password
	_, err = testWrongSymmetricKey.Decrypt(encrypted)
	assert.NotNil(t, err)

	// Decrypt data with the good password
	decrypted, err := testSymmetricKey.Decrypt(encrypted)
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message.GetString(), decrypted.GetString())
}

func TestBinaryMessageEncryptionWithSymmetricKey(t *testing.T) {
	binData, _ := base64.StdEncoding.DecodeString("ExXmnSiQ2QCey20YLH6qlLhkY3xnIBC1AwlIXwK/HvY=")
	var message = NewPlainMessage(binData)

	// Encrypt data with password
	encrypted, err := testSymmetricKey.Encrypt(message)
	if err != nil {
		t.Fatal("Expected no error when encrypting, got:", err)
	}
	// Decrypt data with wrong password
	_, err = testWrongSymmetricKey.Decrypt(encrypted)
	assert.NotNil(t, err)

	// Decrypt data with the good password
	decrypted, err := testSymmetricKey.Decrypt(encrypted)
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message, decrypted)
}

func TestTextMessageEncryption(t *testing.T) {
	var message = NewPlainMessageFromString("plain text")

	testPublicKeyRing, _ = BuildKeyRingArmored(readTestFile("keyring_publicKey", false))
	testPrivateKeyRing, err = BuildKeyRingArmored(readTestFile("keyring_privateKey", false))

	// Password defined in keyring_test
	testPrivateKeyRing, err = testPrivateKeyRing.Unlock(testMailboxPassword)
	if err != nil {
		t.Fatal("Expected no error unlocking privateKey, got:", err)
	}

	ciphertext, err := testPublicKeyRing.Encrypt(message, testPrivateKeyRing)
	if err != nil {
		t.Fatal("Expected no error when encrypting, got:", err)
	}

	decrypted, err := testPrivateKeyRing.Decrypt(ciphertext, testPublicKeyRing, GetUnixTime())
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message.GetString(), decrypted.GetString())
}

func TestBinaryMessageEncryption(t *testing.T) {
	binData, _ := base64.StdEncoding.DecodeString("ExXmnSiQ2QCey20YLH6qlLhkY3xnIBC1AwlIXwK/HvY=")
	var message = NewPlainMessage(binData)

	testPublicKeyRing, _ = BuildKeyRingArmored(readTestFile("keyring_publicKey", false))
	testPrivateKeyRing, err = BuildKeyRingArmored(readTestFile("keyring_privateKey", false))

	// Password defined in keyring_test
	testPrivateKeyRing, err = testPrivateKeyRing.Unlock(testMailboxPassword)
	if err != nil {
		t.Fatal("Expected no error unlocking privateKey, got:", err)
	}

	ciphertext, err := testPublicKeyRing.Encrypt(message, testPrivateKeyRing)
	if err != nil {
		t.Fatal("Expected no error when encrypting, got:", err)
	}

	decrypted, err := testPrivateKeyRing.Decrypt(ciphertext, testPublicKeyRing, GetUnixTime())
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message.GetBinary(), decrypted.GetBinary())

	// Decrypt without verifying
	decrypted, err = testPrivateKeyRing.Decrypt(ciphertext, nil, 0)
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message.GetString(), decrypted.GetString())
}

func TestIssue11(t *testing.T) {
	var issue11Password = [][]byte{ []byte("1234") }
	issue11Keyring, err := BuildKeyRingArmored(readTestFile("issue11_privatekey", false))
	if err != nil {
		t.Fatal("Expected no error while bulding private keyring, got:", err)
	}

	issue11Keyring, err = issue11Keyring.Unlock(issue11Password);
	if err != nil {
		t.Fatal("Expected no error while unlocking private keyring, got:", err)
	}

	senderKeyring, err := BuildKeyRingArmored(readTestFile("issue11_publickey", false))
	if err != nil {
		t.Fatal("Expected no error while building public keyring, got:", err)
	}

	assert.Exactly(t, []uint64{0x643b3595e6ee4fdf}, senderKeyring.KeyIds())

	pgpMessage, err := NewPGPMessageFromArmored(readTestFile("issue11_message", false))
	if err != nil {
		t.Fatal("Expected no error while unlocking private keyring, got:", err)
	}

	plainMessage, err := issue11Keyring.Decrypt(pgpMessage, senderKeyring, 0)
	if err != nil {
		t.Fatal("Expected no error while decrypting/verifying, got:", err)
	}

	assert.Exactly(t, "message from sender", plainMessage.GetString())
}

func TestSignedMessageDecryption(t *testing.T) {
	testPrivateKeyRing, err = BuildKeyRingArmored(readTestFile("keyring_privateKey", false))

	// Password defined in keyring_test
	testPrivateKeyRing, err = testPrivateKeyRing.Unlock(testMailboxPassword)
	if err != nil {
		t.Fatal("Expected no error unlocking privateKey, got:", err)
	}

	pgpMessage, err := NewPGPMessageFromArmored(readTestFile("message_signed", false))
	if err != nil {
		t.Fatal("Expected no error when unarmoring, got:", err)
	}

	decrypted, err := testPrivateKeyRing.Decrypt(pgpMessage, nil, 0)
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, readTestFile("message_plaintext", true), decrypted.GetString())
}

func TestMultipleKeyMessageEncryption(t *testing.T) {
	var message = NewPlainMessageFromString("plain text")

	testPublicKeyRing, _ = BuildKeyRingArmored(readTestFile("keyring_publicKey", false))
	err = testPublicKeyRing.ReadFrom(strings.NewReader(readTestFile("mime_publicKey", false)), true)
	if err != nil {
		t.Fatal("Expected no error adding second public key, got:", err)
	}

	assert.Exactly(t, 2, len(testPublicKeyRing.entities))

	testPrivateKeyRing, err = BuildKeyRingArmored(readTestFile("keyring_privateKey", false))

	// Password defined in keyring_test
	testPrivateKeyRing, err = testPrivateKeyRing.Unlock(testMailboxPassword)
	if err != nil {
		t.Fatal("Expected no error unlocking privateKey, got:", err)
	}

	ciphertext, err := testPublicKeyRing.Encrypt(message, testPrivateKeyRing)
	if err != nil {
		t.Fatal("Expected no error when encrypting, got:", err)
	}

	numKeyPackets := 0
	packets := packet.NewReader(bytes.NewReader(ciphertext.Data))
	for {
		var p packet.Packet
		if p, err = packets.Next(); err == io.EOF {
			err = nil
			break
		}
		switch p.(type) {
			case *packet.EncryptedKey:
				numKeyPackets++
		}
	}
	assert.Exactly(t, 2, numKeyPackets)

	decrypted, err := testPrivateKeyRing.Decrypt(ciphertext, testPublicKeyRing, GetUnixTime())
	if err != nil {
		t.Fatal("Expected no error when decrypting, got:", err)
	}
	assert.Exactly(t, message.GetString(), decrypted.GetString())
}
