package route

// DPI Switch runs this core as SYSTEM, and the account the service is
// installed for -- which need not be an administrator -- reads the API's
// secret: its UI lists the connections and closes them. With the whole API
// that secret replaced the running config (PUT /configs with a payload), and
// with it had SYSTEM download and write files under the home directory, open
// listeners, or swap the binary for an upstream release (/upgrade).
//
// So the API keeps what DPI Switch calls, and the writes among them are
// narrowed to exactly what it sends:
//
//   - PUT /configs re-reads the config the core was started with, from its
//     own path -- never a payload, never another file;
//   - PATCH /configs only switches TUN off, as the service does before it
//     stops the core;
//   - /upgrade, /restart and /configs/geo are gone.
//
// The rest stays: the connections, the proxies and the groups, the
// providers' reloads, the logs.

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	C "github.com/metacubex/mihomo/constant"

	"github.com/metacubex/chi/render"
	"github.com/metacubex/http"
)

// apiBodyMax: the most of a request body read; what DPI Switch sends is a
// few dozen bytes
const apiBodyMax = 64 << 10

// refuse answers a write DPI Switch does not make
func refuse(w http.ResponseWriter, r *http.Request, why string) {
	render.Status(r, http.StatusForbidden)
	render.JSON(w, r, newError(why))
}

// onlyOwnConfig lets PUT /configs re-read the core's own config file and
// nothing else. The handler is given a body of its own making, so whatever
// else the request carried cannot reach it.
func onlyOwnConfig(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, apiBodyMax+1))
		var req struct {
			Path    string `json:"path"`
			Payload string `json:"payload"`
		}
		if err != nil || len(b) > apiBodyMax || json.Unmarshal(b, &req) != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, ErrBadRequest)
			return
		}
		if req.Payload != "" {
			refuse(w, r, "a config sent in the request is not taken: only the core's own file is re-read")
			return
		}
		if req.Path != "" && !samePath(req.Path, C.Path.Config()) {
			refuse(w, r, "only the core's own config file is re-read")
			return
		}
		body, _ := json.Marshal(struct {
			Path string `json:"path"`
		}{C.Path.Config()})
		r.Body = io.NopCloser(bytes.NewReader(body))
		next(w, r)
	}
}

// tunOffBody: the one PATCH /configs taken
const tunOffBody = `{"tun":{"enable":false}}`

// onlyTunOff lets PATCH /configs switch TUN off and nothing else: no field
// besides tun.enable, and that one false.
func onlyTunOff(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, apiBodyMax+1))
		if err != nil || len(b) > apiBodyMax {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, ErrBadRequest)
			return
		}
		var req struct {
			Tun *struct {
				Enable *bool `json:"enable"`
			} `json:"tun"`
		}
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if d.Decode(&req) != nil || req.Tun == nil || req.Tun.Enable == nil || *req.Tun.Enable {
			refuse(w, r, `only {"tun":{"enable":false}} is taken`)
			return
		}
		r.Body = io.NopCloser(strings.NewReader(tunOffBody))
		next(w, r)
	}
}

// samePath: two spellings of one path, the way the system compares them
func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
