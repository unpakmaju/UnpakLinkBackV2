package presentation

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/mehdihadeli/go-mediatr"

	commondomain "UnpakSiamida/common/domain"
	"UnpakSiamida/common/helper"
	commoninfra "UnpakSiamida/common/infrastructure"
	commonpresentation "UnpakSiamida/common/presentation"
	Clickdomain "UnpakSiamida/modules/click/domain"

	GetAllClicks "UnpakSiamida/modules/click/application/GetAllClicks"
)

// =======================================================
// GET /Clicks
// =======================================================

// GetAllClicksHandler godoc
// @Summary Get all Clicks
// @Tags Click
// @Param mode query string false "paging | all | ndjson | sse"
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Param search query string false "Search keyword"
// @Produce json
// @Success 200 {object} Clickdomain.PagedClicks
// @Router /Clicks [get]
func GetAllClicksHandlerfunc(c *fiber.Ctx) error {
	sid := c.FormValue("sid")

	mode := c.Query("mode", "paging")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	search := c.Query("search", "")

	filtersRaw := c.Query("filters", "")
	var filters []commondomain.SearchFilter

	if filtersRaw != "" {
		parts := strings.Split(filtersRaw, ";")
		for _, p := range parts {
			tokens := strings.SplitN(p, ":", 3)
			if len(tokens) != 3 {
				continue
			}

			field := strings.TrimSpace(tokens[0])
			op := strings.TrimSpace(tokens[1])
			rawValue := strings.TrimSpace(tokens[2])

			var ptr *string
			if rawValue != "" && rawValue != "null" {
				ptr = &rawValue
			}

			filters = append(filters, commondomain.SearchFilter{
				Field:    field,
				Operator: op,
				Value:    ptr,
			})
		}
	}

	if sid != os.Getenv("Administrator") {
		filters = append(filters, commondomain.SearchFilter{
			Field:    "creator",
			Operator: "eq",
			Value:    &sid,
		})
	}

	query := GetAllClicks.GetAllClicksQuery{
		Search:        search,
		SearchFilters: filters,
	}

	var adapter OutputAdapter
	switch mode {
	case "all":
		adapter = &AllAdapter{}
	case "ndjson":
		adapter = &NDJSONAdapter{}
	case "sse":
		adapter = &SSEAdapter{}
	default:
		query.Page = &page
		query.Limit = &limit
		adapter = &PagingAdapter{}
	}

	result, err := mediatr.Send[
		GetAllClicks.GetAllClicksQuery,
		Clickdomain.PagedClicks,
	](context.Background(), query)

	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return adapter.Send(c, result)
}

func ModuleClick(app *fiber.App) {
	// admin := []string{"admin"}
	// whoamiURL := "http://localhost:3000/whoami"
	pubKey, err := helper.LoadRSAPublicKey("public.pem")

	if err != nil {
		log.Fatal(err)
	}

	app.Get("/api/clicks", commonpresentation.SmartCompress(), commonpresentation.JWTMiddleware(pubKey), GetAllClicksHandlerfunc)
}

