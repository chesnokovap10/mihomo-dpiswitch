package route

import (
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	C "github.com/metacubex/mihomo/constant"

	"github.com/metacubex/http"
	"github.com/metacubex/http/httptest"
)

// The API DPI Switch's core keeps: the config re-read from its own file
// only, TUN switched off only, and no download, upgrade or restart at all.
func TestDPISwitchAPI(t *testing.T) {
	r := router(false, "", "", Cors{})
	ask := func(method, path, body string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
		return w.Code
	}
	for _, c := range []struct{ method, path, body string }{
		{"PUT", "/configs", `{"payload":"mixed-port: 7890"}`},
		{"PUT", "/configs?force=true", `{"path":"C:\\elsewhere\\config.yaml"}`},
		{"PUT", "/configs", `{"path":"","PAYLOAD":"mode: global"}`},
		{"PATCH", "/configs", `{"allow-lan":true}`},
		{"PATCH", "/configs", `{"tun":{"enable":true}}`},
		{"PATCH", "/configs", `{"tun":{"enable":false,"stack":"gvisor"}}`},
		{"PATCH", "/configs", `{"tun":{"enable":false},"mixed-port":7890}`},
		{"PATCH", "/configs", `{}`},
	} {
		if code := ask(c.method, c.path, c.body); code != http.StatusForbidden && code != http.StatusBadRequest {
			t.Errorf("%s %s %s: %d, want it refused", c.method, c.path, c.body, code)
		}
	}
	for _, c := range []struct{ method, path string }{
		{"POST", "/upgrade"}, {"POST", "/upgrade/ui"}, {"POST", "/upgrade/geo"},
		{"POST", "/restart"}, {"POST", "/configs/geo"},
	} {
		if code := ask(c.method, c.path, ""); code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: %d, want no such route", c.method, c.path, code)
		}
	}
}

// What the wrappers let through reaches the handler as a body of their own
// making: the core's own path, TUN off -- nothing else the request carried.
func TestDPISwitchAPIPasses(t *testing.T) {
	own, sep := C.Path.Config(), string(filepath.Separator)
	var got string
	next := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
	}
	bodies := []string{`{}`, `{"path":""}`, `{"path":` + quote(own) + `}`,
		`{"path":` + quote(filepath.Dir(own)+sep+"."+sep+filepath.Base(own)) + `}`}
	if runtime.GOOS == "windows" {
		// the system's own spelling rules: the case does not matter
		bodies = append(bodies, `{"path":`+quote(strings.ToUpper(own))+`}`)
	}
	for _, body := range bodies {
		got = ""
		w := httptest.NewRecorder()
		onlyOwnConfig(next)(w, httptest.NewRequest("PUT", "/configs", strings.NewReader(body)))
		if want := `{"path":` + quote(own) + `}`; got != want {
			t.Errorf("PUT %s: the handler got %q (%d), want %q", body, got, w.Code, want)
		}
	}
	got = ""
	w := httptest.NewRecorder()
	onlyTunOff(next)(w, httptest.NewRequest("PATCH", "/configs", strings.NewReader(`{"tun":{"enable":false}}`)))
	if got != tunOffBody {
		t.Errorf("PATCH: the handler got %q (%d)", got, w.Code)
	}
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
