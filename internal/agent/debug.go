package agent

import (
	"log"

	"github.com/ofkm/arcane-agent/internal/config"
)

func debugLog(cfg *config.Config, format string, args ...interface{}) {
	if cfg.Debug {
		log.Printf("[DEBUG] "+format, args...)
	}
}
