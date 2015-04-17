package main

import (
	"testing"
)

func TestLabelsToMap(t *testing.T) {
	m := labelsToMap(`{foo="bar",boo="bear"}`)
	if m["foo"] != "bar" {
		t.Errorf("Expected bar, got %v", m["foo"])
	}
	if m["boo"] != "bear" {
		t.Errorf("Expected bear, got %v", m["boo"])
	}
	if len(m) != 2 {
		t.Errorf("Expected 2, got %d", len(m))
	}
}

func TestLabelsToString(t *testing.T) {
	m := make(map[string]string)
	m["foo"] = "bar"
	m["boo"] = "bear"
	m["bla"] = "boo"
	labels := labelsToString(m)
	expected := `{bla="boo",boo="bear",foo="bar"}`
	if labels != expected {
		t.Errorf("Expected %v, got %v", expected, labels)
	}

}
