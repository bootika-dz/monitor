package monitor

import (
	"context"
	"testing"
	"time"
)

type testsrv struct {
	exited bool
	done   chan struct{}
}

func (ts *testsrv) Done() <-chan struct{} {
	return ts.done
}

func (ts *testsrv) Start(ctx context.Context) {
	ts.done = make(chan struct{})
	go func() {
		<-ctx.Done()
		ts.exited = true
		close(ts.done)
	}()
}

func isDone(srv Service) bool {
	select {
	case <-srv.Done():
		return true
	default:
		return false
	}
}

func TestMonitor(t *testing.T) {
	ctx := context.Background()
	m, ctx := New(ctx)
	if isDone(m) {
		t.Fatal("new monitor is already done")
	}
	ctxA, cancelA := context.WithCancel(ctx)
	srvA := &testsrv{}
	srvA.Start(ctxA)
	m.Watch(ctx, "A", srvA)
	srvB := &testsrv{}
	srvB.Start(ctx)
	m.Watch(ctx, "B", srvB)
	srvC := &testsrv{}
	srvC.Start(ctx)
	m.Watch(ctx, "C", srvC)
	if isDone(m) {
		t.Fatal("started watching and monitor is already done")
	}

	// as we cancelled A all other services should exit and monitor should close done
	cancelA()
	select {
	case <-m.Done():
		t.Log("monitor is done")
	case <-time.After(5 * time.Second):
		t.Fatal("5 sec after cancelling A monitor is still not done")
	}
	if !(srvA.exited && srvB.exited && srvC.exited) {
		t.Fatal("not all services have exited")
	}

	// adding new services to done monitor shouldn't cause any issues
	srvD := &testsrv{}
	srvD.Start(ctx)
	m.Watch(ctx, "D", srvD)
	srvE := &testsrv{}
	srvE.Start(ctx)
	m.Watch(ctx, "E", srvE)
}

func TestCancel(t *testing.T) {
	ctx := context.Background()
	m, ctx := New(ctx)
	srv := &testsrv{}
	srv.Start(ctx)
	m.Watch(ctx, "Test", srv)
	m.Cancel()
	select {
	case <-m.Done():
		t.Log("monitor is done")
	case <-time.After(5 * time.Second):
		t.Fatal("5 sec after cancelling A monitor is still not done")
	}
	if !srv.exited {
		t.Fatal("service has not exited")
	}
}

// check that if we cancel before starting watching anything, monitor will be done
func TestNoServices(t *testing.T) {
	ctx := context.Background()
	m, _ := New(ctx)
	m.Cancel()
	select {
	case <-m.Done():
		t.Log("monitor is done")
	case <-time.After(5 * time.Second):
		t.Fatal("5 sec after cancelling A monitor is still not done")
	}
}
