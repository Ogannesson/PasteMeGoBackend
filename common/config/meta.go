package config

import (
	"github.com/PasteUs/PasteMeGoBackend/common/logging"
	"go.uber.org/zap"
)

var version = "3.5.1"
var validConfigVersion = []string{"3.3.0", ""}

func init() {
	logging.Info("PasteMe Go Backend", zap.String("version", version))
}
