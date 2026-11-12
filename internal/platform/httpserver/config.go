package httpserver

import (
	"os"
	"strings"
)

type Config struct {
	Address        string
	AllowedOrigins []string
	EnableOpenAPI  bool
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

	return Config{
		Address:        address,
		AllowedOrigins: allowedOrigins,
		EnableOpenAPI:  os.Getenv("ZWOA_ENABLE_OPENAPI") == "1",
	}
}
