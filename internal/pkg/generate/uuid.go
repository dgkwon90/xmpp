package generate

import (
	"strings"

	"github.com/google/uuid"
)

func UUID() string {
	return strings.Replace(uuid.New().String(), "-", "", -1)
}
