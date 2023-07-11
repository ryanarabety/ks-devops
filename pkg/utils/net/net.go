/*
Copyright 2019-2022 The KubeSphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package net

import (
	"net"
	"net/http"
	"strings"
)

// IsValidPort check if given number is an valid port
// 0 is considered as an invalid port
func IsValidPort(port int) bool {
	return port > 0 && port < 65535
}

// GetRequestIP returns the real ip from a HTTP request
func GetRequestIP(req *http.Request) string {
	address := strings.Trim(req.Header.Get("X-Real-Ip"), " ")
	if address != "" {
		return address
	}

	address = strings.Trim(req.Header.Get("X-Forwarded-For"), " ")
	if address != "" {
		return address
	}

	address, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}

	return address
}

// ParseURL makes sure return a valid URL string
func ParseURL(address string) string {
	address = strings.TrimSuffix(address, "/")
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return address
	}
	address = "https://" + address
	return address
}
