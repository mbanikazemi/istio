// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
)

// The test may run on a system with localhost = 127.0.0.1 or ::1, so we
// determine that value and use it in the "expected" results for the test
// cases in TestResolveAddr(). Need to wrap IPv6 addresses in square
// brackets.
func determineLocalHostIPString(t *testing.T) string {
	ips, err := net.LookupIP("localhost")
	if err != nil || len(ips) == 0 {
		t.Fatalf("Test setup failure - unable to determine IP of localhost: %v", err)
	}
	ip := ips[0]
	if ip.To4() == nil {
		return fmt.Sprintf("[%s]", ip.String())
	}
	return ip.String()
}

func MockLookupIPAddr(ctx context.Context, addr string) ([]net.IPAddr, error) {
	var ret = []net.IPAddr{
		{IP: net.ParseIP("2001:db8::68")},
		{IP: net.IPv4(1, 2, 3, 4)},
		{IP: net.IPv4(1, 2, 3, 5)},
	}
	return ret, nil
}

func MockLookupIPAddrIPv6(ctx context.Context, addr string) ([]net.IPAddr, error) {
	var ret = []net.IPAddr{
		{IP: net.ParseIP("2001:db8::68")},
	}
	return ret, nil
}
func TestResolveAddr(t *testing.T) {
	localIP := determineLocalHostIPString(t)

	var testCases = []struct {
		name     string
		input    string
		expected string
		errStr   string
		lookup   func(ctx context.Context, addr string) ([]net.IPAddr, error)
	}{
		{
			name:     "Host by name",
			input:    "localhost:9080",
			expected: fmt.Sprintf("%s:9080", localIP),
			errStr:   "",
			lookup:   nil,
		},
		{
			name:     "Host by name w/brackets",
			input:    "[localhost]:9080",
			expected: fmt.Sprintf("%s:9080", localIP),
			errStr:   "",
			lookup:   nil,
		},
		{
			name:     "Host by IPv4",
			input:    "127.0.0.1:9080",
			expected: "127.0.0.1:9080",
			errStr:   "",
			lookup:   nil,
		},
		{
			name:     "Host by IPv6",
			input:    "[::1]:9080",
			expected: "[::1]:9080",
			errStr:   "",
			lookup:   nil,
		},
		{
			name:     "Bad IPv4",
			input:    "127.0.0.1.1:9080",
			expected: "",
			errStr:   "lookup failed for IP address: lookup 127.0.0.1.1: no such host",
			lookup:   nil,
		},
		{
			name:     "Bad IPv6",
			input:    "[2001:db8::bad::1]:9080",
			expected: "",
			errStr:   "lookup failed for IP address: lookup 2001:db8::bad::1: no such host",
			lookup:   nil,
		},
		{
			name:     "Empty host",
			input:    "",
			expected: "",
			errStr:   ErrResolveNoAddress.Error(),
			lookup:   nil,
		},
		{
			name:     "IPv6 missing brackets",
			input:    "2001:db8::20:9080",
			expected: "",
			errStr:   "address 2001:db8::20:9080: too many colons in address",
			lookup:   nil,
		},
		{
			name:     "Colon, but no port",
			input:    "localhost:",
			expected: fmt.Sprintf("%s:", localIP),
			errStr:   "",
			lookup:   nil,
		},
		{
			name:     "Missing port",
			input:    "localhost",
			expected: "",
			errStr:   "address localhost: missing port in address",
			lookup:   nil,
		},
		{
			name:     "Missing host",
			input:    ":9080",
			expected: "",
			errStr:   "lookup failed for IP address: lookup : no such host",
			lookup:   nil,
		},
		{
			name:     "Host by name - non local",
			input:    "www.foo.com:9080",
			expected: "1.2.3.4:9080",
			errStr:   "",
			lookup:   MockLookupIPAddr,
		},
		{
			name:     "Host by name - non local 0 IPv6 only address",
			input:    "www.foo.com:9080",
			expected: "[2001:db8::68]:9080",
			errStr:   "",
			lookup:   MockLookupIPAddrIPv6,
		},
	}

	for _, tc := range testCases {
		actual, err := ResolveAddr(tc.input, tc.lookup)
		if err != nil {
			if tc.errStr == "" {
				t.Errorf("[%s] expected success, but saw error: %v", tc.name, err)
			} else if err.Error() != tc.errStr {
				if strings.Contains(err.Error(), "Temporary failure in name resolution") {
					t.Logf("[%s] expected error %q, got %q", tc.name, tc.errStr, err.Error())
					continue
				}
				t.Errorf("[%s] expected error %q, got %q", tc.name, tc.errStr, err.Error())
			}
		} else {
			if tc.errStr != "" {
				t.Errorf("[%s] no error seen, but expected failure: %s", tc.name, tc.errStr)
			} else if actual != tc.expected {
				t.Errorf("[%s] expected address %q, got %q", tc.name, tc.expected, actual)
			}
		}
	}
}
