package fs

import (
	"sync"
)

// Directory type
type Directory struct {
	children *sync.Map
}
