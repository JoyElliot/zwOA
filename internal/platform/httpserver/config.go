package httpserver

import (
	"net/netip"
	"os"
	"strings"
)

type Config struct {
	Address        string
	AllowedOrigins []string
	EnableOpenAPI  bool
	SecureCookies  bool
	TrustedProxies []netip.Addr
}

func ConfigFromEnv() Config {
	address := strings.TrimSpace(os.Getenv("ZWOA_API_ADDR"))
	if address == "" {
		address = "127.0.0.1:8720"
	}

	origins, configured := os.LookupEnv("ZWOA_ALLOWED_ORIGINS")
	if !configured {
		origins = "http://tauri.localhost"
	}
	var allowedOrigins []string
	for origin := range strings.SplitSeq(origins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}
	var trustedProxies []netip.Addr
	for entry := range strings.SplitSeq(os.Getenv("ZWOA_TRUSTED_PROXIES"), ",") {
		if address, err := netip.ParseAddr(strings.TrimSpace(entry)); err == nil {
			trustedProxies = append(trustedProxies, address.Unmap())
		}
	}

	return Config{
		Address:        address,
		AllowedOrigins: allowedOrigins,
		EnableOpenAPI:  os.Getenv("ZWOA_ENABLE_OPENAPI") == "1",
		SecureCookies:  os.Getenv("ZWOA_SECURE_COOKIES") == "1",
		TrustedProxies: trustedProxies,
	}
}
