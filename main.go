package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/mehdihadeli/go-mediatr"

	"github.com/gofiber/fiber/v2/middleware/helmet"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	linkInfrastructure "UnpakSiamida/modules/link/infrastructure"

	linkPresentation "UnpakSiamida/modules/link/presentation"

	accountInfrastructure "UnpakSiamida/modules/account/infrastructure"

	accountPresentation "UnpakSiamida/modules/account/presentation"

	clickInfrastructure "UnpakSiamida/modules/click/infrastructure"

	clickPresentation "UnpakSiamida/modules/click/presentation"
	/////////

	validation "github.com/go-ozzo/ozzo-validation/v4"

	commoninfra "UnpakSiamida/common/infrastructure"

	commonpresentation "UnpakSiamida/common/presentation"

	//////////

	login "UnpakSiamida/modules/account/application/Login"

	// whoami "UnpakSiamida/modules/account/application/Whoami"

	createLink "UnpakSiamida/modules/link/application/CreateLink"

	updateLink "UnpakSiamida/modules/link/application/UpdateLink"

	givePasswordLink "UnpakSiamida/modules/link/application/GivePassword"

	rollbackPasswordLink "UnpakSiamida/modules/link/application/RollbackPassword"

	giveTimeLink "UnpakSiamida/modules/link/application/TimeLink"

	rollbackTimeLink "UnpakSiamida/modules/link/application/RollbackTime"

	moveLink "UnpakSiamida/modules/link/application/MoveLink"

	deleteLink "UnpakSiamida/modules/link/application/DeleteLink"

	// createClick "UnpakSiamida/modules/click/application/CreateClick"

	// eventBeritaAcara "UnpakSiamida/modules/beritaacara/event"
	eventLink "UnpakSiamida/modules/link/event"
	// eventUser "UnpakSiamida/modules/user/event"

	_ "UnpakSiamida/docs"

	"github.com/gofiber/swagger"
	_ "github.com/swaggo/files"

	"github.com/gofiber/fiber/v2/middleware/recover"
)

var startupErrors []fiber.Map

func mustStart(name string, fn func() error) {
	if err := fn(); err != nil {
		startupErrors = append(startupErrors, fiber.Map{
			"module": name,
			"error":  err.Error(),
		})
	}
}

// @title UnpakSiamidaV2 API
// @version 1.0
// @description All Module Siamida
// @host localhost:3000
// @BasePath /
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := commonpresentation.DefaultHeaderSecurityConfig()
	cfg.ResolveAndCheck = false

	app := fiber.New(fiber.Config{
		// DisableStartupMessage: true,
		ReadBufferSize: 32 * 1024,
		// Prefork:        true, // gunakan semua CPU cores
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  10 * time.Second,
		ProxyHeader:  fiber.HeaderXForwardedFor,
	})
	app.Use(recover.New(recover.Config{
		EnableStackTrace: os.Getenv("APP_DEBUG") == "true",
	}))

	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: "*",
	// 	AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
	// 	AllowHeaders: "*",
	// }))
	app.Use(helmet.New(helmet.Config{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",     // X-Content-Type-Options
		XFrameOptions:             "DENY",        // X-Frame-Options
		ReferrerPolicy:            "no-referrer", // Referrer-Policy
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self'; object-src 'none'; base-uri 'none'",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}))
	app.Use(commonpresentation.LoggerMiddleware)
	app.Use(commonpresentation.HeaderSecurityMiddleware(cfg))
	app.Use(func(c *fiber.Ctx) error {
		c.Response().Header.Del("X-Powered-By")
		return c.Next()
	})

	mediatr.RegisterRequestPipelineBehaviors(NewValidationBehavior())

	var db *gorm.DB
	var dbSimak *gorm.DB

	DB_SOURCE := os.Getenv("DB_SOURCE")
	DB_SIMAK := os.Getenv("DB_SIMAK")

	mustStart("Database Main", func() error {
		var err error
		db, err = NewMySQL(
			DB_SOURCE,
		)
		log.Println("Database Main initialized")
		return err
	})

	mustStart("Database Simak", func() error {
		var err error
		log.Println("Database Simak initialized")
		dbSimak, err = NewMySQL(
			DB_SIMAK,
		)
		return err
	})

	//berlaku untuk startup bukan hot reload
	mustStart("Account Module", func() error {
		return accountInfrastructure.RegisterModuleAccount(dbSimak)
	})

	mustStart("Link Module", func() error {
		return linkInfrastructure.RegisterModuleLink(db)
	})

	mustStart("Click Module", func() error {
		return clickInfrastructure.RegisterModuleClick(db)
	})

	if len(startupErrors) > 0 {
		app.Use(func(c *fiber.Ctx) error {
			return c.Status(500).JSON(fiber.Map{
				"Code":    "INTERNAL_SERVER_ERROR",
				"Message": "Startup module failed",
				"Trace":   startupErrors,
			})
		})
	}

	dispatcher := commoninfra.NewEventDispatcher()
	commoninfra.RegisterEvent[eventLink.LinkCountEvent](dispatcher)
	// commoninfra.RegisterEvent[eventKts.KtsUpdatedEvent](dispatcher)
	// commoninfra.RegisterEvent[eventUser.UserCreatedEvent](dispatcher)
	// commoninfra.RegisterEvent[eventUser.UserUpdatedEvent](dispatcher)
	// commoninfra.RegisterEvent[eventBeritaAcara.BeritaAcaraPdfRequestedEvent](dispatcher)
	// commoninfra.RegisterEvent[eventKts.KtsPdfRequestedEvent](dispatcher)

	linkPresentation.ModuleLink(app)
	accountPresentation.ModuleAccount(app)
	clickPresentation.ModuleClick(app)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	outboxProcessor := &commoninfra.OutboxProcessor{
		DB:         db,
		Dispatcher: dispatcher,
	}

	crontabProcessor := &linkInfrastructure.CrontabProcessor{
		DB: db,
	}

	if os.Getenv("APP_DEBUG") == "true" { app.Get("/swagger/*", swagger.HandlerDefault) }
	go commoninfra.StartOutboxWorker(ctx, outboxProcessor)
	go linkInfrastructure.StartCrontabWorker(ctx, crontabProcessor)
	app.Listen(":3000")
}

type ValidationBehavior struct{}

func NewValidationBehavior() *ValidationBehavior {
	return &ValidationBehavior{}
}

func (b *ValidationBehavior) Handle(
	ctx context.Context,
	request interface{},
	next mediatr.RequestHandlerFunc,
) (interface{}, error) {

	switch cmd := request.(type) {
	// === link Commands ===
	case createLink.CreateLinkCommand:
		if err := createLink.CreateLinkCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkCreate.Validation", err)
		}
	case updateLink.UpdateLinkCommand:
		if err := updateLink.UpdateLinkCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkUpdate.Validation", err)
		}
	case givePasswordLink.GivePasswordCommand:
		if err := givePasswordLink.GivePasswordCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkPassword.Validation", err)
		}
	case rollbackPasswordLink.RollbackPasswordCommand:
		if err := rollbackPasswordLink.RollbackPasswordCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkRollbackPassword.Validation", err)
		}
	case giveTimeLink.TimeLinkCommand:
		if err := giveTimeLink.TimeLinkCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkTime.Validation", err)
		}
	case rollbackTimeLink.RollbackTimeCommand:
		if err := rollbackTimeLink.RollbackTimeCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkRollbackTime.Validation", err)
		}
	case moveLink.MoveLinkCommand:
		if err := moveLink.MoveLinkCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkMove.Validation", err)
		}
	case deleteLink.DeleteLinkCommand:
		if err := deleteLink.DeleteLinkCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("LinkDelete.Validation", err)
		}

	// === Generate Login Commands ===
	case login.LoginCommand:
		if err := login.LoginCommandValidation(cmd); err != nil {
			return nil, wrapValidationError("Login.Validation", err)
		}

	default:
		// request lain â†’ skip validation
	}

	return next(ctx)
}

func wrapValidationError(code string, err error) error {
	if ve, ok := err.(validation.Errors); ok {
		msgs := make(map[string]string)
		for field, ferr := range ve {
			key := strings.ToLower(field)
			msgs[key] = ferr.Error()
		}
		return commoninfra.NewResponseError(code, msgs)
	}
	return commoninfra.NewResponseError(code, err.Error())
}

func NewMySQL(dsn string) (*gorm.DB, error) {

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	return db, nil
}

