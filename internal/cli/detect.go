package cli

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// This file reads somebody's existing docker-compose.yml and guesses a recipe from it.
//
// The guess is best-effort and says so: every place a human has to decide gets a TODO
// marker in the generated file, and the marker names the decision rather than shrugging
// at it. What the guess must never do is produce something that does not validate,
// because a contributor whose first command fails has learned that this project's
// tooling does not work, and that is the one thing there is no budget for.

// detectedDir is one directory an application keeps state in.
type detectedDir struct {
	Name      string // the input name: lower-case, [a-z][a-z0-9_]*
	Service   string
	Container string // the path inside the container
	Source    string // the host path or named volume it came from, for the README
}

// detectedDB is the database service a recipe will have to restore.
type detectedDB struct {
	Kind     string // postgres-dump | sqlite | unsupported
	Service  string
	Image    string
	Name     string // database name, from the environment
	User     string
	Password string
	File     string // for sqlite: the path inside the container
}

// detected is everything reading a compose file told us.
type detected struct {
	AppService string
	AppImage   string
	AppPort    int
	Dirs       []detectedDir
	DB         *detectedDB
	Services   []string
	Notes      []string
	// Infra names the services whose state is not the application's: the database,
	// whose restore arrives as a dump, and the caches, whose contents are derived.
	Infra map[string]bool

	// images and envs are the source compose file, kept so the generated
	// compose.yaml can carry over what it is safe to carry over.
	images map[string]string
	envs   map[string]map[string]string
}

type composeService struct {
	Image       string `yaml:"image"`
	Environment any    `yaml:"environment"`
	Volumes     []any  `yaml:"volumes"`
	Ports       []any  `yaml:"ports"`
	Expose      []any  `yaml:"expose"`
	Command     any    `yaml:"command"`
	Entrypoint  any    `yaml:"entrypoint"`
}

type composeDoc struct {
	Services map[string]composeService `yaml:"services"`
}

var (
	postgresImage = regexp.MustCompile(`(?i)(^|/)(postgres|postgis|timescale)`)
	mysqlImage    = regexp.MustCompile(`(?i)(^|/)(mysql|mariadb|percona)`)
	cacheImage    = regexp.MustCompile(`(?i)(^|/)(redis|valkey|memcached|rabbitmq|tika|gotenberg|elasticsearch|opensearch|solr|meilisearch)`)
	sqliteFile    = regexp.MustCompile(`(?i)\.(db|sqlite|sqlite3)$`)
	identifier    = regexp.MustCompile(`[^a-z0-9_]+`)
)

// detectCompose reads a compose file and works out what a recipe for it would say.
func detectCompose(file string) (*detected, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading --compose %q: %w", file, err)
	}
	var doc composeDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parsing --compose %q: %w", file, err)
	}
	if len(doc.Services) == 0 {
		return nil, fmt.Errorf("--compose %q defines no services", file)
	}

	d := &detected{
		Infra:  map[string]bool{},
		images: map[string]string{},
		envs:   map[string]map[string]string{},
	}
	for name, svc := range doc.Services {
		d.Services = append(d.Services, name)
		d.images[name] = svc.Image
		d.envs[name] = envMap(svc.Environment)
	}
	sort.Strings(d.Services)

	// Classify the services first: the application is whatever is left once the
	// databases and the supporting infrastructure are accounted for.
	var candidates []string
	for _, name := range d.Services {
		svc := doc.Services[name]
		env := envMap(svc.Environment)
		switch {
		case postgresImage.MatchString(svc.Image):
			d.Infra[name] = true
			if d.DB == nil {
				d.DB = &detectedDB{
					Kind: "postgres-dump", Service: name, Image: svc.Image,
					Name:     firstOf(env, "POSTGRES_DB", "POSTGRES_DATABASE", "DB_NAME"),
					User:     firstOf(env, "POSTGRES_USER", "DB_USER"),
					Password: firstOf(env, "POSTGRES_PASSWORD", "DB_PASSWORD"),
				}
			}
		case mysqlImage.MatchString(svc.Image):
			d.Infra[name] = true
			d.Notes = append(d.Notes, fmt.Sprintf(
				"service %q runs %s, and drillback v0.1 has no mysql input kind: the recipe "+
					"below cannot restore it yet. Adding one is a good first contribution; "+
					"see CONTRIBUTING.md, \"Contributing a source\".", name, svc.Image))
			if d.DB == nil {
				d.DB = &detectedDB{Kind: "unsupported", Service: name, Image: svc.Image}
			}
		case cacheImage.MatchString(svc.Image):
			d.Infra[name] = true
			d.Notes = append(d.Notes, fmt.Sprintf(
				"service %q looks like a cache or a helper (%s). It is kept so the "+
					"application starts, but nothing about it is restored, and no check "+
					"should depend on it.", name, svc.Image))
		default:
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		candidates = d.Services
	}
	// Prefer a service that publishes a port: that is the one a human talks to.
	d.AppService = candidates[0]
	for _, name := range candidates {
		if len(doc.Services[name].Ports) > 0 || len(doc.Services[name].Expose) > 0 {
			d.AppService = name
			break
		}
	}
	d.AppImage = doc.Services[d.AppService].Image
	d.AppPort = detectPort(doc.Services[d.AppService])
	if d.AppPort == 0 {
		d.AppPort = 8080
		d.Notes = append(d.Notes, fmt.Sprintf(
			"service %q publishes no port, so the ready probe below guesses 8080. "+
				"Put the port the application actually listens on into vars.app_port.",
			d.AppService))
	}

	used := map[string]bool{}
	for _, name := range d.Services {
		svc := doc.Services[name]
		if d.Infra[name] {
			// A database's own data directory is not an input: the restore arrives
			// as a dump and is loaded into a database this run created. A cache's
			// contents are derived, so restoring them would prove nothing.
			continue
		}
		for _, v := range svc.Volumes {
			source, target, ok := splitVolume(v)
			if !ok || target == "" {
				continue
			}
			if sqliteFile.MatchString(target) {
				if d.DB == nil || d.DB.Kind == "unsupported" {
					d.DB = &detectedDB{Kind: "sqlite", Service: name, File: target}
				}
				continue
			}
			if path.Ext(target) != "" {
				// A single file mount is configuration, not state. A recipe that
				// restored one would be restoring the operator's decisions rather
				// than the application's data.
				continue
			}
			key := inputName(target)
			for used[key] {
				key += "_" + inputName(name)
			}
			used[key] = true
			d.Dirs = append(d.Dirs, detectedDir{
				Name: key, Service: name, Container: target, Source: source,
			})
		}
		if d.DB == nil || d.DB.Kind != "sqlite" {
			for _, v := range envMap(svc.Environment) {
				if sqliteFile.MatchString(v) && strings.HasPrefix(v, "/") {
					d.DB = &detectedDB{Kind: "sqlite", Service: name, File: v}
					break
				}
			}
		}
	}

	if len(d.Dirs) == 0 {
		d.Notes = append(d.Notes, "no volume was found, so this compose file keeps no "+
			"state outside its containers. Either the application stores everything in a "+
			"database, or the compose file is incomplete. A recipe needs at least one input.")
	}
	if d.DB == nil {
		d.Notes = append(d.Notes, "no database service was recognised. If the application "+
			"has one, add the input by hand; if it does not, the checks have to prove the "+
			"restore from the files alone.")
	}
	return d, nil
}

// detectPort reads the container-side port out of ports: or expose:.
func detectPort(svc composeService) int {
	for _, p := range svc.Ports {
		if n := containerPort(p); n > 0 {
			return n
		}
	}
	for _, p := range svc.Expose {
		if n := toPort(fmt.Sprint(p)); n > 0 {
			return n
		}
	}
	return 0
}

// containerPort takes the right-hand side of a published port, which is the one that
// exists inside the network. drillback publishes nothing, so the left-hand side, which
// is the only part the operator chose, is exactly the part to discard.
func containerPort(p any) int {
	switch t := p.(type) {
	case map[string]any:
		return toPort(fmt.Sprint(t["target"]))
	default:
		s := fmt.Sprint(p)
		s = strings.TrimSuffix(strings.TrimSuffix(s, "/tcp"), "/udp")
		parts := strings.Split(s, ":")
		return toPort(parts[len(parts)-1])
	}
}

func toPort(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 || n > 65535 {
		return 0
	}
	return n
}

// splitVolume handles both compose volume syntaxes and returns source and target.
func splitVolume(v any) (source, target string, ok bool) {
	switch t := v.(type) {
	case string:
		parts := strings.Split(t, ":")
		switch len(parts) {
		case 1:
			return "", parts[0], true // an anonymous volume
		default:
			// A Windows source can carry its own colon; the target is the last
			// absolute segment before an option list.
			if len(parts) >= 3 && (parts[2] == "ro" || parts[2] == "rw" || parts[2] == "z" || parts[2] == "Z") {
				return parts[0], parts[1], true
			}
			return parts[0], parts[len(parts)-1], true
		}
	case map[string]any:
		return fmt.Sprint(t["source"]), fmt.Sprint(t["target"]), true
	}
	return "", "", false
}

// envMap flattens both environment syntaxes.
func envMap(v any) map[string]string {
	out := map[string]string{}
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			out[k] = fmt.Sprint(val)
		}
	case []any:
		for _, item := range t {
			k, val, ok := strings.Cut(fmt.Sprint(item), "=")
			if ok {
				out[k] = val
			}
		}
	}
	return out
}

func firstOf(env map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := env[k]; ok && v != "" && !strings.Contains(v, "$") {
			return v
		}
	}
	return ""
}

// inputName turns a container path into an input name the schema accepts.
func inputName(container string) string {
	base := path.Base(strings.TrimSuffix(container, "/"))
	name := identifier.ReplaceAllString(strings.ToLower(base), "_")
	name = strings.Trim(name, "_")
	if name == "" || name[0] < 'a' || name[0] > 'z' {
		name = "data_" + name
	}
	return strings.TrimSuffix(name, "_")
}
