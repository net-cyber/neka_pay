package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/net-cyber/neka_pay/db/sqlc"
)

type financialInstitutionResponse struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	LogoURL string `json:"logo_url"`
	Code    string `json:"code"`
	Active  bool   `json:"active"`
}

type createFinancialInstitutionRequest struct {
	Name string `form:"name" binding:"required"`
	Type string `form:"type" binding:"required,oneof=bank wallet mfi"`
	Code string `form:"code" binding:"required"`
}

// createFinancialInstitution handles creation of a new financial institution
func (server *Server) createFinancialInstitution(ctx *gin.Context) {
	var req createFinancialInstitutionRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Get the logo file
	file, header, err := ctx.Request.FormFile("logo")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("logo file is required")))
		return
	}
	defer file.Close()

	// Upload logo to Cloudinary
	logoURL, err := server.cloudinary.UploadLogo(ctx, file, header, req.Code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Create financial institution in database
	arg := db.CreateFinancialInstitutionParams{
		Name:    req.Name,
		Type:    db.FinancialInstitutionType(req.Type),
		LogoUrl: logoURL,
		Code:    req.Code,
		Active:  true,
	}

	fi, err := server.store.CreateFinancialInstitution(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusCreated, financialInstitutionResponse{
		ID:      fi.ID,
		Name:    fi.Name,
		Type:    string(fi.Type),
		LogoURL: fi.LogoUrl,
		Code:    fi.Code,
		Active:  fi.Active,
	})
}

type getFinancialInstitutionRequest struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) getFinancialInstitution(ctx *gin.Context) {
	var req getFinancialInstitutionRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	fi, err := server.store.GetFinancialInstitution(ctx, req.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, financialInstitutionResponse{
		ID:      fi.ID,
		Name:    fi.Name,
		Type:    string(fi.Type),
		LogoURL: fi.LogoUrl,
		Code:    fi.Code,
		Active:  fi.Active,
	})
}

type listFinancialInstitutionsRequest struct {
	PageID   int32  `form:"page_id" binding:"required,min=1"`
	PageSize int32  `form:"page_size" binding:"required,min=5,max=100"`
}

func (server *Server) listFinancialInstitutions(ctx *gin.Context) {
	var req listFinancialInstitutionsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}




	arg := db.ListFinancialInstitutionsParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	fis, err := server.store.ListFinancialInstitutions(ctx, arg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	response := make([]financialInstitutionResponse, len(fis))
	for i, fi := range fis {
		response[i] = financialInstitutionResponse{
			ID:      fi.ID,
			Name:    fi.Name,
			Type:    string(fi.Type),
			LogoURL: fi.LogoUrl,
			Code:    fi.Code,
			Active:  fi.Active,
		}
	}

	ctx.JSON(http.StatusOK, response)
}

type updateFinancialInstitutionRequest struct {
	Name   string `form:"name"`
	Active *bool  `form:"active"`
}

func (server *Server) updateFinancialInstitution(ctx *gin.Context) {
	var uri getFinancialInstitutionRequest
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateFinancialInstitutionRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Initialize outside conditional blocks
	arg := db.UpdateFinancialInstitutionParams{
		ID: uri.ID,
	}

	// Convert string to sql.NullString
	if req.Name != "" {
		arg.Name = sql.NullString{
			String: req.Name,
			Valid:  true,
		}
	}

	// Convert *bool to sql.NullBool
	if req.Active != nil {
		arg.Active = sql.NullBool{
			Bool:  *req.Active,
			Valid: true,
		}
	}

	// Check if logo is being updated
	file, header, err := ctx.Request.FormFile("logo")
	if err == nil {
		defer file.Close()

		// Get the financial institution to get the code
		fi, err := server.store.GetFinancialInstitution(ctx, uri.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		// Upload new logo to Cloudinary
		logoURL, err := server.cloudinary.UploadLogo(ctx, file, header, fi.Code)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		// Convert string to sql.NullString for LogoUrl
		arg.LogoUrl = sql.NullString{
			String: logoURL,
			Valid:  true,
		}
	}

	fi, err := server.store.UpdateFinancialInstitution(ctx, arg)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, financialInstitutionResponse{
		ID:      fi.ID,
		Name:    fi.Name,
		Type:    string(fi.Type),
		LogoURL: fi.LogoUrl,
		Code:    fi.Code,
		Active:  fi.Active,
	})
}
