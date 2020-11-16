// Copyright 2020 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package resilience exposes an http.Handler that returns an HTTP failure
// randomly to improve client resiliency.
package resilience

import (
	"fmt"
	"net/http"
)

// ShouldFail returns non-zero when the current request should fail with the
// specified http status code.
//
// It is called first with afterHeader set as false. If this doesn't return
// true, it is called a second time in WriteHeader() with afterHeader set to
// true.
//
// The return value must be either 0 or between 400 and 599.
type ShouldFail func(r *http.Request, afterHeader bool) int

// Handle wraps a http.Handler to return random errors (generally 500) on the
// desired probability.
type Handler struct {
	Handler    http.Handler
	ShouldFail ShouldFail
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Random early exit.
	if s := h.ShouldFail(r, false); s != 0 {
		if s < 400 || s >= 600 {
			panic(fmt.Sprintf("unexpected status code %d", s))
		}
		w.WriteHeader(s)
		return
	}
	h.Handler.ServeHTTP(&responseWriter{w, r, h.ShouldFail, 0}, r)
}

type responseWriter struct {
	http.ResponseWriter
	req        *http.Request
	shouldFail ShouldFail
	status     int
}

func (r *responseWriter) Write(data []byte) (size int, err error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(data)
}

func (r *responseWriter) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	// Random late fail.
	if s := r.shouldFail(r.req, true); s != 0 {
		if s < 400 || s >= 600 {
			panic(fmt.Sprintf("unexpected status code %d", s))
		}
		status = s
	}
	r.ResponseWriter.WriteHeader(status)
	r.status = status
}
