package main

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/ckovacs/web-stats-mgmt/internal/pagespeed"
)

const analysisTimeout = 2 * time.Minute

type scoresRequest struct {
	URL      string `json:"url"`
	Token    string `json:"token"`
	Strategy string `json:"strategy"`
}

func scoresHandler(psi *pagespeed.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req scoresRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "body must be JSON with \"url\" and \"token\"")
		}

		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			return fiber.NewError(fiber.StatusBadRequest, "\"token\" is required")
		}
		if !isHTTPURL(req.URL) {
			return fiber.NewError(fiber.StatusBadRequest, "\"url\" must be an absolute http(s) URL")
		}

		strategy := "MOBILE"
		switch strings.ToLower(req.Strategy) {
		case "", "mobile":
		case "desktop":
			strategy = "DESKTOP"
		default:
			return fiber.NewError(fiber.StatusBadRequest, "\"strategy\" must be \"mobile\" or \"desktop\"")
		}

		ctx, cancel := context.WithTimeout(context.Background(), analysisTimeout)
		defer cancel()

		res, err := psi.Run(ctx, req.Token, req.URL, strategy)
		if err != nil {
			var apiErr *pagespeed.APIError
			if errors.As(err, &apiErr) {
				// Pass Google's client errors (bad key, quota, unreachable page) through;
				// anything else from Google is an upstream failure.
				code := apiErr.StatusCode
				if code < 400 || code >= 500 {
					code = fiber.StatusBadGateway
				}
				return fiber.NewError(code, apiErr.Message)
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return fiber.NewError(fiber.StatusGatewayTimeout, "pagespeed analysis timed out")
			}
			return fiber.NewError(fiber.StatusBadGateway, "failed to reach pagespeed api")
		}

		return c.JSON(res)
	}
}

func isHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
