package presentation

import (
	"context"
	"crypto/rsa"
	"errors"
	"html"
	"net"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	commoninfra "UnpakSiamida/common/infrastructure"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/text/unicode/norm"
)

// =======================
// CONFIG
// =======================
var jwtSecret = []byte("secret")

type HeaderSecurityConfig struct {
	BlacklistedHeaderNames map[string]bool
	AllowDomains           []string
	MaxHeaderLen           int
	ResolveAndCheck        bool
	LookupTimeout          time.Duration
	BlockedCIDRs           []string
}

func DefaultBlacklistedHeaderNames() map[string]bool {
	names := []string{
		"x-forwarded-for", "x-forwarded-host", "forwarded", "forwarded-host",
		"x-forwarded-proto", "x-forwarded-port", "x-forwarded-scheme",
		"x-real-ip", "client-ip", "true-client-ip", "cf-connecting-ip",
		"x-remote-ip", "x-originating-ip",
		"x-original-host", "via", "x-via",
		"host", "x-host", "x-rewrite-url", "x-original-url",
		"x-request-url", "x-request-uri", "redirect", "location",
		"authorization", "proxy-authorization", "x-api-key",
		"metadata", "x-aws-ec2-metadata", "referer",
	}
	out := map[string]bool{}
	for _, v := range names {
		out[strings.ToLower(v)] = true
	}
	return out
}

func DefaultHeaderSecurityConfig() *HeaderSecurityConfig {
	raw := os.Getenv("CORS_ALLOWED_ORIGINS")

	var domains []string

	if raw != "" {
		parts := strings.Split(raw, ",")
		for _, p := range parts {
			d := strings.TrimSpace(p)
			if d != "" {
				domains = append(domains, d)
			}
		}
	}

	// fallback default jika env kosong
	if len(domains) == 0 {
		domains = []string{
			"unpak.link",
			"localhost",
			"localhost:3000",
			"thunderclient.com",
		}
	}

	return &HeaderSecurityConfig{
		BlacklistedHeaderNames: DefaultBlacklistedHeaderNames(),
		AllowDomains:           domains,
		MaxHeaderLen:           8192,
		ResolveAndCheck:        false,
		LookupTimeout:          1 * time.Second,
		BlockedCIDRs:           []string{},
	}
}

// =======================
// REGEX
// =======================

var (
	crlfRe      = regexp.MustCompile(`[\r\n]`)
	nullRe      = regexp.MustCompile(`\x00`)
	protoRe     = regexp.MustCompile(`(?i)^(javascript|data|vbscript|file|view-source):`)
	punyRe      = regexp.MustCompile(`(?i)xn--[a-z0-9-]+`)
	zeroWidthRe = regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{2060}\x{FEFF}]`)
	hostExtract = regexp.MustCompile(`(?i)(?:https?://)?([a-z0-9\.\-]+\.[a-z]{2,})(:\d+)?`)
)

type Account struct {
	UUID         string      `json:"uuid"`
	NidnUsername string      `json:"nidn_username"`
	Password     string      `json:"password"`
	Level        string      `json:"level"`
	Name         string      `json:"name"`
	Email        string      `json:"email"`
	FakultasUnit string      `json:"fakultas_unit"`
	ExtraRole    []ExtraRole `gorm:"-" json:"extrarole,omitempty"`
}
type ExtraRole struct {
	Tahun string `json:"tahun"`
	Role  string `json:"role"`
}

// =======================
// MIDDLEWARE
// =======================

func HeaderSecurityMiddleware(cfg *HeaderSecurityConfig) fiber.Handler {
	if cfg == nil {
		cfg = DefaultHeaderSecurityConfig()
	}

	var blocked []*net.IPNet
	for _, c := range cfg.BlockedCIDRs {
		_, ipnet, err := net.ParseCIDR(c)
		if err == nil {
			blocked = append(blocked, ipnet)
		}
	}

	return func(c *fiber.Ctx) error {
		for name, vals := range c.GetReqHeaders() {
			for _, val := range vals {
				// 1) Max header length
				if len(val) > cfg.MaxHeaderLen {
					return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+1]", "header too long: "+name))
				}

				// 2) Malicious control chars
				if crlfRe.MatchString(val) || nullRe.MatchString(val) {
					return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+2]", "header ctrl char: "+name))
				}

				// 3) Decode value safely
				decoded := multiUnescape(html.UnescapeString(val), 3)
				decoded = zeroWidthRe.ReplaceAllString(decoded, "")
				decoded = norm.NFKC.String(decoded)

				// 4) Dangerous protocols
				if protoRe.MatchString(decoded) {
					return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+3]", "protocol attack: "+name))
				}

				// 5) Punycode / IDN injection
				if punyRe.MatchString(decoded) {
					return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+4]", "punycode forbidden: "+name))
				}

				// 6) Zero-width check
				if zeroWidthRe.MatchString(val) {
					return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+5]", "zero width attack: "+name))
				}

				// 7) Domain allowlist cek untuk URL valid
				u, err := url.Parse(decoded)
				if err == nil && u.Host != "" {
					host := u.Hostname()
					if !domainAllowed(host, cfg.AllowDomains) {
						return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+6]", "domain not allowed: "+host))
					}

					// Optional DNS resolve
					if cfg.ResolveAndCheck {
						ctx, cancel := context.WithTimeout(context.Background(), cfg.LookupTimeout)
						defer cancel()
						ips, _ := net.DefaultResolver.LookupIP(ctx, "ip", host)
						for _, ip := range ips {
							if ipInNets(ip, blocked) {
								return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+7]", "domain resolves to forbidden IP: "+host))
							}
						}
					}
				}

				// 8) Cek host header spoof (Host harus sama atau allow domain)
				// if strings.ToLower(name) == "host" {
				// 	host := decoded
				// 	if !domainAllowed(host, cfg.AllowDomains) {
				// 		return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+8]", "host header spoof: "+host))
				// 	}
				// }

				// 9) Cek embedded domain dalam text
				// hosts := extractHosts(decoded) //ini kenapa null
				// godump.Dd(hosts, cfg.AllowDomains)
				// for _, h := range hosts {
				// 	if !domainAllowed(h, cfg.AllowDomains) {
				// 		return c.Status(400).JSON(commoninfra.NewResponseError("common.check[A+9]", "embedded domain not allowed: "+h))
				// 	}
				// }

				// =======================
				// 1) Embedded domain check hanya untuk header URL
				// =======================
				urlHeaders := []string{"referer", "origin", "location", "refferer", "referrer", "redirect", "url", "http-url", "x-rewrite-url", "x-http-destinationurl", "x-http-host-override", "x-forwarded-host"}
				for _, h := range urlHeaders {
					val := c.Get(h)
					if val == "" {
						continue
					}

					decoded := multiUnescape(html.UnescapeString(val), 3)
					decoded = zeroWidthRe.ReplaceAllString(decoded, "")
					decoded = norm.NFKC.String(decoded)

					hosts := extractHostsFromText(decoded)
					// godump.Dd(hosts, cfg.AllowDomains)

					for _, host := range hosts {
						if !domainAllowed(host, cfg.AllowDomains) {
							return c.Status(400).JSON(commoninfra.NewResponseError(
								"common.check[A+9]", "embedded domain not allowed: "+host))
						}
					}
				}
			}
		}

		return c.Next()
	}
}

// func JWTMiddleware() fiber.Handler {
// 	return func(c *fiber.Ctx) error {
// 		authHeader := c.Get("Authorization")
// 		if authHeader == "" {
// 			return c.Status(400).JSON(commoninfra.NewResponseError("common.token", "authorization header missing"))
// 		}

// 		parts := strings.Split(authHeader, " ")
// 		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
// 			return c.Status(400).JSON(commoninfra.NewResponseError("common.token", "authorization header format must be Bearer token"))
// 		}

// 		tokenStr := parts[1]

// 		// Parse & validate token
// 		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
// 			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
// 				return nil, errors.New("invalid signing method")
// 			}
// 			return jwtSecret, nil
// 		})

// 		if err != nil || !token.Valid {
// 			if err != nil {
// 				return c.Status(400).JSON(commoninfra.NewResponseError("common.token", err.Error()))
// 			}
// 			return c.Status(400).JSON(commoninfra.NewResponseError("common.token", "invalid token"))
// 		}

// 		claims, ok := token.Claims.(jwt.MapClaims)
// 		if !ok {
// 			return c.Status(400).JSON(commoninfra.NewResponseError("common.token", "invalid token claims"))
// 		}

// 		if exp, ok := claims["exp"].(float64); ok && int64(exp) < time.Now().Unix() {
// 			return c.Status(400).JSON(commoninfra.NewResponseError("common.token", "token expired"))
// 		}

// 		// Inject sid ke form value
// 		if sid, ok := claims["sid"].(string); ok {
// 			c.Request().PostArgs().Set("sid", sid)
// 		}

// 		c.Request().PostArgs().Set("token", tokenStr)
// 		// c.Locals("token", tokenStr)

//			// lanjut ke handler berikutnya
//			return c.Next()
//		}
//	}
func JWTMiddleware(publicKey *rsa.PublicKey) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authHeader := c.Get("Authorization")

		if authHeader == "" {
			return c.Status(400).JSON(
				commoninfra.NewResponseError(
					"common.token",
					"authorization header missing",
				),
			)
		}

		parts := strings.Split(authHeader, " ")

		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(400).JSON(
				commoninfra.NewResponseError(
					"common.token",
					"authorization header format must be Bearer token",
				),
			)
		}

		tokenStr := parts[1]

		/* =========================
		 * PARSE TOKEN
		 * ========================= */
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); ok {
				return jwtSecret, nil
			}
			if _, ok := t.Method.(*jwt.SigningMethodRSA); ok {
				return publicKey, nil
			}

			return nil, errors.New("invalid signing method")
		})

		if err != nil {
			return c.Status(401).JSON(
				commoninfra.NewResponseError(
					"common.token",
					err.Error(),
				),
			)
		}

		if !token.Valid {
			return c.Status(401).JSON(
				commoninfra.NewResponseError(
					"common.token",
					"invalid token",
				),
			)
		}

		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			return c.Status(401).JSON(
				commoninfra.NewResponseError(
					"common.token",
					"invalid token claims",
				),
			)
		}

		/* =========================
		 * EXP CHECK
		 * ========================= */
		if exp, ok := claims["exp"].(float64); ok {
			if int64(exp) < time.Now().Unix() {
				return c.Status(401).JSON(
					commoninfra.NewResponseError(
						"common.token",
						"token expired",
					),
				)
			}
		}

		/* =========================
		 * SAVE CLAIMS
		 * ========================= */
		var sid string

		// jika RSA (Keycloak SSO) ambil employeeid / adm_pusat
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			var groups []string

			if memberOf, ok := claims["group"].([]interface{}); ok {
				for _, v := range memberOf {
					if s, ok := v.(string); ok {
						groups = append(groups, s)
					}
				}
			}

			if slices.Contains(groups, "adm_pusat") {
				sid = "putiklink"
			} else if val, ok := claims["employeeid"].(string); ok {
				sid = val
			}
		} else {
			// selain itu ambil sid
			if val, ok := claims["sid"].(string); ok {
				sid = val
			}
		}

		var creatorName string
		if val, ok := claims["name"].(string); ok && val != "" {
			creatorName = val
		} else if val, ok := claims["preferred_username"].(string); ok && val != "" {
			creatorName = val
		}

		c.Request().PostArgs().Set("sid", sid)
		c.Request().PostArgs().Set("creator_name", creatorName)
		c.Request().PostArgs().Set("token", tokenStr)

		return c.Next()
	}
}

func getTahun(c *fiber.Ctx) string {
	if t := c.Params("tahun"); t != "" {
		return t
	}
	return c.Query("ctxtahun")
}

func RBACMiddleware(whitelist []string, whoamiURL string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		log.Printf("[RBAC] Authorization header: %s", authHeader)

		if authHeader == "" {
			log.Println("[RBAC] Authorization header missing")
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "authorization header missing"))
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Println("[RBAC] Invalid authorization header format")
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "authorization header format must be Bearer token"))
		}
		token := parts[1]
		log.Printf("[RBAC] Token: %s", token)

		req, err := http.NewRequest("GET", whoamiURL, nil)
		if err != nil {
			log.Printf("[RBAC] Failed to create request: %v", err)
			return c.Status(500).JSON(commoninfra.NewResponseError("common.rbac", "Failed to create request: "+err.Error()))
		}
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[RBAC] Failed to call whoami: %v", err)
			return c.Status(500).JSON(commoninfra.NewResponseError("common.rbac", "Failed to call whoami: "+err.Error()))
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("[RBAC] Whoami response status: %d, body: %s", resp.StatusCode, string(body))

		if resp.StatusCode != 200 {
			var errResp struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &errResp); err == nil && errResp.Message != "" {
				log.Printf("[RBAC] Whoami error code: %s, message: %s", errResp.Code, errResp.Message)
				return c.Status(400).JSON(commoninfra.NewResponseError(errResp.Code, errResp.Message))
			}
			log.Println("[RBAC] Whoami response not JSON or invalid format")
			return c.Status(401).JSON(commoninfra.NewResponseError("common.rbac", "Invalid format response"))
		}

		var user Account
		if err := json.Unmarshal(body, &user); err != nil {
			log.Printf("[RBAC] Failed to parse whoami response: %v", err)
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "Failed to parse whoami response"))
		}
		log.Printf("[RBAC] Whoami user: %+v", user)

		if user.Level == "admin" {
			log.Println("[RBAC] User is admin, access granted")
			return c.Next()
		}

		tahun := getTahun(c) //c.Query("tahun") -> /?tahun=
		log.Printf("[RBAC] Tahun: %s", tahun)
		if tahun == "" {
			log.Println("[RBAC] Tahun query param missing")
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "Query parameter 'tahun' is required"))
		}
		tahunInt, err := strconv.Atoi(tahun)
		if err != nil || tahunInt < 2000 {
			log.Printf("[RBAC] Tahun query invalid: %s", tahun)
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "Query parameter 'tahun' invalid"))
		}

		hasAccess := false
		grantedAccess := []string{}
		for _, r := range user.ExtraRole {
			key := r.Tahun + "#" + r.Role
			grantedAccess = append(grantedAccess, key)

			log.Printf("[RBAC] Acces data: %s = %s", r.Tahun, tahun)
			if r.Tahun == tahun {
				role := strings.ToLower(strings.TrimSpace(r.Role))
				for _, w := range whitelist {
					log.Printf("[RBAC] role: %s = %s", role, strings.ToLower(w))
					if role == strings.ToLower(w) {
						hasAccess = true
						log.Printf("[RBAC] User has role '%s' for tahun %s, access granted", r.Role, r.Tahun)
						break
					}
				}
			}
			if hasAccess {
				break
			}
		}

		if !hasAccess {
			log.Println("[RBAC] Access denied")
			return c.Status(400).JSON(commoninfra.NewResponseError("common.rbac", "Access denied"))
		}

		log.Println("[RBAC] Middleware passed, continue to handler")
		grantedAccessStr := strings.Join(grantedAccess, ", ")
		c.Request().PostArgs().Set("grantedaccess", grantedAccessStr)

		return c.Next()
	}
}

func WSError(conn *websocket.Conn, code string, msg string) error {

	conn.WriteJSON(map[string]interface{}{
		"code":        code,
		"description": msg,
	})

	conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(
			websocket.ClosePolicyViolation,
			msg,
		),
		time.Now().Add(time.Second),
	)

	conn.Close()
	return errors.New(msg)
}

type WSSession struct {
	Token         string
	SID           string
	User          *Account
	GrantedAccess []string
}

// =======================
// HELPERS
// =======================

//	func domainAllowed(host string, allow []string) bool {
//		host = strings.ToLower(host)
//		for _, a := range allow {
//			if strings.HasSuffix(host, strings.ToLower(a)) {
//				return true
//			}
//		}
//		return false
//	}
func domainAllowed(host string, allow []string) bool {
	host = strings.ToLower(host)
	for _, a := range allow {
		u, err := url.Parse(a)
		var domain string
		if err == nil && u.Host != "" {
			domain = u.Hostname()
		} else {
			domain = strings.ToLower(a)
		}
		if strings.HasSuffix(host, domain) {
			return true
		}
	}
	return false
}

//	func extractHosts(s string) []string {
//		out := []string{}
//		words := strings.Fields(s)
//		for _, w := range words {
//			u, err := url.Parse(w)
//			if err == nil && u.Host != "" {
//				out = append(out, u.Hostname())
//				continue
//			}
//			m := hostExtract.FindStringSubmatch(w)
//			if len(m) > 1 {
//				out = append(out, m[1])
//			}
//		}
//		return out
//	}
func extractHostsFromText(s string) []string {
	out := []string{}

	// Split string berdasarkan spasi, koma, titik koma, newline
	words := regexp.MustCompile(`[ \t\r\n,;]+`).Split(s, -1)

	for _, w := range words {
		if w == "" {
			continue
		}

		// Hanya parsing kata yang terlihat seperti URL
		if strings.Contains(w, "://") {
			u, err := url.Parse(w)
			if err == nil && u.Host != "" {
				out = append(out, u.Hostname())
				continue
			}
		}

		// Fallback regex: cocokkan host sederhana (domain.tld)
		m := hostExtract.FindStringSubmatch(w)
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}

	return out
}

func multiUnescape(s string, n int) string {
	cur := s
	for i := 0; i < n; i++ {
		u, err := url.QueryUnescape(cur)
		if err != nil || u == cur {
			break
		}
		cur = u
	}
	return cur
}

func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func SmartCompress() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ct := string(c.Response().Header.ContentType())

		// Jangan compress streaming
		if strings.Contains(ct, "text/event-stream") ||
			strings.Contains(ct, "application/x-ndjson") {
			return c.Next()
		}

		return compress.New(compress.Config{
			Level: compress.LevelBestCompression,
		})(c)
	}
}
