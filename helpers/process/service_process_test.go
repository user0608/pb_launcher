package process_test

import (
	"bytes"
	"pb_launcher/helpers/process"
	"testing"
	"time"
)

func TestProcess_StartAndStop(t *testing.T) {
	errChan := make(chan process.ProcessErrorMessage, 1)
	service := process.New("test-service", "sleep", []string{"2"}, process.WithErrorChan(errChan))
	if service.PID() != 0 {
		t.Fatal("stopped process has a PID")
	}

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Fatalf("service should be running")
	}
	if service.PID() <= 0 {
		t.Fatal("running process has no PID")
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}

	if service.IsRunning() {
		t.Fatalf("service should not be running")
	}
	if service.PID() != 0 {
		t.Fatal("stopped process still has a PID")
	}
}

func TestProcess_MultipleStartStop(t *testing.T) {
	service := process.New("test-service", "sleep", []string{"2"})

	for i := range 5 {
		if err := service.Start(); err != nil {
			t.Fatalf("failed to start service (iteration %d): %v", i, err)
		}

		if !service.IsRunning() {
			t.Fatalf("service should be running (iteration %d)", i)
		}

		if err := service.Stop(); err != nil {
			t.Fatalf("failed to stop service (iteration %d): %v", i, err)
		}

		if service.IsRunning() {
			t.Fatalf("service should not be running after stop (iteration %d)", i)
		}
	}
}

func TestProcess_AggressiveStartStop(t *testing.T) {
	errChan := make(chan process.ProcessErrorMessage, 1)
	service := process.New("test-service", "sleep", []string{"5"}, process.WithErrorChan(errChan))

	for i := range 5 {
		if err := service.Start(); err != nil {
			t.Fatalf("failed to start service (iteration %d): %v", i, err)
		}

		if !service.IsRunning() {
			t.Fatalf("service should be running (iteration %d)", i)
		}

		if err := service.Stop(); err != nil {
			t.Fatalf("failed to stop service (iteration %d): %v", i, err)
		}

		if service.IsRunning() {
			t.Fatalf("service should not be running after stop (iteration %d)", i)
		}
	}

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start service at final phase: %v", err)
	}

	if !service.IsRunning() {
		t.Fatalf("service should be running at final phase")
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("failed to stop service at final phase: %v", err)
	}

	if service.IsRunning() {
		t.Fatalf("service should not be running after final stop")
	}

	select {
	case err := <-errChan:
		t.Fatalf("unexpected error received after stop: %v", err)
	default:
	}
}

func TestProcess_ErrorChannel(t *testing.T) {
	errChan := make(chan process.ProcessErrorMessage, 1)
	service := process.New("test-service", "false", nil, process.WithErrorChan(errChan))

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start error-prone service: %v", err)
	}

	select {
	case errMsg := <-errChan:
		if errMsg.Error == nil {
			t.Fatalf("expected error, got nil")
		}
		t.Logf("received expected error: %v", errMsg.Error)
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for error message")
	}

	if service.IsRunning() {
		t.Fatalf("service should not be running after error")
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("failed to stop service after error: %v", err)
	}
}

func TestProcess_StartDoesNotRestart(t *testing.T) {
	service := process.New("test-service", "sleep", []string{"2"})

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	if !service.IsRunning() {
		t.Fatalf("service should be running")
	}

	if err := service.Start(); err != nil {
		t.Fatalf("start called on running service should return nil, got: %v", err)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("failed to stop service: %v", err)
	}
}

func TestProcess_StopWhenAlreadyStopped(t *testing.T) {
	service := process.New("test-service", "sleep", []string{"1"})

	if err := service.Stop(); err != nil {
		t.Fatalf("stop on already stopped service should return nil, got: %v", err)
	}
}

func TestProcess_StdoutAndStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	service := process.New("test-service", "echo", []string{"hello world"},
		process.WithStdout(&stdout), process.WithStderr(&stderr))

	if err := service.Start(); err != nil {
		t.Fatalf("failed to start service: %v", err)
	}

	// Waiting for Stopped synchronizes with cmd.Wait and its output writers;
	// sleeping alone does not make bytes.Buffer safe for concurrent reads.
	deadline := time.Now().Add(5 * time.Second)
	for service.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if service.IsRunning() {
		t.Fatal("echo did not finish")
	}

	if got := stdout.String(); got != "hello world\n" {
		t.Fatalf("unexpected stdout, got: %q", got)
	}

	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr output, got: %q", stderr.String())
	}
}
