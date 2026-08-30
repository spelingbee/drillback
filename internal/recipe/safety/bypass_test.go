package safety

import (
	"strings"
	"testing"
)

// The two P0 findings of the session 4 security review, and the class each of them
// belongs to. Every case here was accepted by `recipe validate --strict` before
// ADR-056 and ADR-057; each one is a real bypass with a real blast radius, not a
// hypothetical, so each gets a test that names it.

// SEC-02: a "named volume" whose driver options make it a bind mount of an arbitrary
// host path. The service body looks unremarkable; the whole trick is in the
// top-level volumes block that nothing used to validate.
func TestTopLevelVolumeCannotBindAHostPath(t *testing.T) {
	cases := map[string]string{
		"driver_opts binding the host root": `services:
  app:
    image: example/app:1.0.0
    volumes:
      - ${RESTORED_INPUT_data}:/data
      - hostroot:/host
    networks: [restored]
networks:
  restored:
    internal: true
volumes:
  hostroot:
    driver: local
    driver_opts:
      type: none
      device: /
      o: bind
`,
		"driver_opts binding the docker socket directory": `services:
  app:
    image: example/app:1.0.0
    volumes:
      - ${RESTORED_INPUT_data}:/data
      - sock:/sock
    networks: [restored]
networks:
  restored:
    internal: true
volumes:
  sock:
    driver: local
    driver_opts:
      type: none
      device: /var/run
      o: bind
`,
		"an external volume belonging to someone else": `services:
  app:
    image: example/app:1.0.0
    volumes:
      - ${RESTORED_INPUT_data}:/data
      - borrowed:/borrowed
    networks: [restored]
networks:
  restored:
    internal: true
volumes:
  borrowed:
    external: true
    name: someone-elses-data
`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateSchema([]byte(doc)); err == nil {
				t.Fatal("accepted a top-level volume that reaches outside the workspace")
			}
		})
	}
}

// A plain named volume is the point of the block and must keep working.
func TestTopLevelNamedVolumeIsAccepted(t *testing.T) {
	doc := `services:
  app:
    image: example/app:1.0.0
    volumes:
      - ${RESTORED_INPUT_data}:/data
      - scratch:/scratch
    networks: [restored]
networks:
  restored:
    internal: true
volumes:
  scratch:
  labelled:
    labels:
      com.example.why: a named volume with labels is still a named volume
`
	if err := ValidateSchema([]byte(doc)); err != nil {
		t.Fatalf("rejected a plain named volume: %v", err)
	}
}

// SEC-04: the service body used to be a deny-list, so every compose key nobody had
// thought of was granted. volumes_from is the one with teeth - it attaches the
// volumes of a container already running on the host.
func TestServiceBodyIsAnAllowList(t *testing.T) {
	cases := map[string]string{
		"volumes_from": "    volumes_from: [container:someone-elses]\n",
		"extra_hosts":  "    extra_hosts: [reachme:host-gateway]\n",
		"uts":          "    uts: host\n",
		"sysctls":      "    sysctls: {net.ipv4.ip_forward: 1}\n",
		"group_add":    "    group_add: [nine]\n",
		"env_file":     "    env_file: ../../../etc/passwd\n",
		"links":        "    links: [other]\n",
		"dns":          "    dns: [1.1.1.1]\n",
	}
	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			doc := `services:
  app:
    image: example/app:1.0.0
    networks: [restored]
` + line + `networks:
  restored:
    internal: true
`
			if err := ValidateSchema([]byte(doc)); err == nil {
				t.Fatalf("accepted service key %q, which the allow-list does not name", name)
			}
		})
	}
}

// MNT-01: configs and secrets both take a `file:` the daemon reads from the host.
func TestTopLevelConfigsAndSecretsAreRejected(t *testing.T) {
	for _, key := range []string{"configs", "secrets"} {
		t.Run(key, func(t *testing.T) {
			doc := `services:
  app:
    image: example/app:1.0.0
    networks: [restored]
networks:
  restored:
    internal: true
` + key + `:
  leak:
    file: /etc/shadow
`
			err := ValidateSchema([]byte(doc))
			if err == nil {
				t.Fatalf("accepted a top-level %s block", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Fatalf("the error does not name %s: %v", key, err)
			}
		})
	}
}

// SEC-01: the injection itself. The compose file is valid and the schema sees it
// with the placeholder intact; the variable supplies the rest of the document.
func TestVariableCannotInjectYAML(t *testing.T) {
	const doc = `services:
  app:
    image: example/app:1.0.0
    environment:
      APP_PORT: ${RESTORED_VAR_port}
    volumes:
      - ${RESTORED_INPUT_data}:/data
    networks: [restored]
networks:
  restored:
    internal: true
`
	env := map[string]string{
		"RESTORED_INPUT_data": "/ws/inputs/data",
		"RESTORED_VAR_port":   "8080\n    privileged: true\n    network_mode: host\n    pid: host",
	}
	if _, err := Render([]byte(doc), env); err == nil {
		t.Fatal("rendered a compose file with privileged, network_mode and pid injected through a variable")
	}
	// The same value with the line breaks taken out is an ordinary value.
	env["RESTORED_VAR_port"] = "8080"
	if _, err := Render([]byte(doc), env); err != nil {
		t.Fatalf("rejected an ordinary variable value: %v", err)
	}
}

// The invariant Render is built on, tested directly: interpolation may change what a
// value is, never what the document is. This is the check that closes the class
// rather than the instance, so it is tested without going through Interpolate.
func TestCheckInterpolationShape(t *testing.T) {
	const before = `services:
  app:
    image: example/app:1.0.0
    environment:
      PORT: PLACEHOLDER
    networks: [restored]
`
	cases := map[string]struct {
		after string
		ok    bool
	}{
		"a value changes": {
			after: strings.Replace(before, "PLACEHOLDER", "8080", 1),
			ok:    true,
		},
		"a key is added": {after: `services:
  app:
    image: example/app:1.0.0
    environment:
      PORT: 8080
    privileged: true
    networks: [restored]
`},
		"a key is removed": {after: `services:
  app:
    image: example/app:1.0.0
    networks: [restored]
`},
		"a scalar becomes a list": {after: `services:
  app:
    image: example/app:1.0.0
    environment:
      PORT: [8080, 8081]
    networks: [restored]
`},
		"a list grows": {after: `services:
  app:
    image: example/app:1.0.0
    environment:
      PORT: 8080
    networks: [restored, other]
`},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			err := CheckInterpolationShape([]byte(before), []byte(c.after))
			if c.ok && err != nil {
				t.Fatalf("rejected a legitimate substitution: %v", err)
			}
			if !c.ok && err == nil {
				t.Fatal("accepted a substitution that changed the shape of the document")
			}
		})
	}
}
