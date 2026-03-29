package store

import "errors"

var ErrTaskConflict = errors.New("task conflict: idempotency key reused with different payload")
