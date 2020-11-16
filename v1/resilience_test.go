// Copyright 2020 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package resilience_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maruel/resilience/v1"
)

func ExampleHandler() {
	// Serves the current directory over HTTP and fails occasionally.
	s := &http.Server{
		Addr: ":6060",
		Handler: &resilience.Handler{
			Handler: http.FileServer(http.Dir(".")),
			ShouldFail: func(r *http.Request, afterHeader bool) int {
				return 500
			},
		},
	}
	log.Fatal(s.ListenAndServe())
}

func TestServeHTTP_early(t *testing.T) {
	req := httptest.NewRequest("GET", "/foo", &bytes.Buffer{})
	h := resilience.Handler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "hello")
		}),
		ShouldFail: func(r *http.Request, afterHeader bool) int {
			if !afterHeader {
				return 0
			}
			return 500
		},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	b, _ := ioutil.ReadAll(res.Body)
	if expected := "hello"; expected != string(b) {
		t.Fatalf("%q != %q", string(b), expected)
	}
	if res.StatusCode != 500 {
		t.Fatal(res.Status)
	}
}

func TestServeHTTP_late(t *testing.T) {
	req := httptest.NewRequest("GET", "/foo", &bytes.Buffer{})
	h := resilience.Handler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "hello")
		}),
		ShouldFail: func(r *http.Request, afterHeader bool) int {
			if afterHeader {
				return 0
			}
			return 500
		},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	b, _ := ioutil.ReadAll(res.Body)
	if expected := ""; expected != string(b) {
		t.Fatalf("%q != %q", string(b), expected)
	}
	if res.StatusCode != 500 {
		t.Fatal(res.Status)
	}
}

func TestServeHTTP_double_WriteHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/foo", &bytes.Buffer{})
	h := resilience.Handler{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(201)
			w.WriteHeader(202)
			io.WriteString(w, "hello")
		}),
		ShouldFail: func(r *http.Request, afterHeader bool) int {
			return 0
		},
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	b, _ := ioutil.ReadAll(res.Body)
	if expected := "hello"; expected != string(b) {
		t.Fatalf("%q != %q", string(b), expected)
	}
	if res.StatusCode != 201 {
		t.Fatal(res.Status)
	}
}

func TestServeHTTP_panic_early(t *testing.T) {
	req := httptest.NewRequest("GET", "/foo", &bytes.Buffer{})
	d := dummy{}
	h := resilience.Handler{
		Handler: &d,
		ShouldFail: func(r *http.Request, afterHeader bool) int {
			if afterHeader {
				return 0
			}
			return 200
		},
	}
	w := httptest.NewRecorder()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic")
		}
		if d.called {
			t.Error("should not have been called")
		}
	}()
	h.ServeHTTP(w, req)
}

func TestServeHTTP_panic_late(t *testing.T) {
	req := httptest.NewRequest("GET", "/foo", &bytes.Buffer{})
	d := dummy{}
	h := resilience.Handler{
		Handler: &d,
		ShouldFail: func(r *http.Request, afterHeader bool) int {
			if !afterHeader {
				return 0
			}
			return 200
		},
	}
	w := httptest.NewRecorder()
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected a panic")
		}
		if !d.called {
			t.Error("should have been called")
		}
	}()
	h.ServeHTTP(w, req)
}

type dummy struct {
	called bool
}

func (d *dummy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.called = true
	io.WriteString(w, "hello")
}
