package chain

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

func NewLogger(v zerolog.Level, name string) zerolog.Logger {
	out := zerolog.ConsoleWriter{
		Out: os.Stdout,
		FormatLevel: func(i interface{}) string {
			if i == nil {
				return strings.ToUpper(fmt.Sprintf("[ %s ] |%-6s|", name, "INFO"))
			}
			return strings.ToUpper(fmt.Sprintf("[ %s ] |%-6s|", name, i))
		},
		TimeFormat: "06-01-02 15:04:05"}

	return zerolog.New(out).Level(v)
}
