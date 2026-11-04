package serversetup

import (
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Run the actual embedded parser offline, without SSH, WireGuard or root.
func TestPeerNameMarkers(t *testing.T) {
	if _, err := osexec.LookPath("bash"); err != nil {
		t.Skip("bash is unavailable")
	}
	dir := t.TempDir()
	conf := filepath.Join(dir, "wg.conf")
	fixture := `[Interface]
# client: Must not leak to a peer
PrivateKey = ignored
[Peer]
# client: Алиса 🙂
PublicKey = alice=
AllowedIPs = 10.13.13.3/32
[Peer]
# client: Old name
# ft-user: Qm9i
# client: Must not replace Bob
PublicKey = bob=
AllowedIPs = 10.13.13.4/32
[Peer]
PublicKey = unnamed=
AllowedIPs = 10.13.13.5/32
`
	if err := os.WriteFile(conf, []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	parser, err := scriptFS.ReadFile("server-control/src/65-share.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := `set -euo pipefail
WG_CONF=$1
SHARE_DIR=$2
WG_IFACE=ft-wg0
wg_present() { return 0; }
wg_is_ours() { return 0; }
wg() { return 0; }
esc() { printf '%s' "$1"; }
` + string(parser) + "\npeers_json\n"
	cmd := osexec.Command("bash", "-s", "--", conf, dir)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("parser: %v: %s", err, out)
	}
	list, err := parseShareList(json.RawMessage(`{"peers":` + string(out) + `}`))
	if err != nil {
		t.Fatalf("invalid peer JSON: %v: %s", err, out)
	}
	if len(list.Peers) != 3 || list.Peers[0].Name != "Алиса 🙂" || list.Peers[1].Name != "Bob" || list.Peers[2].Name != "" {
		t.Fatalf("names: %+v", list.Peers)
	}
}
