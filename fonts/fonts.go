// fonts embed fonts into Go.
package fonts

import (
	_ "embed"
)

var (
	//go:embed PTS55F.ttf
	PTSansRegular []byte
)
