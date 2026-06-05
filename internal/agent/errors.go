package agent

import "errors"

var errNoConfigPath = errors.New("nginx config path is not provided and not set in config")
