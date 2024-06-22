package common

import (
	"fmt"
)

const Version = "v0.1.0"

func PrintVersion(tool string, version string) string {
	return fmt.Sprintf(`%s: Version %s

Copyright 2024 The Simple Dev

Author:         Steven Stanton
License:        MIT - No Warranty
Author Github:  https//github.com/StevenDStanton
Project Github: https://github.com/StevemStanton/cli-tools-for-windows

Part of my CLI Tools for Windows project.`, tool, version)
}
