package concurrency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPool(t *testing.T) {
	type args struct {
		size int
	}
	tests := []struct {
		want *Pool
		name string
		args args
	}{
		{
			name: "TestNewPool",
			args: args{
				size: 0,
			},
			want: &Pool{
				jobs:  0,
				queue: make(chan struct{}, 0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPool(tt.args.size)
			assert.Equal(t, got.jobs, tt.want.jobs)
			assert.Equal(t, cap(got.queue), cap(tt.want.queue))
		})
	}
}

func TestPool_Enqueue(t *testing.T) {
	type fields struct {
		queue chan struct{}
		jobs  int
	}
	type args struct {
		job    func(params ...any)
		params []any
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestPool_Enqueue",
			fields: fields{
				queue: make(chan struct{}, 0),
				jobs:  0,
			},
			args: args{
				job: func(params ...any) {
					return
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pool{
				queue: tt.fields.queue,
				jobs:  tt.fields.jobs,
			}
			p.Enqueue(tt.args.job, tt.args.params...)
		})
	}
}
