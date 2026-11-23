package infrastructure

import (
	commondomain "UnpakSiamida/common/domain"
	domainClick "UnpakSiamida/modules/click/domain"
	domainLink "UnpakSiamida/modules/link/domain"
	"context"
	"strings"

	"gorm.io/hints"

	"gorm.io/gorm"
)

type ClickRepository struct {
	db *gorm.DB
}

func NewClickRepository(db *gorm.DB) domainClick.IClickRepository {
	return &ClickRepository{db: db}
}

func (r *ClickRepository) Create(
	ctx context.Context,
	click *domainClick.Click,
) error {

	err := r.db.WithContext(ctx).
		Create(click).Error

	if err != nil {
		return err
	}

	return nil
}

var allowedSearchColumns = map[string]string{
	// key:param -> db column
	"short_url": "short_url",
	"long_url":  "long_url",
	"creator":   "creator",
}

var allowedClickColumns = map[string]string{
	"created_at": "created_at",
}

// ------------------------
// GET ALL
// ------------------------
func (r *ClickRepository) GetAll(
	ctx context.Context,
	search string,
	searchFilters []commondomain.SearchFilter,
	page, limit *int,
) ([]domainClick.ClickDefault, int64, error) {

	var clicks []domainClick.Click
	var rows []domainClick.ClickDefault
	var total int64

	base := r.db.WithContext(ctx).
		Model(&domainClick.Click{}).
		Clauses(hints.ForceIndex("PRIMARY"))

	// -------------------------------------------------
	// FILTER (khusus kolom link â†’ pakai subquery, kolom click â†’ langsung)
	// -------------------------------------------------
	hasLinkFilter := false
	linkSub := r.db.Model(&domainLink.Link{}).Select("id")

	for _, f := range searchFilters {
		fieldLower := strings.ToLower(f.Field)

		// Cek apakah kolom click
		if clickCol, ok := allowedClickColumns[fieldLower]; ok {
			val := ""
			if f.Value != nil {
				val = strings.TrimSpace(*f.Value)
			}
			switch strings.ToLower(f.Operator) {
			case "eq":
				base = base.Where(clickCol+" = ?", val)
			case "neq":
				base = base.Where(clickCol+" <> ?", val)
			case "gte":
				base = base.Where(clickCol+" >= ?", val)
			case "lte":
				base = base.Where(clickCol+" <= ?", val)
			case "like":
				base = base.Where(clickCol+" LIKE ?", "%"+val+"%")
			case "in":
				base = base.Where(clickCol+" IN ?", strings.Split(val, ","))
			}
			continue
		}

		// Cek apakah kolom link
		if linkCol, ok := allowedSearchColumns[fieldLower]; ok {
			hasLinkFilter = true
			val := ""
			if f.Value != nil {
				val = strings.TrimSpace(*f.Value)
			}
			switch strings.ToLower(f.Operator) {
			case "eq":
				linkSub = linkSub.Where(linkCol+" = ?", val)
			case "neq":
				linkSub = linkSub.Where(linkCol+" <> ?", val)
			case "like":
				linkSub = linkSub.Where(linkCol+" LIKE ?", "%"+val+"%")
			case "in":
				linkSub = linkSub.Where(linkCol+" IN ?", strings.Split(val, ","))
			}
		}
	}

	if hasLinkFilter || strings.TrimSpace(search) != "" {
		// global search
		if strings.TrimSpace(search) != "" {
			like := "%" + search + "%"
			var or []string
			var args []interface{}

			for _, col := range allowedSearchColumns {
				or = append(or, col+" LIKE ?")
				args = append(args, like)
			}

			linkSub = linkSub.Where("("+strings.Join(or, " OR ")+")", args...)
		}

		base = base.Where("link_id IN (?)", linkSub)
	}

	// -------------------------------------------------
	// COUNT (TANPA JOIN)
	// -------------------------------------------------
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// -------------------------------------------------
	// PAGINATION (di clicks saja)
	// -------------------------------------------------
	base = base.Order("id DESC")

	if page != nil && limit != nil && *limit > 0 {
		offset := (*page - 1) * (*limit)
		base = base.Offset(offset).Limit(*limit)
	}

	if err := base.Find(&clicks).Error; err != nil {
		return nil, 0, err
	}

	if len(clicks) == 0 {
		return []domainClick.ClickDefault{}, total, nil
	}

	// -------------------------------------------------
	// AMBIL LINKS SEKALI SAJA
	// -------------------------------------------------
	linkIDs := make([]uint, 0)
	seen := make(map[uint]struct{})

	for _, c := range clicks {
		if _, ok := seen[c.LinkId]; !ok {
			seen[c.LinkId] = struct{}{}
			linkIDs = append(linkIDs, c.LinkId)
		}
	}

	var links []domainLink.Link
	if err := r.db.WithContext(ctx).
		Model(&domainLink.Link{}).
		Clauses(hints.ForceIndex("PRIMARY")).
		Where("id IN ?", linkIDs).
		Find(&links).Error; err != nil {
		return nil, 0, err
	}

	linkMap := make(map[uint]domainLink.Link)
	for _, l := range links {
		linkMap[l.ID] = l
	}

	// -------------------------------------------------
	// MERGE MEMORY (SUPER CEPAT)
	// -------------------------------------------------
	rows = make([]domainClick.ClickDefault, 0, len(clicks))

	for _, c := range clicks {
		l := linkMap[c.LinkId]

		rows = append(rows, domainClick.ClickDefault{
			LinkId:    l.ID,
			LinkShort: l.ShortUrl,
			LinkLong:  l.LongUrl,
			IP:        c.IP,
			IpClient:  c.IpClient,
			ISO:       c.ISO,
			Country:   c.Country,
			UserAgent: c.UserAgent,
			Created:   c.Created,
		})
	}

	return rows, total, nil
}

