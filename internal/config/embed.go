package config

import _ "embed"

// ---------- Шаблон config.yaml ----------
// Используется nginx-lens init для создания конфига с комментариями.

//go:embed template.yaml
var ConfigTemplate string
