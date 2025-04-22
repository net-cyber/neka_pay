package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"regexp"
	"time"
)

// RTCMaker is responsible for creating RTC tokens for Agora service
type RTCMaker struct {
	appID          string
	appCertificate string
}

// RTCRole defines the role of user in channel
type RTCRole uint16

const (
	// RolePublisher can publish streams to channel
	RolePublisher RTCRole = 1
	// RoleSubscriber can only subscribe to streams in channel
	RoleSubscriber RTCRole = 2
)

// RTCPrivilege constants for AccessToken2
const (
	RTCPrivilegeJoinChannel        uint16 = 1
	RTCPrivilegePublishAudioStream uint16 = 2
	RTCPrivilegePublishVideoStream uint16 = 3
	RTCPrivilegePublishDataStream  uint16 = 4
)

// NewRTCMaker creates a new RTCMaker instance
func NewRTCMaker(appID, appCertificate string) (*RTCMaker, error) {
	if !isValidUUID(appID) || !isValidUUID(appCertificate) {
		return nil, fmt.Errorf("invalid appID or appCertificate")
	}

	return &RTCMaker{
		appID:          appID,
		appCertificate: appCertificate,
	}, nil
}

// CreateRTCToken generates an AccessToken2 for RTC service
func (maker *RTCMaker) CreateRTCToken(channelName string, uid uint32, expireDuration time.Duration) (string, error) {
	// Default to RolePublisher
	role := RolePublisher

	// Convert duration to seconds
	expireSeconds := uint32(expireDuration.Seconds())

	// Use BuildTokenWithUid to generate the token
	return maker.BuildTokenWithUid(channelName, uid, role, expireSeconds, expireSeconds)
}

// BuildTokenWithUid generates an AccessToken2 with user ID and role
func (maker *RTCMaker) BuildTokenWithUid(channelName string, uid uint32, role RTCRole, tokenExpire, privilegeExpire uint32) (string, error) {
	// Get current timestamp
	issueTs := uint32(time.Now().Unix())

	// Calculate expiration time
	expireTs := issueTs + tokenExpire

	// Create random salt
	rand.Seed(time.Now().UnixNano())
	salt := rand.Uint32()

	// Create privilege map
	privileges := make(map[uint16]uint32)
	privileges[RTCPrivilegeJoinChannel] = issueTs + privilegeExpire

	// If role is publisher, add publishing privileges
	if role == RolePublisher {
		privileges[RTCPrivilegePublishAudioStream] = issueTs + privilegeExpire
		privileges[RTCPrivilegePublishVideoStream] = issueTs + privilegeExpire
		privileges[RTCPrivilegePublishDataStream] = issueTs + privilegeExpire
	}

	// Create access token
	accessToken := createAccessToken2(maker.appID, maker.appCertificate, channelName,
		fmt.Sprintf("%d", uid), issueTs, expireTs, salt, privileges)

	return accessToken, nil
}

// BuildTokenWithUserAccount generates an AccessToken2 with user account
func (maker *RTCMaker) BuildTokenWithUserAccount(channelName, account string, role RTCRole, tokenExpire, privilegeExpire uint32) (string, error) {
	// Get current timestamp
	issueTs := uint32(time.Now().Unix())

	// Calculate expiration time
	expireTs := issueTs + tokenExpire

	// Create random salt
	rand.Seed(time.Now().UnixNano())
	salt := rand.Uint32()

	// Create privilege map
	privileges := make(map[uint16]uint32)
	privileges[RTCPrivilegeJoinChannel] = issueTs + privilegeExpire

	// If role is publisher, add publishing privileges
	if role == RolePublisher {
		privileges[RTCPrivilegePublishAudioStream] = issueTs + privilegeExpire
		privileges[RTCPrivilegePublishVideoStream] = issueTs + privilegeExpire
		privileges[RTCPrivilegePublishDataStream] = issueTs + privilegeExpire
	}

	// Create access token
	accessToken := createAccessToken2(maker.appID, maker.appCertificate, channelName,
		account, issueTs, expireTs, salt, privileges)

	return accessToken, nil
}

func createAccessToken2(appID, appCertificate, channelName, account string,
	issueTs, expireTs uint32, salt uint32, privileges map[uint16]uint32) string {

	token := AccessToken2{
		AppID:          appID,
		AppCertificate: appCertificate,
		IssueTs:        issueTs,
		ExpireTs:       expireTs,
		Salt:           salt,
		Services:       make(map[uint16]Service),
	}

	// Add RTC service with privileges
	service := Service{
		ServiceType: 1, // RTC service
		Privileges:  privileges,
	}
	token.AddService(service)

	// Set channel name and user account
	token.ChannelName = channelName
	token.UserAccount = account

	// Build and return token
	return token.Build()
}

// AccessToken2 represents the new token format
type AccessToken2 struct {
	AppID          string
	AppCertificate string
	IssueTs        uint32
	ExpireTs       uint32
	Salt           uint32
	Services       map[uint16]Service
	ChannelName    string
	UserAccount    string
}

// Service represents a service in the token
type Service struct {
	ServiceType uint16
	Privileges  map[uint16]uint32
}

// AddService adds a service to the token
func (token *AccessToken2) AddService(service Service) {
	token.Services[service.ServiceType] = service
}

// Build generates the token string
func (token *AccessToken2) Build() string {
	// Generate signature
	signature := token.generateSignature()

	// Create buffer to build the token
	buffer := NewBuffer()

	// Add version (version 2)
	buffer.PackString("002")

	// Add app ID
	buffer.PackString(token.AppID)

	// Add issue timestamp
	buffer.PackUint32(token.IssueTs)

	// Add expiration timestamp
	buffer.PackUint32(token.ExpireTs)

	// Add signature
	buffer.PackString(signature)

	// Add salt
	buffer.PackUint32(token.Salt)

	// Add services count
	buffer.PackUint16(uint16(len(token.Services)))

	// Add each service
	for serviceType, service := range token.Services {
		buffer.PackUint16(serviceType)

		// Add privileges count
		buffer.PackUint16(uint16(len(service.Privileges)))

		// Add each privilege
		for privilegeKey, expireTs := range service.Privileges {
			buffer.PackUint16(privilegeKey)
			buffer.PackUint32(expireTs)
		}
	}

	// Add channel name
	buffer.PackString(token.ChannelName)

	// Add user account
	buffer.PackString(token.UserAccount)

	// Calculate base64 encoded token
	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

// generateSignature creates a signature for the token
func (token *AccessToken2) generateSignature() string {
	// Create buffer for signature
	buffer := NewBuffer()

	// Pack app ID
	buffer.PackString(token.AppID)

	// Pack channel name
	buffer.PackString(token.ChannelName)

	// Pack user account
	buffer.PackString(token.UserAccount)

	// Pack issue timestamp
	buffer.PackUint32(token.IssueTs)

	// Pack expiration timestamp
	buffer.PackUint32(token.ExpireTs)

	// Pack salt
	buffer.PackUint32(token.Salt)

	// Pack services count
	buffer.PackUint16(uint16(len(token.Services)))

	// Calculate HMAC-SHA256
	h := hmac.New(sha256.New, []byte(token.AppCertificate))
	h.Write(buffer.Bytes())

	// Return hex-encoded signature
	return hex.EncodeToString(h.Sum(nil))
}

// Buffer is a helper for building binary data
type Buffer struct {
	buffer []byte
}

// NewBuffer creates a new Buffer
func NewBuffer() *Buffer {
	return &Buffer{
		buffer: make([]byte, 0),
	}
}

// PackUint16 adds a uint16 to the buffer
func (b *Buffer) PackUint16(val uint16) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, val)
	b.buffer = append(b.buffer, buf...)
}

// PackUint32 adds a uint32 to the buffer
func (b *Buffer) PackUint32(val uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, val)
	b.buffer = append(b.buffer, buf...)
}

// PackString adds a string to the buffer
func (b *Buffer) PackString(val string) {
	// Add string length
	b.PackUint16(uint16(len(val)))

	// Add string content
	b.buffer = append(b.buffer, []byte(val)...)
}

// Bytes returns the buffer content
func (b *Buffer) Bytes() []byte {
	return b.buffer
}

// Helper functions

func (maker *RTCMaker) generateSignature(message string) string {
	h := hmac.New(sha256.New, []byte(maker.appCertificate))
	io.WriteString(h, message)
	return hex.EncodeToString(h.Sum(nil))
}

func isValidUUID(uuid string) bool {
	match, _ := regexp.MatchString("^[0-9a-fA-F]{32}$", uuid)
	return match
}
