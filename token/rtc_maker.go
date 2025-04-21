package token

import (
	"bytes"
	"compress/zlib"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// RTCMaker is responsible for creating RTC tokens for Agora service
type RTCMaker struct {
	appID          string
	appCertificate string
	version        string
}

// RTCPrivilege constants
const (
	RTCPrivilegeJoinChannel        = 1
	RTCPrivilegePublishAudioStream = 2
	RTCPrivilegePublishVideoStream = 3
	RTCPrivilegePublishDataStream  = 4
)

// NewRTCMaker creates a new RTCMaker instance
func NewRTCMaker(appID, appCertificate string) (*RTCMaker, error) {
	if !isValidUUID(appID) || !isValidUUID(appCertificate) {
		return nil, fmt.Errorf("invalid appID or appCertificate")
	}

	return &RTCMaker{
		appID:          appID,
		appCertificate: appCertificate,
		version:        "007", // Using the same version as in PHP implementation
	}, nil
}

// CreateRTCToken generates a token for RTC service
func (maker *RTCMaker) CreateRTCToken(channelName string, uid uint32, expireDuration time.Duration) (string, error) {
	// Current timestamp
	issueTime := uint32(time.Now().Unix())
	expireTime := uint32(time.Now().Add(expireDuration).Unix())

	// Generate a random salt between 1 and 99999999
	rand.Seed(time.Now().UnixNano())
	salt := rand.Intn(99999999) + 1 // Use Intn to match PHP's rand function

	// Pack the data into a buffer
	var buffer bytes.Buffer

	// First pack the signature
	signature := maker.generateSignature(issueTime, uint32(salt))
	packString(&buffer, string(signature))

	// Pack appID
	packString(&buffer, maker.appID)

	// Pack issueTime
	packUint32(&buffer, issueTime)

	// Pack expireTime
	packUint32(&buffer, expireTime)

	// Pack salt
	packUint32(&buffer, uint32(salt))

	// Pack service type count (1 for RTC only)
	packUint16(&buffer, 1)

	// Pack service type (1 for RTC)
	packUint16(&buffer, 1)

	// Pack privileges map
	packMapUint32(&buffer, map[uint16]uint32{
		RTCPrivilegeJoinChannel: expireTime,
	})

	// Pack channel name
	packString(&buffer, channelName)

	// Pack UID as string
	uidStr := fmt.Sprintf("%d", uid)
	packString(&buffer, uidStr)

	// Compress with zlib
	var zlibBuffer bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&zlibBuffer, zlib.DefaultCompression)
	_, err := zw.Write(buffer.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to compress token: %w", err)
	}
	zw.Close()

	// Base64 encode
	b64 := base64.StdEncoding.EncodeToString(zlibBuffer.Bytes())

	// Add version prefix
	return maker.version + b64, nil
}

// Helper functions

func (maker *RTCMaker) generateSignature(issueTime, salt uint32) []byte {
	// First HMAC with app certificate and issue time
	var timeBuffer bytes.Buffer
	packUint32(&timeBuffer, issueTime)

	h := hmac.New(sha256.New, []byte(maker.appCertificate))
	h.Write(timeBuffer.Bytes())
	hmacResult := h.Sum(nil)

	// Second HMAC with salt
	var saltBuffer bytes.Buffer
	packUint32(&saltBuffer, salt)

	h = hmac.New(sha256.New, hmacResult)
	h.Write(saltBuffer.Bytes())

	return h.Sum(nil)
}

func packUint16(buffer *bytes.Buffer, val uint16) {
	b := make([]byte, 2)
	// Use little-endian to match PHP's pack("v", $x)
	binary.LittleEndian.PutUint16(b, val)
	buffer.Write(b)
}

func packUint32(buffer *bytes.Buffer, val uint32) {
	b := make([]byte, 4)
	// Use little-endian to match PHP's pack("V", $x)
	binary.LittleEndian.PutUint32(b, val)
	buffer.Write(b)
}

func packString(buffer *bytes.Buffer, s string) {
	// Pack string length as uint16
	packUint16(buffer, uint16(len(s)))
	// Pack string content
	buffer.WriteString(s)
}

func packMapUint32(buffer *bytes.Buffer, m map[uint16]uint32) {
	// Pack map size
	packUint16(buffer, uint16(len(m)))

	// Sort keys for consistent output
	// Note: In a real implementation, you'd want to sort the keys here
	for k, v := range m {
		packUint16(buffer, k)
		packUint32(buffer, v)
	}
}

func isValidUUID(uuid string) bool {
	match, _ := regexp.MatchString("^[0-9a-fA-F]{32}$", uuid)
	return match
}
