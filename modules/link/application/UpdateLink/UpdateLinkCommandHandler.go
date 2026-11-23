package application

import (
	"context"
	"strings"

	domainlink "UnpakSiamida/modules/link/domain"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpdateLinkCommandHandler struct {
	Repo domainlink.ILinkRepository
}

func (h *UpdateLinkCommandHandler) Handle(
	ctx context.Context,
	cmd UpdateLinkCommand,
) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// -------------------------
	// VALIDATE UUID
	// -------------------------
	linkUUID, err := uuid.Parse(cmd.Uuid)
	if err != nil {
		return "", domainlink.InvalidUuid()
	}

	// -------------------------
	// GET EXISTING link
	// -------------------------
	countLink := h.Repo.CountLink(ctx, cmd.ShortUrl, cmd.Creator)
	existingLink, err := h.Repo.GetByUuid(ctx, linkUUID) // â† memastikan pakai nama interface yg benar
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", domainlink.NotFound(cmd.Uuid)
		}
		return "", err
	}

	// -------------------------
	// AGGREGATE ROOT LOGIC
	// -------------------------
	result := domainlink.Update(
		existingLink,
		linkUUID,
		cmd.ShortUrl,
		cmd.Creator,
		strings.EqualFold(existingLink.ShortUrl, cmd.ShortUrl) || countLink == 0,
	)

	if !result.IsSuccess {
		return "", result.Error
	}

	updatedLink := result.Value

	// -------------------------
	// SAVE TO REPOSITORY
	// -------------------------
	if err := h.Repo.Update(ctx, updatedLink); err != nil {
		return "", err
	}

	return updatedLink.UUID.String(), nil
}

