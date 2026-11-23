package infrastructure

import (
	commondomainLink "UnpakSiamida/common/domain"
	commoninfra "UnpakSiamida/common/infrastructure"
	domainLink "UnpakSiamida/modules/link/domain"
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LinkRepository struct {
	db  *gorm.DB
	uow *commoninfra.UnitOfWork
}

func NewLinkRepository(db *gorm.DB) domainLink.ILinkRepository {
	return &LinkRepository{db: db, uow: commoninfra.NewUnitOfWork(db)}
}

func (r *LinkRepository) CountLink(ctx context.Context, link string, creator string) uint {
	var Link domainLink.Link
	var count int64

	err := r.db.WithContext(ctx).
		Model(Link).
		Where("creator = ?", creator).
		Where("short_url = ?", link).
		Count(&count).Error

	if err != nil {
		return 0
	}

	return uint(count)
}

// ------------------------
// GET BY UUID
// ------------------------
func (r *LinkRepository) GetByUuid(ctx context.Context, uid uuid.UUID) (*domainLink.Link, error) {
	var Link domainLink.Link

	err := r.db.WithContext(ctx).
		Debug().
		Where("uuid = ?", uid).
		First(&Link).Error

	// if errors.Is(err, gorm.ErrRecordNotFound) {
	// 	return nil, nil
	// }

	if err != nil {
		return nil, err
	}

	return &Link, nil
}

// ------------------------
// GET DEFAULT BY UUID
// ------------------------
func (r *LinkRepository) GetDefaultByUuid(
	ctx context.Context,
	id uuid.UUID,
) (*domainLink.LinkDefault, error) {

	var rowData domainLink.LinkDefault

	err := r.db.WithContext(ctx).
		Debug().
		Table("links").
		Select("id, uuid, nama").
		Where("uuid = ?", id).
		Take(&rowData).Error // Take otomatis LIMIT 1

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	return &rowData, nil
}

// ------------------------
// GET DEFAULT BY Short
// ------------------------
func (r *LinkRepository) GetDefaultByShort(
	ctx context.Context,
	short string,
) (*domainLink.LinkDefault, error) {

	var rowData domainLink.LinkDefault

	err := r.db.WithContext(ctx).
		Debug().
		Table("links").
		Where("short_url = ?", short).
		Take(&rowData).Error // Take otomatis LIMIT 1

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	return &rowData, nil
}

var allowedSearchColumns = map[string]string{
	// key:param -> db column
	"short_url": "short_url",
	"long_url":  "long_url",
	"status":    "status",
	"creator":   "creator",
}

// ------------------------
// GET ALL
// ------------------------
func (r *LinkRepository) GetAll(
	ctx context.Context,
	search string,
	searchFilters []commondomainLink.SearchFilter,
	page, limit *int,
) ([]domainLink.Link, int64, error) {

	var Links = make([]domainLink.Link, 0)
	var total int64

	db := r.db.WithContext(ctx).Model(&domainLink.Link{})

	// -------------------------------
	// SEARCH FILTERS (ADVANCED)
	// -------------------------------
	if len(searchFilters) > 0 {
		for _, f := range searchFilters {
			field := strings.TrimSpace(strings.ToLower(f.Field))
			operator := strings.TrimSpace(strings.ToLower(f.Operator))

			var value string
			if f.Value != nil {
				value = strings.TrimSpace(*f.Value)
			} else {
				value = "" // nil dianggap kosong
			}

			// if value == "" {
			// 	continue
			// }

			// Validate allowed column
			col, ok := allowedSearchColumns[field]
			if !ok {
				continue // skip unknown field
			}

			switch operator {
			case "eq":
				db = db.Where(fmt.Sprintf("%s = ?", col), value)
			case "neq":
				db = db.Where(fmt.Sprintf("%s <> ?", col), value)
			case "like":
				db = db.Where(fmt.Sprintf("%s LIKE ?", col), "%"+value+"%")
			case "gt":
				db = db.Where(fmt.Sprintf("%s > ?", col), value)
			case "gte":
				db = db.Where(fmt.Sprintf("%s >= ?", col), value)
			case "lt":
				db = db.Where(fmt.Sprintf("%s < ?", col), value)
			case "lte":
				db = db.Where(fmt.Sprintf("%s <= ?", col), value)
			case "in":
				db = db.Where(fmt.Sprintf("%s IN (?)", col), strings.Split(value, ","))
			default:
				// default fallback â†’ LIKE
				db = db.Where(fmt.Sprintf("%s LIKE ?", col), "%"+value+"%")
			}
		}

	}
	if strings.TrimSpace(search) != "" {

		// -------------------------------
		// GLOBAL SEARCH
		// -------------------------------
		like := "%" + search + "%"
		var orParts []string
		var params []interface{}

		for _, col := range allowedSearchColumns {
			orParts = append(orParts, fmt.Sprintf("%s LIKE ?", col))
			params = append(params, like)
		}

		db = db.Where("("+strings.Join(orParts, " OR ")+")", params...)
	}

	// -------------------------------
	// COUNT
	// -------------------------------
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	db = db.Order("id DESC")

	// -------------------------------
	// PAGINATION
	// -------------------------------
	if page != nil && limit != nil && *limit > 0 {
		p := *page
		l := *limit

		if p < 1 {
			p = 1
		}

		offset := (p - 1) * l
		db = db.Offset(offset).Limit(l)
	}

	// -------------------------------
	// EXECUTE QUERY
	// -------------------------------
	if err := db.Find(&Links).Error; err != nil {
		return nil, 0, err
	}

	return Links, total, nil
}

// ------------------------
// CREATE
// ------------------------
func (r *LinkRepository) Create(ctx context.Context, link *domainLink.Link) error {
	return r.db.WithContext(ctx).Create(link).Error
}

// ------------------------
// UPDATE
// ------------------------
func (r *LinkRepository) Update(ctx context.Context, link *domainLink.Link) error {
	return r.db.WithContext(ctx).Save(link).Error
}

// ------------------------
// DELETE (by UUID)
// ------------------------
func (r *LinkRepository) Delete(ctx context.Context, uid uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("uuid = ?", uid).
		Unscoped().
		Delete(&domainLink.Link{}).Error
}

func (r *LinkRepository) SetupUuid(ctx context.Context) error {
	const chunkSize = 500

	var ids []uint
	if err := r.db.WithContext(ctx).
		Debug().
		Model(&domainLink.Link{}).
		Where("uuid IS NULL OR uuid = ''").
		Pluck("id", &ids).Error; err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}

		chunk := ids[i:end]

		caseSQL := "CASE id "
		args := make([]any, 0, len(chunk)*2+1)

		for _, id := range chunk {
			u := uuid.NewString()
			caseSQL += "WHEN ? THEN ? "
			args = append(args, id, u)
		}

		caseSQL += "END"
		args = append(args, chunk)

		query := fmt.Sprintf(
			"UPDATE links SET uuid = %s WHERE id IN (?)",
			caseSQL,
		)

		if err := r.db.WithContext(ctx).
			Exec(query, args...).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *LinkRepository) WithTx(
	ctx context.Context,
	fn func(txRepo domainLink.ILinkRepositoryTx) error,
) error {

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txRepo := &LinkRepository{
			db:  tx,
			uow: commoninfra.NewUnitOfWork(tx),
		}
		return fn(txRepo)
	})
}

func (r *LinkRepository) InsertOutbox(
	ctx context.Context,
	msg *commoninfra.OutboxMessage,
) error {
	return r.db.WithContext(ctx).Create(msg).Error
}


