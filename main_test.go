package main

import (
	"testing"
)

func TestGetPortPathTypical(t *testing.T) {
	port, path, err := getPortPath("8080/metrics")
	if port != "8080" {
		t.Errorf("Expected 8080, got %v", port)
	}
	if path != "/metrics" {
		t.Errorf("Expected /metrics, got %v", path)
	}
	if err != nil {
		t.Errorf("Expected a nil error, got %v", err)
	}
}

func TestGetPortPathNonStandardPath(t *testing.T) {
	port, path, err := getPortPath("8080/my.metrics")
	if port != "8080" {
		t.Errorf("Expected 8080, got %v", port)
	}
	if path != "/my.metrics" {
		t.Errorf("Expected /my.metrics, got %v", path)
	}
	if err != nil {
		t.Errorf("Expected a nil error, got %v", err)
	}
}

func TestGetPortPathWithSubPaths(t *testing.T) {
	port, path, err := getPortPath("8080/my/fancy/metrics")
	if port != "8080" {
		t.Errorf("Expected 8080, got %v", port)
	}
	if path != "/my/fancy/metrics" {
		t.Errorf("Expected /my/fancy/metrics, got %v", path)
	}
	if err != nil {
		t.Errorf("Expected a nil error, got %v", err)
	}
}

func TestGetPortPathWithOutPath(t *testing.T) {
	port, path, err := getPortPath("8080")
	if port != "8080" {
		t.Errorf("Expected 8080, got %v", port)
	}
	if path != "" {
		t.Errorf("Expected no path, got %v", path)
	}
	if err != nil {
		t.Errorf("Expected a nil error, got %v", err)
	}
}
