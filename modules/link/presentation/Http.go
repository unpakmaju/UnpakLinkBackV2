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
	Linkdomain "UnpakSiamida/modules/link/domain"

	CreateLink "UnpakSiamida/modules/link/application/CreateLink"
	DeleteLink "UnpakSiamida/modules/link/application/DeleteLink"
	GetAllLinks "UnpakSiamida/modules/link/application/GetAllLinks"
	GetLink "UnpakSiamida/modules/link/application/GetLink"
	GetLinkDefault "UnpakSiamida/modules/link/application/GetLinkDefault"
	SetupUuidLink "UnpakSiamida/modules/link/application/SetupUuidLink"
	UpdateLink "UnpakSiamida/modules/link/application/UpdateLink"

	PasswordLink "UnpakSiamida/modules/link/application/GivePassword"
	MoveLink "UnpakSiamida/modules/link/application/MoveLink"
	RollbackPasswordLink "UnpakSiamida/modules/link/application/RollbackPassword"
	RollbackTimeLink "UnpakSiamida/modules/link/application/RollbackTime"
	TimeLink "UnpakSiamida/modules/link/application/TimeLink"
)

// =======================================================
// POST /link
// =======================================================

// CreateLinkHandler godoc
// @Summary Create new Link
// @Tags Link
// @Param shortUrl formData string true "Short Url"
// @Param longUrl formData string true "Long Url"
// @Param password formData string false "Password"
// @Param start formData string false "Start Date"
// @Param end formData string false "End Date"
// @Produce json
// @Success 200 {object} map[string]string "uuid of created Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link [post]
func CreateLinkHandlerfunc(c *fiber.Ctx) error {

	shortUrl := c.FormValue("shortUrl")
	longUrl := c.FormValue("longUrl")
	password := helper.StrPtr(c.FormValue("password"))
	start := helper.StrPtr(c.FormValue("start"))
	end := helper.StrPtr(c.FormValue("end"))
	sid := c.FormValue("sid")
	creatorName := c.FormValue("creator_name")

	cmd := CreateLink.CreateLinkCommand{
		ShortUrl:    shortUrl,
		LongUrl:     longUrl,
		Password:    password,
		Start:       start,
		End:         end,
		Creator:     sid,
		CreatorName: creatorName,
	}

	uuid, err := mediatr.Send[CreateLink.CreateLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return commonpresentation.JsonUUID(c, uuid)
}

// =======================================================
// PUT /link/{uuid}
// =======================================================

// UpdateLinkHandler godoc
// @Summary Update existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Param shortUrl formData string true "Short Url"
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [put]
func UpdateLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	shortUrl := c.FormValue("shortUrl")
	sid := c.FormValue("sid")

	cmd := UpdateLink.UpdateLinkCommand{
		Uuid:     uuid,
		ShortUrl: shortUrl,
		Creator:  sid,
	}

	updatedID, err := mediatr.Send[UpdateLink.UpdateLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// PUT /link/password/{uuid}
// =======================================================

// PasswordLinkHandler godoc
// @Summary Give password existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Param password formData string true "password"
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [put]
func PasswordLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	password := c.FormValue("password")
	sid := c.FormValue("sid")

	cmd := PasswordLink.GivePasswordCommand{
		Uuid:     uuid,
		Password: password,
		Creator:  sid,
	}

	updatedID, err := mediatr.Send[PasswordLink.GivePasswordCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// PUT /link/rollback-password/{uuid}
// =======================================================

// RolebackPasswordLinkHandler godoc
// @Summary Update existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [put]
func RolebackPasswordLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	sid := c.FormValue("sid")

	cmd := RollbackPasswordLink.RollbackPasswordCommand{
		Uuid:    uuid,
		Creator: sid,
	}

	updatedID, err := mediatr.Send[RollbackPasswordLink.RollbackPasswordCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// PUT /link/time/{uuid}
// =======================================================

// TimeLinkHandler godoc
// @Summary Give time existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Param start formData string true "Time start (Y-m-d H:i:s)"
// @Param end formData string true "Time end (Y-m-d H:i:s)"
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [put]
func TimeLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	start := c.FormValue("start")
	end := c.FormValue("end")
	sid := c.FormValue("sid")

	cmd := TimeLink.TimeLinkCommand{
		Uuid:    uuid,
		Start:   start,
		End:     end,
		Creator: sid,
	}

	updatedID, err := mediatr.Send[TimeLink.TimeLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// PUT /link/rollback-time/{uuid}
// =======================================================

// RolebackTimeLinkHandler godoc
// @Summary Update existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [put]
func RolebackTimeLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	sid := c.FormValue("sid")

	cmd := RollbackTimeLink.RollbackTimeCommand{
		Uuid:    uuid,
		Creator: sid,
	}

	updatedID, err := mediatr.Send[RollbackTimeLink.RollbackTimeCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// PUT /link/move/{uuid}/{state}
// =======================================================

// MoveLinkHandler godoc
// @Summary Update existing Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Param state path string true "state"
// @Produce json
// @Success 200 {object} map[string]string "uuid of updated Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid}/{state} [put]
func MoveLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	state := c.Params("state")
	sid := c.FormValue("sid")

	cmd := MoveLink.MoveLinkCommand{
		Uuid:    uuid,
		State:   state,
		Creator: sid,
	}

	updatedID, err := mediatr.Send[MoveLink.MoveLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": updatedID})
}

// =======================================================
// DELETE /link/{uuid}
// =======================================================

// DeleteLinkHandler godoc
// @Summary Delete a Link
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Produce json
// @Success 200 {object} map[string]string "uuid of deleted Link"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /link/{uuid} [delete]
func DeleteLinkHandlerfunc(c *fiber.Ctx) error {

	uuid := c.Params("uuid")
	sid := c.FormValue("sid")

	cmd := DeleteLink.DeleteLinkCommand{
		Uuid:    uuid,
		Creator: sid,
	}

	deletedID, err := mediatr.Send[DeleteLink.DeleteLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"uuid": deletedID})
}

// =======================================================
// GET /Link/{uuid}
// =======================================================

// GetLinkHandler godoc
// @Summary Get Link by UUID
// @Tags Link
// @Param uuid path string true "Link UUID" format(uuid)
// @Produce json
// @Success 200 {object} Linkdomain.Link
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /Link/{uuid} [get]
func GetLinkHandlerfunc(c *fiber.Ctx) error {
	uuid := c.Params("uuid")

	query := GetLink.GetLinkByUuidQuery{Uuid: uuid}

	Link, err := mediatr.Send[
		GetLink.GetLinkByUuidQuery,
		*Linkdomain.Link,
	](context.Background(), query)

	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	if Link == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Link not found"})
	}

	return c.JSON(Link)
}

// =======================================================
// GET /Link/short/{short}
// =======================================================

// GetLinkByShortHandler godoc
// @Summary Get Link by short
// @Tags Link
// @Param short path string true "Short Code"
// @Produce json
// @Success 200 {object} Linkdomain.Link
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /Link/{uuid} [get]
func GetLinkByShortHandlerfunc(c *fiber.Ctx) error {
	short := c.Params("short")
	apiKey := c.Get("X-API-KEY")

	ip := c.Get("X-Forwarded-For")
	if ip == "" {
		ip = c.IP()
	}
	ipClient := c.Get("X-IP")

	iso := c.Get("X-ISO")
	country := c.Get("X-COUNTRY")
	referer := c.Get("Referer")
	userAgent := c.Get("User-Agent")

	query := GetLinkDefault.GetLinkDefaultByShortQuery{
		Short:     short,
		DoCounter: apiKey == os.Getenv("Administrator"),
		IpClient:  helper.StrPtr(ipClient),
		IP:        helper.StrPtr(ip),
		ISO:       helper.StrPtr(iso),
		Country:   helper.StrPtr(country),
		Referer:   helper.StrPtr(referer),
		UserAgent: helper.StrPtr(userAgent),
	}

	Link, err := mediatr.Send[
		GetLinkDefault.GetLinkDefaultByShortQuery,
		*Linkdomain.LinkDefault,
	](context.Background(), query)

	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	if Link == nil {
		return c.Status(404).JSON(fiber.Map{"error": "Link not found"})
	}

	return c.JSON(Link)
}

// =======================================================
// GET /Links
// =======================================================

// GetAllLinksHandler godoc
// @Summary Get all Links
// @Tags Link
// @Param mode query string false "paging | all | ndjson | sse"
// @Param page query int false "Page number"
// @Param limit query int false "Limit per page"
// @Param search query string false "Search keyword"
// @Produce json
// @Success 200 {object} Linkdomain.PagedLinks
// @Router /Links [get]
func GetAllLinksHandlerfunc(c *fiber.Ctx) error {
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

	query := GetAllLinks.GetAllLinksQuery{
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
		GetAllLinks.GetAllLinksQuery,
		Linkdomain.PagedLinks,
	](context.Background(), query)

	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return adapter.Send(c, result)
}

var reservedShortSlugs = map[string]bool{
	"dashboard":    true,
	"link":         true,
	"links":        true,
	"callback":     true,
	"callback_sso": true,
	"api":          true,
	"assets":       true,
	"login":        true,
	"logout":       true,
	"swagger":      true,
}

func CheckLinkAvailabilityHandlerfunc(c *fiber.Ctx) error {
	short := strings.TrimSpace(c.Params("short"))
	excludeUuid := strings.TrimSpace(c.Query("exclude_uuid"))

	if short == "" {
		return c.Status(400).JSON(fiber.Map{
			"available": false,
			"shortUrl":  short,
			"reason":    "empty",
			"message":   "Link pendek tidak boleh kosong",
		})
	}

	if reservedShortSlugs[strings.ToLower(short)] {
		return c.Status(200).JSON(fiber.Map{
			"available": false,
			"shortUrl":  short,
			"reason":    "reserved",
			"message":   "Nama link ini dicadangkan oleh sistem",
		})
	}

	query := GetLinkDefault.GetLinkDefaultByShortQuery{
		Short:     short,
		DoCounter: false,
	}

	link, err := mediatr.Send[
		GetLinkDefault.GetLinkDefaultByShortQuery,
		*Linkdomain.LinkDefault,
	](context.Background(), query)

	if err != nil || link == nil {
		return c.Status(200).JSON(fiber.Map{
			"available": true,
			"shortUrl":  short,
			"message":   "Link pendek tersedia dan dapat digunakan",
		})
	}

	if excludeUuid != "" && strings.EqualFold(link.UUID.String(), excludeUuid) {
		return c.Status(200).JSON(fiber.Map{
			"available": true,
			"shortUrl":  short,
			"message":   "Ini adalah link pendek Anda saat ini",
		})
	}

	return c.Status(200).JSON(fiber.Map{
		"available": false,
		"shortUrl":  short,
		"reason":    "taken",
		"message":   "Link pendek sudah digunakan oleh pengguna lain",
	})
}

func SetupUuidLinksHandlerfunc(c *fiber.Ctx) error {
	cmd := SetupUuidLink.SetupUuidLinkCommand{}

	message, err := mediatr.Send[SetupUuidLink.SetupUuidLinkCommand, string](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{"message": message})
}

func ModuleLink(app *fiber.App) {
	// admin := []string{"admin"}
	// whoamiURL := "http://localhost:3000/whoami"
	pubKey, err := helper.LoadRSAPublicKey("public.pem")

	if err != nil {
		log.Fatal(err)
	}

	app.Get("/api/link/setupuuid", commonpresentation.JWTMiddleware(pubKey), SetupUuidLinksHandlerfunc)

	app.Post("/api/link", commonpresentation.JWTMiddleware(pubKey), CreateLinkHandlerfunc)
	app.Put("/api/link/:uuid", commonpresentation.JWTMiddleware(pubKey), UpdateLinkHandlerfunc)
	app.Put("/api/link/password/:uuid", commonpresentation.JWTMiddleware(pubKey), PasswordLinkHandlerfunc)
	app.Put("/api/link/rollback-password/:uuid", commonpresentation.JWTMiddleware(pubKey), RolebackPasswordLinkHandlerfunc)
	app.Put("/api/link/time/:uuid", commonpresentation.JWTMiddleware(pubKey), TimeLinkHandlerfunc)
	app.Put("/api/link/rollback-time/:uuid", commonpresentation.JWTMiddleware(pubKey), RolebackTimeLinkHandlerfunc)
	app.Put("/api/link/:uuid/:state", commonpresentation.JWTMiddleware(pubKey), MoveLinkHandlerfunc)

	app.Delete("/api/link/:uuid", commonpresentation.JWTMiddleware(pubKey), DeleteLinkHandlerfunc)
	app.Get("/api/link/:uuid", commonpresentation.SmartCompress(), commonpresentation.JWTMiddleware(pubKey), GetLinkHandlerfunc)
	app.Get("/api/links", commonpresentation.SmartCompress(), commonpresentation.JWTMiddleware(pubKey), GetAllLinksHandlerfunc)

	app.Get("/api/link/check/:short", commonpresentation.SmartCompress(), CheckLinkAvailabilityHandlerfunc)
	app.Get("/api/link/short/:short", commonpresentation.SmartCompress(), GetLinkByShortHandlerfunc)
}


