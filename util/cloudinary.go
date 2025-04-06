package util

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryService handles image upload operations
type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

// NewCloudinaryService creates a new Cloudinary service instance
func NewCloudinaryService(cloudURL string) (*CloudinaryService, error) {
	cld, err := cloudinary.NewFromURL(cloudURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}
	return &CloudinaryService{cld: cld}, nil
}

// UploadLogo uploads a financial institution logo to Cloudinary
func (s *CloudinaryService) UploadLogo(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader, fiCode string) (string, error) {
	// Create a unique public ID based on the financial institution code
	publicID := fmt.Sprintf("financial_institutions/%s_%d", fiCode, time.Now().Unix())

	// Set upload parameters
	uploadParams := uploader.UploadParams{
		PublicID:     publicID,
		ResourceType: "image",
		Folder:       "financial_institutions",
		Tags:         []string{"financial_institution", fiCode},
	}

	// Upload the file to Cloudinary
	result, err := s.cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		return "", fmt.Errorf("failed to upload logo to cloudinary: %w", err)
	}

	return result.SecureURL, nil
}
