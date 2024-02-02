package wsman

import (
	"crypto/md5" //nolint: gosec // we're constrained to MD5 by Intel AMT
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

type challenge struct {
	Username   string
	Password   string
	Realm      string
	CSRFToken  string
	Domain     string
	Nonce      string
	Opaque     string
	Stale      string
	Algorithm  string
	Qop        string
	Cnonce     string
	NonceCount int
}

func h(data string) string {
	hf := md5.New() //nolint: gosec // we're constrained to MD5 by Intel AMT
	_, _ = io.WriteString(hf, data)
	return fmt.Sprintf("%x", hf.Sum(nil))
}

func kd(secret, data string) string {
	return h(fmt.Sprintf("%s:%s", secret, data))
}

func (c *challenge) ha1() string {
	return h(fmt.Sprintf("%s:%s:%s", c.Username, c.Realm, c.Password))
}

func (c *challenge) ha2(method, uri string) string {
	return h(fmt.Sprintf("%s:%s", method, uri))
}

func (c *challenge) resp(method, uri, cnonce string) (string, error) {
	c.NonceCount++
	if c.Qop == "auth" {
		if cnonce != "" {
			c.Cnonce = cnonce
		} else {
			b := make([]byte, 8)
			if _, err := io.ReadFull(rand.Reader, b); err != nil {
				return "", err
			}
			c.Cnonce = fmt.Sprintf("%x", b)[:16]
		}
		return kd(c.ha1(), fmt.Sprintf("%s:%08x:%s:%s:%s",
			c.Nonce, c.NonceCount, c.Cnonce, c.Qop, c.ha2(method, uri))), nil
	} else if c.Qop == "" {
		return kd(c.ha1(), fmt.Sprintf("%s:%s", c.Nonce, c.ha2(method, uri))), nil
	}
	return "", fmt.Errorf("alg not implemented")
}

// source https://code.google.com/p/mlab-ns2/source/browse/gae/ns/digest/digest.go#178
func (c *challenge) authorize(method, uri string) (string, error) {
	// Note that this is only implemented for MD5 and NOT MD5-sess.
	// MD5-sess is rarely supported and those that do are a big mess.
	if c.Algorithm != "MD5" {
		return "", fmt.Errorf("alg not implemented")
	}
	// Note that this is NOT implemented for "qop=auth-int".  Similarly the
	// auth-int server side implementations that do exist are a mess.
	if c.Qop != "auth" && c.Qop != "" {
		return "", fmt.Errorf("alg not implemented")
	}
	resp, err := c.resp(method, uri, "")
	if err != nil {
		return "", fmt.Errorf("alg not implemented")
	}
	sl := []string{fmt.Sprintf(`username="%s"`, c.Username)}
	sl = append(sl, fmt.Sprintf(`realm="%s"`, c.Realm))
	sl = append(sl, fmt.Sprintf(`nonce="%s"`, c.Nonce))
	sl = append(sl, fmt.Sprintf(`uri="%s"`, uri))
	sl = append(sl, fmt.Sprintf(`response="%s"`, resp))
	if c.Algorithm != "" {
		sl = append(sl, fmt.Sprintf(`algorithm="%s"`, c.Algorithm))
	}
	if c.Opaque != "" {
		sl = append(sl, fmt.Sprintf(`opaque="%s"`, c.Opaque))
	}
	if c.Qop != "" {
		sl = append(sl, fmt.Sprintf("qop=%s", c.Qop))
		sl = append(sl, fmt.Sprintf("nc=%08x", c.NonceCount))
		sl = append(sl, fmt.Sprintf(`cnonce="%s"`, c.Cnonce))
	}
	return fmt.Sprintf("Digest %s", strings.Join(sl, ",")), nil
}

// origin https://code.google.com/p/mlab-ns2/source/browse/gae/ns/digest/digest.go#90
func (c *challenge) parseChallenge(input string) error {
	const ws = " \n\r\t"
	const qs = `"`
	s := strings.Trim(input, ws)
	if !strings.HasPrefix(s, "Digest ") {
		return fmt.Errorf("challenge is bad, missing prefix: %s", input)
	}
	s = strings.Trim(s[7:], ws)
	sl := strings.Split(s, ",")
	c.Algorithm = "MD5"
	var r []string
	for i := range sl {
		r = strings.SplitN(sl[i], "=", 2)
		switch strings.TrimSpace(r[0]) {
		case "realm":
			c.Realm = strings.Trim(r[1], qs)
		case "domain":
			c.Domain = strings.Trim(r[1], qs)
		case "nonce":
			c.Nonce = strings.Trim(r[1], qs)
		case "opaque":
			c.Opaque = strings.Trim(r[1], qs)
		case "stale":
			c.Stale = strings.Trim(r[1], qs)
		case "algorithm":
			c.Algorithm = strings.Trim(r[1], qs)
		case "qop":
			// TODO(gavaletz) should be an array of strings?
			c.Qop = strings.Trim(r[1], qs)
		default:
			return fmt.Errorf("challenge is bad, unexpected token: %s", sl)
		}
	}
	return nil
}
