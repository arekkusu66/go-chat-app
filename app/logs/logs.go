package logs

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func InitLogger() {
	Log = slog.New(slog.NewTextHandler(os.Stderr, nil))
}