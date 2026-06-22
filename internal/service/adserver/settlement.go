package adserver

import (
	"crypto/aes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	iqiyiBase64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	iqiyiBase64Pad   = '!'
	kPriceCipherSize = 16
)

var iqiyiBase64DecodeMap [256]byte

func init() {
	for i := range iqiyiBase64DecodeMap {
		iqiyiBase64DecodeMap[i] = 0xFF
	}
	for i, c := range []byte(iqiyiBase64Chars) {
		iqiyiBase64DecodeMap[c] = byte(i)
	}
}

// DecodeSettlementPrice decodes and decrypts the ${SETTLEMENT} macro value
// from iQiyi win notice URL. encoded is the base64-encoded Settlement protobuf
// message (iQiyi custom charset). token is 32-char hex string (16 bytes binary
// key). bidID is the Bid.id from BidRequest.
// Returns decrypted price in micros (分).
func DecodeSettlementPrice(encoded, token, bidID string) (int64, error) {
	data, err := iqiyiBase64Decode(encoded)
	if err != nil {
		return 0, fmt.Errorf("base64 decode: %w", err)
	}

	var priceCipher []byte

	for len(data) > 0 {
		tag, n := readVarint(data)
		if n <= 0 {
			break
		}
		data = data[n:]
		fieldNum := int(tag >> 3)
		wireType := int(tag & 0x7)

		if fieldNum == 2 && wireType == 2 {
			length, ln := readVarint(data)
			data = data[ln:]
			priceCipher = make([]byte, length)
			copy(priceCipher, data[:length])
			break
		}

		if wireType == 0 {
			_, ln := readVarint(data)
			data = data[ln:]
		} else if wireType == 2 {
			length, ln := readVarint(data)
			data = data[ln+int(length):]
		}
	}

	if len(priceCipher) != kPriceCipherSize {
		return 0, fmt.Errorf("invalid price cipher length: %d", len(priceCipher))
	}

	key, err := hex.DecodeString(token)
	if err != nil || len(key) != 16 {
		return 0, fmt.Errorf("invalid token: must be 32 hex chars (16 bytes)")
	}

	plaintext, err := aesECBDecrypt(priceCipher, key)
	if err != nil {
		return 0, fmt.Errorf("aes decrypt: %w", err)
	}

	price := int64(binary.BigEndian.Uint64(plaintext))
	return price, nil
}

func aesECBDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("ciphertext not multiple of block size")
	}
	plaintext := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += block.BlockSize() {
		block.Decrypt(plaintext[i:i+block.BlockSize()], ciphertext[i:i+block.BlockSize()])
	}
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > block.BlockSize() {
		return nil, fmt.Errorf("invalid pkcs7 padding")
	}
	return plaintext[:len(plaintext)-padding], nil
}

func iqiyiBase64Decode(s string) ([]byte, error) {
	s = strings.TrimRight(s, string(iqiyiBase64Pad))
	var result []byte
	var buf uint32
	var bits uint8

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c > 255 || iqiyiBase64DecodeMap[c] == 0xFF {
			return nil, fmt.Errorf("invalid base64 char: %c", c)
		}
		buf = (buf << 6) | uint32(iqiyiBase64DecodeMap[c])
		bits += 6
		if bits >= 8 {
			bits -= 8
			result = append(result, byte(buf>>bits))
			buf &= (1 << bits) - 1
		}
	}
	return result, nil
}

func readVarint(data []byte) (uint64, int) {
	var x uint64
	var s uint
	for i, b := range data {
		if i >= 10 {
			return 0, -1
		}
		x |= uint64(b&0x7F) << s
		if b < 0x80 {
			return x, i + 1
		}
		s += 7
	}
	return 0, -1
}
