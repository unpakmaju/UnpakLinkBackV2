package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"UnpakSiamida/common/helper"
	commoninfra "UnpakSiamida/common/infrastructure"
	domainLink "UnpakSiamida/modules/link/domain"
	eventLink "UnpakSiamida/modules/link/event"

	"gorm.io/gorm"
)

type GetLinkDefaultByShortQueryHandler struct {
	Repo domainLink.ILinkRepository
}

func (h *GetLinkDefaultByShortQueryHandler) Handle(
	ctx context.Context,
	q GetLinkDefaultByShortQuery,
) (*domainLink.LinkDefault, error) {

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	link, err := h.Repo.GetDefaultByShort(ctx, q.Short)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainLink.NotFound(q.Short)
		}
		return nil, err
	}

	now := time.Now()
	isWaiting := false
	isExpired := false
	if link.StartAccess != nil && link.EndAccess != nil {
		if now.Before(*link.StartAccess) {
			isWaiting = true
		} else if now.After(*link.EndAccess) {
			isExpired = true
		}
	}

	hasPassword := link.Password != nil && *link.Password != ""

	// Jika waktu akses belum dimulai atau sudah expired, sembunyikan LongUrl & Password asli
	if isWaiting || isExpired {
		link.LongUrl = ""
		if hasPassword {
			link.Password = helper.StrPtr("PROTECTED")
		}
		return link, nil
	}

	// Jika link memiliki proteksi password
	if hasPassword {
		if q.Code == "" {
			// Request awal: jangan hitung klik, sembunyikan LongUrl dan mask Password
			link.LongUrl = ""
			link.Password = helper.StrPtr("PROTECTED")
			return link, nil
		}
		if q.Code != *link.Password {
			return nil, commoninfra.NewResponseError("Link.InvalidPassword", "Password tidak sesuai")
		}
		// Password cocok: hapus password dari response sebelum dikirim
		link.Password = nil
	}

	if q.DoCounter {
		err := h.Repo.WithTx(ctx, func(txRepo domainLink.ILinkRepositoryTx) error {

			event := eventLink.NewLinkCountEvent(
				link.Id,
				helper.NullableString(q.IP),
				helper.NullableString(q.IpClient),
				helper.NullableString(q.ISO),
				helper.NullableString(q.Country),
				helper.NullableString(q.Referer),
				helper.NullableString(q.UserAgent),
			)

			payload, err := json.Marshal(event)
			if err != nil {
				return err
			}

			outbox := commoninfra.OutboxMessage{
				ID:            event.ID(),
				Type:          commoninfra.CanonicalTypeName(event),
				Payload:       string(payload),
				OccurredOnUTC: event.OccurredOnUTC(),
			}

			if err := txRepo.InsertOutbox(ctx, &outbox); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return link, nil
}
