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
	salt := rand.Uint32()%99999999 + 1

	// Create signature
	signature := maker.generateSignature(issueTime, salt)

	// Pack the data
	var buffer bytes.Buffer

	// Pack appID
	packString(&buffer, maker.appID)

	// Pack issueTime, expireTime, salt
	binary.Write(&buffer, binary.LittleEndian, issueTime)
	binary.Write(&buffer, binary.LittleEndian, expireTime)
	binary.Write(&buffer, binary.LittleEndian, salt)

	// Pack service type count (1 for RTC only)
	binary.Write(&buffer, binary.LittleEndian, uint16(1))

	// Pack service type (1 for RTC)
	binary.Write(&buffer, binary.LittleEndian, uint16(1))

	// Pack privileges map size (1 for join channel privilege)
	binary.Write(&buffer, binary.LittleEndian, uint16(1))

	// Pack join channel privilege and expiration
	binary.Write(&buffer, binary.LittleEndian, uint16(RTCPrivilegeJoinChannel))
	binary.Write(&buffer, binary.LittleEndian, expireTime)

	// Pack channel name
	packString(&buffer, channelName)

	// Pack UID as string
	uidStr := fmt.Sprintf("%d", uid)
	packString(&buffer, uidStr)

	// Combine signature with data
	var finalBuffer bytes.Buffer
	packString(&finalBuffer, string(signature))
	finalBuffer.Write(buffer.Bytes())

	// Compress with zlib
	var zlibBuffer bytes.Buffer
	zw := zlib.NewWriter(&zlibBuffer)
	zw.Write(finalBuffer.Bytes())
	zw.Close()

	// Base64 encode
	b64 := base64.StdEncoding.EncodeToString(zlibBuffer.Bytes())

	// Add version prefix
	return maker.version + b64, nil
}

// Helper functions

func (maker *RTCMaker) generateSignature(issueTime, salt uint32) []byte {
	// First HMAC with app certificate
	h := hmac.New(sha256.New, []byte(maker.appCertificate))
	var buffer bytes.Buffer
	binary.Write(&buffer, binary.LittleEndian, issueTime)
	h.Write(buffer.Bytes())
	hh := h.Sum(nil)

	// Second HMAC with salt
	h = hmac.New(sha256.New, hh)
	buffer.Reset()
	binary.Write(&buffer, binary.LittleEndian, salt)
	h.Write(buffer.Bytes())

	return h.Sum(nil)
}

func packString(buffer *bytes.Buffer, s string) {
	binary.Write(buffer, binary.LittleEndian, uint16(len(s)))
	buffer.WriteString(s)
}

func isValidUUID(uuid string) bool {
	match, _ := regexp.MatchString("^[0-9a-fA-F]{32}$", uuid)
	return match
}
