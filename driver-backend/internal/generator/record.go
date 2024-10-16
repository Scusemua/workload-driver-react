package generator

import (
	"sync"
	"time"
)

type Record interface {
	GetTS() time.Time
}

type RecordProvider interface {
	Get() Record
	Recycle(Record)
}

type RecordPool struct {
	pool sync.Pool
}

func (p *RecordPool) Get() Record {
	rec, _ := p.pool.Get().(Record)
	return rec
}

func (p *RecordPool) Recycle(rec Record) {
	p.pool.Put(rec)
}
