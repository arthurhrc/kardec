package pdf

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"encoding/binary"
	"fmt"
)

// Standard Security Handler implementation per PDF 1.7 §7.6. Kardec
// emits V=4 / R=4 / AES-128 (the "Public-Key" handler is out of
// scope; we ship the password-and-permissions form most users
// expect).
//
// The algorithms mirror the spec algorithms 3.2 (compute encryption
// key), 3.3 (compute O entry), 3.4-3.5 (compute U entry), and
// algorithm 1 / 1.A (encrypt strings and streams with per-object
// keys). Comments cite the spec-numbered algorithm where helpful.

// pdfPasswordPad is the 32-byte constant the spec mandates for
// padding short or empty passwords. Spec §7.6.3.3 algorithm 2 step 1.
var pdfPasswordPad = []byte{
	0x28, 0xBF, 0x4E, 0x5E, 0x4E, 0x75, 0x8A, 0x41,
	0x64, 0x00, 0x4E, 0x56, 0xFF, 0xFA, 0x01, 0x08,
	0x2E, 0x2E, 0x00, 0xB6, 0xD0, 0x68, 0x3E, 0x80,
	0x2F, 0x0C, 0xA9, 0xFE, 0x64, 0x53, 0x69, 0x7A,
}

// padPassword pads or truncates s to exactly 32 bytes using the
// spec constant. Used by O, U, and key-derivation algorithms.
func padPassword(s string) []byte {
	out := make([]byte, 32)
	n := copy(out, s)
	copy(out[n:], pdfPasswordPad[:32-n])
	return out
}

// computeOwnerHash implements PDF spec algorithm 3.3: derive the /O
// entry from the owner and user passwords. Returns 32 bytes.
//
// For V=4 R=4 (the only revision Kardec emits), the algorithm runs
// MD5 50 times on the owner key derivation, RC4-encrypts the padded
// user password, then XOR-iterates RC4 19 more times.
func computeOwnerHash(userPwd, ownerPwd string) []byte {
	if ownerPwd == "" {
		ownerPwd = userPwd
	}
	// Step 2: derive the RC4 key from the padded owner password.
	hash := md5.Sum(padPassword(ownerPwd))
	for i := 0; i < 50; i++ {
		hash = md5.Sum(hash[:])
	}
	rc4Key := hash[:16] // 128-bit key

	// Step 5-7: RC4-encrypt the padded user password, then 19 more
	// rounds XOR-ing key bytes with the iteration index.
	out := padPassword(userPwd)
	for i := 0; i <= 19; i++ {
		key := make([]byte, len(rc4Key))
		for j, b := range rc4Key {
			key[j] = b ^ byte(i)
		}
		c, _ := rc4.NewCipher(key)
		c.XORKeyStream(out, out)
	}
	return out
}

// computeEncryptionKey implements PDF spec algorithm 3.2: derive
// the file encryption key (16 bytes for AES-128) from the user
// password, /O entry, /P permissions, and first /ID component.
//
// For V=4 R=4 the algorithm appends 0xFFFFFFFF when metadata is
// not encrypted; Kardec encrypts metadata, so the trailing 4 bytes
// are omitted (R=4 spec note 7.6.3.3 algorithm 2 step 6).
func computeEncryptionKey(userPwd string, oHash []byte, perms int32, idA []byte) []byte {
	var input []byte
	input = append(input, padPassword(userPwd)...)
	input = append(input, oHash...)
	pBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(pBytes, uint32(perms))
	input = append(input, pBytes...)
	input = append(input, idA...)
	hash := md5.Sum(input)
	for i := 0; i < 50; i++ {
		hash = md5.Sum(hash[:16])
	}
	return append([]byte(nil), hash[:16]...)
}

// computeUserHash implements PDF spec algorithm 3.5 (R >= 3): the
// /U entry the reader checks the supplied user password against.
// Returns 32 bytes (16 bytes of derived hash + 16 bytes padding).
func computeUserHash(fileKey, idA []byte) []byte {
	// Step 2-3: MD5 of password padding constant + file ID.
	h := md5.New()
	h.Write(pdfPasswordPad)
	h.Write(idA)
	digest := h.Sum(nil) // 16 bytes

	// Step 4-5: RC4-encrypt with file key, then 19 more rounds.
	out := append([]byte(nil), digest...)
	for i := 0; i <= 19; i++ {
		key := make([]byte, len(fileKey))
		for j, b := range fileKey {
			key[j] = b ^ byte(i)
		}
		c, _ := rc4.NewCipher(key)
		c.XORKeyStream(out, out)
	}
	// Step 6: pad to 32 bytes.
	out = append(out, pdfPasswordPad[:16]...)
	return out
}

// objectKey derives the per-object encryption key per spec
// §7.6.3.4 algorithm 3.1.A (extended for AES). The "sAlT" suffix
// is the AES-specific marker; RC4 omits it.
func objectKey(fileKey []byte, objNum, gen int, aes bool) []byte {
	buf := make([]byte, 0, len(fileKey)+5+4)
	buf = append(buf, fileKey...)
	// Object number (low 24 bits, little-endian).
	buf = append(buf,
		byte(objNum&0xFF),
		byte((objNum>>8)&0xFF),
		byte((objNum>>16)&0xFF),
	)
	// Generation (low 16 bits, little-endian).
	buf = append(buf,
		byte(gen&0xFF),
		byte((gen>>8)&0xFF),
	)
	if aes {
		buf = append(buf, 's', 'A', 'l', 'T')
	}
	hash := md5.Sum(buf)
	keyLen := len(fileKey) + 5
	if keyLen > 16 {
		keyLen = 16
	}
	return append([]byte(nil), hash[:keyLen]...)
}

// decodeHexID converts a hex-encoded /ID component back to bytes.
// stableDocumentID returns 32 hex characters (16 bytes) per
// component; the encryption key derivation needs raw bytes.
func decodeHexID(hexed string) []byte {
	out := make([]byte, len(hexed)/2)
	for i := 0; i < len(out); i++ {
		hi := hexNibbleToByte(hexed[i*2])
		lo := hexNibbleToByte(hexed[i*2+1])
		out[i] = hi<<4 | lo
	}
	return out
}

func hexNibbleToByte(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

// buildEncryptDict assembles the body of the /Encrypt indirect
// object for V=4 / R=4 / AES-128. Both streams and strings flow
// through StdCF (AESV2): /Info Title/Author/Subject/Keywords,
// link-annotation /URIs, and any other literal-string field that
// the writer encrypts via encryptString are rendered as
// hex-encoded ciphertext, opaque to anyone without the password.
//
// (v0.16 shipped with /StrF /Identity which left strings plaintext;
// v0.22 closes that gap so strict regulators that require Title
// confidentiality see a fully encrypted document.)
func buildEncryptDict(oHash, uHash []byte, perms int32) string {
	return fmt.Sprintf(
		"<< /Filter /Standard /V 4 /R 4 /Length 128 /P %d "+
			"/O <%s> /U <%s> "+
			"/CF << /StdCF << /CFM /AESV2 /Length 16 /AuthEvent /DocOpen >> >> "+
			"/StmF /StdCF /StrF /StdCF >>",
		perms,
		hexEncodeBytes(oHash), hexEncodeBytes(uHash),
	)
}

// encryptString returns the hex-encoded ciphertext form of s
// suitable for embedding in a PDF dictionary that lives in the
// indirect object identified by objNum. The output already
// includes the surrounding `<…>` delimiters so callers can drop
// it in where they would have written `(plain)`.
//
// Callers must NOT escape s before passing it in — encryption
// works on the raw bytes. The output's hex digits are themselves
// always safe in a PDF dict and need no escaping.
func encryptString(fileKey []byte, objNum int, s string) string {
	if len(fileKey) == 0 {
		return escapeLiteralString(s)
	}
	cipher := aesEncryptObject(fileKey, objNum, 0, []byte(s))
	return "<" + hexEncodeBytes(cipher) + ">"
}

func hexEncodeBytes(b []byte) string {
	const hexChars = "0123456789ABCDEF"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hexChars[v>>4]
		out[i*2+1] = hexChars[v&0x0F]
	}
	return string(out)
}

// aesEncryptObject encrypts plaintext for a single PDF indirect
// object with AES-128-CBC. The result is a 16-byte random IV
// prepended to the ciphertext, with PKCS#7 padding on the
// plaintext to a 16-byte boundary. Spec §7.6.3.4 algorithm 1.A.
func aesEncryptObject(fileKey []byte, objNum, gen int, plaintext []byte) []byte {
	key := objectKey(fileKey, objNum, gen, true)
	block, err := aes.NewCipher(key)
	if err != nil {
		// 16-byte AES key — NewCipher only fails on invalid sizes,
		// which we control. A panic here would be a Kardec bug.
		panic("pdf: AES init: " + err.Error())
	}

	// PKCS#7 pad to 16-byte boundary.
	padLen := 16 - (len(plaintext) % 16)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	// Random IV.
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		// crypto/rand failures are fatal everywhere; surfacing as
		// panic is the right move — encrypted output without a
		// secure IV would silently weaken the cipher.
		panic("pdf: rand.Read: " + err.Error())
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	out := make([]byte, 16+len(padded))
	copy(out, iv)
	mode.CryptBlocks(out[16:], padded)
	return out
}
