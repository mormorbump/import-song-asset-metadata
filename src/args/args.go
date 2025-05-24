package args

import (
	"fmt"
	"os"
	"strings"
)

// Config はアプリケーションの設定を管理
type Config struct {
	ForceOverwrite bool
}

// ParseArgs はコマンドライン引数を解析
func ParseArgs() (inputPath string, config *Config, err error) {
	config = &Config{}

	if len(os.Args) < 2 {
		return "", nil, fmt.Errorf("insufficient arguments")
	}

	args := os.Args[1:]
	var inputFound bool

	for _, arg := range args {
		switch arg {
		case "--force", "-f":
			config.ForceOverwrite = true
		case "--help", "-h":
			return "", nil, fmt.Errorf("help requested")
		default:
			if !inputFound && !strings.HasPrefix(arg, "-") {
				inputPath = arg
				inputFound = true
			}
		}
	}

	if !inputFound {
		return "", nil, fmt.Errorf("no input path specified")
	}

	return inputPath, config, nil
}