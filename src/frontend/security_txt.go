// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// securityTxtExpires is the RFC 3339 formatted Expires value for the
// security.txt response, computed once at startup, ~1 year out.
var securityTxtExpires = time.Now().AddDate(1, 0, 0).Format(time.RFC3339)

// securityTxtHandler serves the RFC 9116 security.txt file.
// TODO(P3-05): write the real Contact/Expires body.
func securityTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// registerSecurityTxt registers the security.txt endpoint on r, rooted at base.
func registerSecurityTxt(r *mux.Router, base string) {
	r.HandleFunc(base+"/.well-known/security.txt", securityTxtHandler).
		Methods(http.MethodGet, http.MethodHead)
}
