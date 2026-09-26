// Package mhurl parses URLs support multiple hosts.
//
// Copy and modify from https://github.com/golang/go/blob/go1.26.1/src/net/url/url.go
// The reason is that the origin `Parse()` func does not support multiple hosts in the Host part after Go 1.26.
// See the original issue for more details:
//
//	https://github.com/golang/go/issues/75859
//	https://github.com/golang/go/issues/78077
package mhurl

import (
	"errors"
	"fmt"
	"net/netip"
	neturl "net/url"
	"strings"
)

const upperhex = "0123456789ABCDEF"

func ishex(c byte) bool {
	return table[c]&hexChar != 0
}

// Precondition: ishex(c) is true.
func unhex(c byte) byte {
	return 9*(c>>6) + (c & 15)
}

// See the reference implementation in gen_encoding_table.go.
func shouldEscape(c byte, mode encoding) bool {
	return table[c]&mode == 0
}

// unescape unescapes a string; the mode specifies
// which section of the URL string is being unescaped.
func unescape(s string, mode encoding) (string, error) {
	// Count %, check that they're well-formed.
	n := 0
	hasPlus := false
	for i := 0; i < len(s); {
		switch s[i] {
		case '%':
			n++
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				s = s[i:]
				if len(s) > 3 {
					s = s[:3]
				}
				return "", neturl.EscapeError(s)
			}
			// Per https://tools.ietf.org/html/rfc3986#page-21
			// in the host component %-encoding can only be used
			// for non-ASCII bytes.
			// But https://tools.ietf.org/html/rfc6874#section-2
			// introduces %25 being allowed to escape a percent sign
			// in IPv6 scoped-address literals. Yay.
			if mode == encodeHost && unhex(s[i+1]) < 8 && s[i:i+3] != "%25" {
				return "", neturl.EscapeError(s[i : i+3])
			}
			if mode == encodeZone {
				// RFC 6874 says basically "anything goes" for zone identifiers
				// and that even non-ASCII can be redundantly escaped,
				// but it seems prudent to restrict %-escaped bytes here to those
				// that are valid host name bytes in their unescaped form.
				// That is, you can use escaping in the zone identifier but not
				// to introduce bytes you couldn't just write directly.
				// But Windows puts spaces here! Yay.
				v := unhex(s[i+1])<<4 | unhex(s[i+2])
				if s[i:i+3] != "%25" && v != ' ' && shouldEscape(v, encodeHost) {
					return "", neturl.EscapeError(s[i : i+3])
				}
			}
			i += 3
		case '+':
			hasPlus = mode == encodeQueryComponent
			i++
		default:
			if (mode == encodeHost || mode == encodeZone) && s[i] < 0x80 && shouldEscape(s[i], mode) {
				return "", neturl.InvalidHostError(s[i : i+1])
			}
			i++
		}
	}

	if n == 0 && !hasPlus {
		return s, nil
	}

	var unescapedPlusSign byte
	switch mode {
	case encodeQueryComponent:
		unescapedPlusSign = ' '
	default:
		unescapedPlusSign = '+'
	}
	var t strings.Builder
	t.Grow(len(s) - 2*n)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '%':
			// In the loop above, we established that unhex's precondition is
			// fulfilled for both s[i+1] and s[i+2].
			t.WriteByte(unhex(s[i+1])<<4 | unhex(s[i+2]))
			i += 2
		case '+':
			t.WriteByte(unescapedPlusSign)
		default:
			t.WriteByte(s[i])
		}
	}
	return t.String(), nil
}

func escape(s string, mode encoding) string {
	spaceCount, hexCount := 0, 0
	for _, c := range []byte(s) {
		if shouldEscape(c, mode) {
			if c == ' ' && mode == encodeQueryComponent {
				spaceCount++
			} else {
				hexCount++
			}
		}
	}

	if spaceCount == 0 && hexCount == 0 {
		return s
	}

	var buf [64]byte
	var t []byte

	required := len(s) + 2*hexCount
	if required <= len(buf) {
		t = buf[:required]
	} else {
		t = make([]byte, required)
	}

	if hexCount == 0 {
		copy(t, s)
		for i := 0; i < len(s); i++ {
			if s[i] == ' ' {
				t[i] = '+'
			}
		}
		return string(t)
	}

	j := 0
	for _, c := range []byte(s) {
		switch {
		case c == ' ' && mode == encodeQueryComponent:
			t[j] = '+'
			j++
		case shouldEscape(c, mode):
			t[j] = '%'
			t[j+1] = upperhex[c>>4]
			t[j+2] = upperhex[c&15]
			j += 3
		default:
			t[j] = c
			j++
		}
	}
	return string(t)
}

// Maybe rawURL is of the form scheme:path.
// (Scheme must be [a-zA-Z][a-zA-Z0-9+.-]*)
// If so, return scheme, path; else return "", rawURL.
func getScheme(rawURL string) (scheme, path string, err error) {
	for i := 0; i < len(rawURL); i++ {
		c := rawURL[i]
		switch {
		case 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z':
		// do nothing
		case '0' <= c && c <= '9' || c == '+' || c == '-' || c == '.':
			if i == 0 {
				return "", rawURL, nil
			}
		case c == ':':
			if i == 0 {
				return "", "", errors.New("missing protocol scheme")
			}
			return rawURL[:i], rawURL[i+1:], nil
		default:
			// we have encountered an invalid character,
			// so there is no valid scheme
			return "", rawURL, nil
		}
	}
	return "", rawURL, nil
}

// Parse parses a raw url into a [URL] structure.
//
// The url may be relative (a path, without a host) or absolute
// (starting with a scheme). Trying to parse a hostname and path
// without a scheme is invalid but may not necessarily return an
// error, due to parsing ambiguities.
func Parse(rawURL string) (*neturl.URL, error) {
	// Cut off #frag
	u, frag, _ := strings.Cut(rawURL, "#")
	url, err := parse(u, false)
	if err != nil {
		return nil, &neturl.Error{"parse", u, err}
	}
	if frag == "" {
		return url, nil
	}
	if err = setFragment(url, frag); err != nil {
		return nil, &neturl.Error{"parse", rawURL, err}
	}
	return url, nil
}

// parse parses a URL from a string in one of two contexts. If
// viaRequest is true, the URL is assumed to have arrived via an HTTP request,
// in which case only absolute URLs or path-absolute relative URLs are allowed.
// If viaRequest is false, all forms of relative URLs are allowed.
func parse(rawURL string, viaRequest bool) (*neturl.URL, error) {
	var rest string
	var err error

	if stringContainsCTLByte(rawURL) {
		return nil, errors.New("net/url: invalid control character in URL")
	}

	if rawURL == "" && viaRequest {
		return nil, errors.New("empty url")
	}
	url := new(neturl.URL)

	if rawURL == "*" {
		url.Path = "*"
		return url, nil
	}

	// Split off possible leading "http:", "mailto:", etc.
	// Cannot contain escaped characters.
	if url.Scheme, rest, err = getScheme(rawURL); err != nil {
		return nil, err
	}
	url.Scheme = strings.ToLower(url.Scheme)

	if strings.HasSuffix(rest, "?") && strings.Count(rest, "?") == 1 {
		url.ForceQuery = true
		rest = rest[:len(rest)-1]
	} else {
		rest, url.RawQuery, _ = strings.Cut(rest, "?")
	}

	if !strings.HasPrefix(rest, "/") {
		if url.Scheme != "" {
			// We consider rootless paths per RFC 3986 as opaque.
			url.Opaque = rest
			return url, nil
		}
		if viaRequest {
			return nil, errors.New("invalid URI for request")
		}

		// Avoid confusion with malformed schemes, like cache_object:foo/bar.
		// See golang.org/issue/16822.
		//
		// RFC 3986, §3.3:
		// In addition, a URI reference (Section 4.1) may be a relative-path reference,
		// in which case the first path segment cannot contain a colon (":") character.
		if segment, _, _ := strings.Cut(rest, "/"); strings.Contains(segment, ":") {
			// First path segment has colon. Not allowed in relative URL.
			return nil, errors.New("first path segment in URL cannot contain colon")
		}
	}

	if (url.Scheme != "" || !viaRequest && !strings.HasPrefix(rest, "///")) && strings.HasPrefix(rest, "//") {
		var authority string
		authority, rest = rest[2:], ""
		if i := strings.Index(authority, "/"); i >= 0 {
			authority, rest = authority[:i], authority[i:]
		}
		url.User, url.Host, err = parseAuthority(url.Scheme, authority)
		if err != nil {
			return nil, err
		}
	} else if url.Scheme != "" && strings.HasPrefix(rest, "/") {
		// OmitHost is set to true when rawURL has an empty host (authority).
		// See golang.org/issue/46059.
		url.OmitHost = true
	}

	// Set Path and, optionally, RawPath.
	// RawPath is a hint of the encoding of Path. We don't want to set it if
	// the default escaping of Path is equivalent, to help make sure that people
	// don't rely on it in general.
	if err := setPath(url, rest); err != nil {
		return nil, err
	}
	return url, nil
}

func parseAuthority(scheme, authority string) (user *neturl.Userinfo, host string, err error) {
	i := strings.LastIndex(authority, "@")
	if i < 0 {
		host, err = parseHost(scheme, authority)
	} else {
		host, err = parseHost(scheme, authority[i+1:])
	}
	if err != nil {
		return nil, "", err
	}
	if i < 0 {
		return nil, host, nil
	}
	userinfo := authority[:i]
	if !validUserinfo(userinfo) {
		return nil, "", errors.New("net/url: invalid userinfo")
	}
	if !strings.Contains(userinfo, ":") {
		if userinfo, err = unescape(userinfo, encodeUserPassword); err != nil {
			return nil, "", err
		}
		user = neturl.User(userinfo)
	} else {
		username, password, _ := strings.Cut(userinfo, ":")
		if username, err = unescape(username, encodeUserPassword); err != nil {
			return nil, "", err
		}
		if password, err = unescape(password, encodeUserPassword); err != nil {
			return nil, "", err
		}
		user = neturl.UserPassword(username, password)
	}
	return user, host, nil
}

// parseHost parses host as an authority without user
// information. That is, as host[:port].
func parseHost(scheme, host string) (string, error) {
	var hosts []string
	for _, h := range strings.Split(host, ",") {
		result, err := _parseHost(scheme, h)
		if err != nil {
			return "", err
		}
		hosts = append(hosts, result)
	}
	return strings.Join(hosts, ","), nil
}

func _parseHost(scheme, host string) (string, error) {
	if openBracketIdx := strings.LastIndex(host, "["); openBracketIdx > 0 {
		return "", errors.New("invalid IP-literal")
	} else if openBracketIdx == 0 {
		// Parse an IP-Literal in RFC 3986 and RFC 6874.
		// E.g., "[fe80::1]", "[fe80::1%25en0]", "[fe80::1]:80".
		closeBracketIdx := strings.LastIndex(host, "]")
		if closeBracketIdx < 0 {
			return "", errors.New("missing ']' in host")
		}

		colonPort := host[closeBracketIdx+1:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort)
		}
		unescapedColonPort, err := unescape(colonPort, encodeHost)
		if err != nil {
			return "", err
		}

		hostname := host[openBracketIdx+1 : closeBracketIdx]
		var unescapedHostname string
		// RFC 6874 defines that %25 (%-encoded percent) introduces
		// the zone identifier, and the zone identifier can use basically
		// any %-encoding it likes. That's different from the host, which
		// can only %-encode non-ASCII bytes.
		// We do impose some restrictions on the zone, to avoid stupidity
		// like newlines.
		zoneIdx := strings.Index(hostname, "%25")
		if zoneIdx >= 0 {
			hostPart, err := unescape(hostname[:zoneIdx], encodeHost)
			if err != nil {
				return "", err
			}
			zonePart, err := unescape(hostname[zoneIdx:], encodeZone)
			if err != nil {
				return "", err
			}
			unescapedHostname = hostPart + zonePart
		} else {
			var err error
			unescapedHostname, err = unescape(hostname, encodeHost)
			if err != nil {
				return "", err
			}
		}

		// Per RFC 3986, only a host identified by a valid
		// IPv6 address can be enclosed by square brackets.
		// This excludes any IPv4, but notably not IPv4-mapped addresses.
		addr, err := netip.ParseAddr(unescapedHostname)
		if err != nil {
			return "", fmt.Errorf("invalid host: %w", err)
		}
		if addr.Is4() {
			return "", errors.New("invalid IP-literal")
		}
		return "[" + unescapedHostname + "]" + unescapedColonPort, nil
	} else if i := strings.Index(host, ":"); i != -1 {
		colonPort := host[i:]
		if !validOptionalPort(colonPort) {
			return "", fmt.Errorf("invalid port %q after host", colonPort)
		}
	}

	var err error
	if host, err = unescape(host, encodeHost); err != nil {
		return "", err
	}
	return host, nil
}

// setPath sets the Path and RawPath fields of the URL based on the provided
// escaped path p. It maintains the invariant that RawPath is only specified
// when it differs from the default encoding of the path.
// For example:
// - setPath("/foo/bar")   will set Path="/foo/bar" and RawPath=""
// - setPath("/foo%2fbar") will set Path="/foo/bar" and RawPath="/foo%2fbar"
// setPath will return an error only if the provided path contains an invalid
// escaping.
func setPath(u *neturl.URL, p string) error {
	path, err := unescape(p, encodePath)
	if err != nil {
		return err
	}
	u.Path = path
	if escp := escape(path, encodePath); p == escp {
		// Default encoding is fine.
		u.RawPath = ""
	} else {
		u.RawPath = p
	}
	return nil
}

// setFragment is like setPath but for Fragment/RawFragment.
func setFragment(u *neturl.URL, f string) error {
	frag, err := unescape(f, encodeFragment)
	if err != nil {
		return err
	}
	u.Fragment = frag
	if escf := escape(frag, encodeFragment); f == escf {
		// Default encoding is fine.
		u.RawFragment = ""
	} else {
		u.RawFragment = f
	}
	return nil
}

// validOptionalPort reports whether port is either an empty string
// or matches /^:\d*$/
func validOptionalPort(port string) bool {
	if port == "" {
		return true
	}
	if port[0] != ':' {
		return false
	}
	for _, b := range port[1:] {
		if b < '0' || b > '9' {
			return false
		}
	}
	return true
}

// validUserinfo reports whether s is a valid userinfo string per RFC 3986
// Section 3.2.1:
//
//	userinfo    = *( unreserved / pct-encoded / sub-delims / ":" )
//	unreserved  = ALPHA / DIGIT / "-" / "." / "_" / "~"
//	sub-delims  = "!" / "$" / "&" / "'" / "(" / ")"
//	              / "*" / "+" / "," / ";" / "="
//
// It doesn't validate pct-encoded. The caller does that via func unescape.
func validUserinfo(s string) bool {
	for _, r := range s {
		if 'A' <= r && r <= 'Z' {
			continue
		}
		if 'a' <= r && r <= 'z' {
			continue
		}
		if '0' <= r && r <= '9' {
			continue
		}
		switch r {
		case '-', '.', '_', ':', '~', '!', '$', '&', '\'',
			'(', ')', '*', '+', ',', ';', '=', '%':
			continue
		case '@':
			// `RFC 3986 section 3.2.1` does not allow '@' in userinfo.
			// It is a delimiter between userinfo and host.
			// However, URLs are diverse, and in some cases,
			// the userinfo may contain an '@' character,
			// for example, in "http://username:p@ssword@google.com",
			// the string "username:p@ssword" should be treated as valid userinfo.
			// Ref:
			//   https://go.dev/issue/3439
			//   https://go.dev/issue/22655
			continue
		default:
			return false
		}
	}
	return true
}

// stringContainsCTLByte reports whether s contains any ASCII control character.
func stringContainsCTLByte(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < ' ' || b == 0x7f {
			return true
		}
	}
	return false
}

type encoding uint8

const (
	encodePath encoding = 1 << iota
	encodePathSegment
	encodeHost
	encodeZone
	encodeUserPassword
	encodeQueryComponent
	encodeFragment

	// hexChar is actually NOT an encoding mode, but there are only seven
	// encoding modes. We might as well abuse the otherwise unused most
	// significant bit in uint8 to indicate whether a character is
	// hexadecimal.
	hexChar
)

var table = [256]encoding{
	'!':  encodeFragment | encodeZone | encodeHost,
	'"':  encodeZone | encodeHost,
	'$':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'&':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'\'': encodeZone | encodeHost,
	'(':  encodeFragment | encodeZone | encodeHost,
	')':  encodeFragment | encodeZone | encodeHost,
	'*':  encodeFragment | encodeZone | encodeHost,
	'+':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	',':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePath,
	'-':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'.':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'/':  encodeFragment | encodePath,
	'0':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'1':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'2':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'3':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'4':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'5':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'6':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'7':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'8':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'9':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	':':  encodeFragment | encodeZone | encodeHost | encodePathSegment | encodePath,
	';':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePath,
	'<':  encodeZone | encodeHost,
	'=':  encodeFragment | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'>':  encodeZone | encodeHost,
	'?':  encodeFragment,
	'@':  encodeFragment | encodePathSegment | encodePath,
	'A':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'B':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'C':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'D':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'E':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'F':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'G':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'H':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'I':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'J':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'K':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'L':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'M':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'N':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'O':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'P':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'Q':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'R':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'S':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'T':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'U':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'V':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'W':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'X':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'Y':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'Z':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'[':  encodeZone | encodeHost,
	']':  encodeZone | encodeHost,
	'_':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'a':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'b':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'c':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'd':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'e':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'f':  hexChar | encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'g':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'h':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'i':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'j':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'k':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'l':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'm':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'n':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'o':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'p':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'q':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'r':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	's':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	't':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'u':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'v':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'w':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'x':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'y':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'z':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
	'~':  encodeFragment | encodeQueryComponent | encodeUserPassword | encodeZone | encodeHost | encodePathSegment | encodePath,
}
