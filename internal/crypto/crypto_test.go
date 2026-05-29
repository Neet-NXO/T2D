package crypto

import (
	"bytes"
	"testing"
)

func TestCipherRoundTrip(t *testing.T) {
	tests := []Config{
		{Type: CipherNone},
		{Type: CipherXOR},
		{Type: CipherAES128GCM, Password: "test-password"},
		{Type: CipherAES256GCM, Password: "test-password"},
		{Type: CipherChaCha20Poly1305, Password: "test-password"},
		{Type: CipherXChaCha20Poly1305, Password: "test-password"},
	}
	plain := bytes.Repeat([]byte("a"), 1400)
	for _, tc := range tests {
		cipher, err := NewCipher(&tc)
		if err != nil {
			t.Fatalf("%s: new cipher: %v", tc.Type, err)
		}
		encrypted, err := cipher.Encrypt(plain)
		if err != nil {
			t.Fatalf("%s: encrypt: %v", tc.Type, err)
		}
		decrypted, err := cipher.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("%s: decrypt: %v", tc.Type, err)
		}
		if !bytes.Equal(decrypted, plain) {
			t.Fatalf("%s: round trip mismatch", tc.Type)
		}
	}
}

func BenchmarkCipherEncrypt(b *testing.B) {
	payload := bytes.Repeat([]byte{0x42}, 1420)
	tests := []Config{
		{Type: CipherNone},
		{Type: CipherAES256GCM, Password: "test-password"},
		{Type: CipherChaCha20Poly1305, Password: "test-password"},
	}
	for _, tc := range tests {
		b.Run(string(tc.Type), func(b *testing.B) {
			cipher, err := NewCipher(&tc)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(payload)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := cipher.Encrypt(payload); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
