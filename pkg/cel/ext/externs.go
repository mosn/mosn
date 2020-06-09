/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ext

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
)

// externIP creates an IP address
func externIP(in string) ([]byte, error) {
	if ip := net.ParseIP(in); ip != nil {
		return ip, nil
	}
	return []byte{}, fmt.Errorf("could not convert %s to IP_ADDRESS", in)
}

// externIPEqual compares two IP addresses for equality
func externIPEqual(a []byte, b []byte) bool {
	// net.IP is an alias for []byte, so these are safe to convert
	ip1 := net.IP(a)
	ip2 := net.IP(b)
	return ip1.Equal(ip2)
}

// This IDNA profile is for performing validations, but does not otherwise modify the string.
var externDNSNameProfile = idna.New(
	idna.StrictDomainName(true),
	idna.ValidateLabels(true),
	idna.VerifyDNSLength(true),
	idna.BidiRule())

// externDNSName converts a string to a DNS name
func externDNSName(in string) (string, error) {
	s, err := externDNSNameProfile.ToUnicode(in)
	if err != nil {
		return "", fmt.Errorf("error converting '%s' to dns name: '%v'", in, err)
	}
	return s, err
}

// This IDNA profile converts the string for lookup, which ends up canonicalizing the dns name, for the most
// part.
var externDNSNameEqualProfile = idna.New(
	idna.MapForLookup(),
	idna.BidiRule())

// externDNSNameEqual compares two DNS names for equality
func externDNSNameEqual(n1 string, n2 string) (bool, error) {
	var err error

	if n1, err = externDNSNameEqualProfile.ToUnicode(n1); err != nil {
		return false, err
	}

	if n2, err = externDNSNameEqualProfile.ToUnicode(n2); err != nil {
		return false, err
	}

	return n1 == n2, nil
}

// externEmail converts a string to an email address
func externEmail(in string) (string, error) {
	_, err := mail.ParseAddress(in)
	if err != nil {
		return "", fmt.Errorf("error converting '%s' to e-mail: '%v'", in, err)
	}
	return in, nil
}

// externEmailEqual compares two email addresses for equality
func externEmailEqual(e1 string, e2 string) (bool, error) {
	a1, err := mail.ParseAddress(e1)
	if err != nil {
		return false, err
	}

	a2, err := mail.ParseAddress(e2)
	if err != nil {
		return false, err
	}

	local1, domain1 := getEmailParts(a1.Address)
	local2, domain2 := getEmailParts(a2.Address)

	domainEq, err := externDNSNameEqual(domain1, domain2)
	if err != nil {
		return false, fmt.Errorf("error comparing e-mails '%s' and '%s': %v", e1, e2, err)
	}

	if !domainEq {
		return false, nil
	}

	return local1 == local2, nil
}

// externURI converts a string to a URI
func externURI(in string) (string, error) {
	if _, err := url.Parse(in); err != nil {
		return "", fmt.Errorf("error converting string to uri '%s': '%v'", in, err)
	}
	return in, nil
}

// externURIEqual compares two URIs for equality
func externURIEqual(u1 string, u2 string) (bool, error) {
	url1, err := url.Parse(u1)
	if err != nil {
		return false, fmt.Errorf("error converting string to uri '%s': '%v'", u1, err)
	}

	url2, err := url.Parse(u2)
	if err != nil {
		return false, fmt.Errorf("error converting string to uri '%s': '%v'", u2, err)
	}

	// Try to apply as much normalization logic as possible.
	scheme1 := strings.ToLower(url1.Scheme)
	scheme2 := strings.ToLower(url2.Scheme)
	if scheme1 != scheme2 {
		return false, nil
	}

	// normalize schemes
	url1.Scheme = scheme1
	url2.Scheme = scheme1

	if scheme1 == "http" || scheme1 == "https" {
		// Special case http(s) URLs

		dnsEq, err := externDNSNameEqual(url1.Hostname(), url2.Hostname())
		if err != nil {
			return false, err
		}

		if !dnsEq {
			return false, nil
		}

		if url1.Port() != url2.Port() {
			return false, nil
		}

		// normalize host names
		url1.Host = url2.Host
	}

	return url1.String() == url2.String(), nil
}

func getEmailParts(email string) (local string, domain string) {
	idx := strings.IndexByte(email, '@')
	if idx == -1 {
		local = email
		domain = ""
		return
	}

	local = email[:idx]
	domain = email[idx+1:]
	return
}

// externMatch provides wildcard matching for strings
func externMatch(str string, pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(str, pattern[:len(pattern)-1])
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(str, pattern[1:])
	}
	return str == pattern
}

func externReverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func externConditionalString(condition bool, trueStr, falseStr string) string {
	if condition {
		return trueStr
	}
	return falseStr
}
