package release

import (
	"bytes"
	"dockerun/internal/utils"
	"sort"
	"strings"
	"sync"
)

// 脚本输出可能跨多次 Write；按完整行脱敏，避免切开的敏感参数泄漏。
type scriptLogWriter struct {
	mu         sync.Mutex
	pending    []byte
	discarding bool
	secrets    []string
	log        func(string)
}

func scriptSecrets(s Snapshot) []string {
	keys := map[string]bool{}
	for _, key := range s.SensitiveParameters {
		keys[key] = true
	}
	seen := map[string]bool{}
	for key, value := range s.Environment.Parameters {
		if value != "" && (keys[key] || utils.SensitiveParameterName(key)) {
			for _, part := range strings.Split(value, "\n") {
				if part != "" {
					seen[part] = true
				}
			}
		}
	}
	var secrets []string
	for value := range seen {
		secrets = append(secrets, value)
	}
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	return secrets
}

func (w *scriptLogWriter) flush() {
	if len(w.pending) == 0 {
		return
	}
	message := string(w.pending)
	for _, secret := range w.secrets {
		message = strings.ReplaceAll(message, secret, "******")
	}
	w.log(message)
	w.pending = nil
}

func (w *scriptLogWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := len(data)
	for len(data) > 0 {
		end := bytes.IndexByte(data, '\n')
		complete := end >= 0
		if complete {
			end++
		} else {
			end = len(data)
		}
		if !w.discarding {
			if len(w.pending)+end > 1<<20 {
				w.pending = nil
				w.discarding = true
				w.log("单行日志超过 1 MiB，已省略该行\n")
			} else {
				w.pending = append(w.pending, data[:end]...)
			}
		}
		if complete {
			w.flush()
			w.discarding = false
		}
		data = data[end:]
	}
	return n, nil
}

func (w *scriptLogWriter) Flush() { w.mu.Lock(); defer w.mu.Unlock(); w.flush() }
