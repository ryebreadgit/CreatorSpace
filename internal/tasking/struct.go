package tasking

import (
	"sync"
	"time"
)

type Task struct {
	sync.RWMutex
	Name     string
	Epoch    int64
	Interval time.Duration
	Task     func(...interface{}) error
	Args     []interface{}
}
