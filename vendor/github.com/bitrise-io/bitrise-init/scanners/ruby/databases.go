package ruby

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

// databaseEnvVar represents an environment variable with its name and default value.
type databaseEnvVar struct {
	name         string
	defaultValue string
}

// databaseGem represents a detected database dependency and its container configuration.
type databaseGem struct {
	gemName             string
	adapterName         string // Rails adapter name in database.yml (e.g. "postgresql", "mysql2")
	containerName       string
	image               string
	ports               []string
	containerEnvKey     string // env var name the container needs (e.g., POSTGRES_PASSWORD)
	healthCheck         string
	isRelationalDB      bool
	connectionURLEnvKey string   // app-level env var for the service URL (e.g., REDIS_URL)
	connectionURL       string   // value for connectionURLEnvKey (e.g., redis://localhost:6379/0)
	aptPackages         []string // system packages required to compile the gem's native extension
	// hostValue overrides the default "localhost" for DB_HOST. Use "127.0.0.1" for MySQL,
	// which treats "localhost" as a Unix socket path rather than a TCP address.
	hostValue string
}

// databaseYMLInfo holds env var names and defaults extracted from config/database.yml.
type databaseYMLInfo struct {
	adapter        string // e.g. "postgresql", "mysql2", "sqlite3"
	hostEnvVar     databaseEnvVar
	usernameEnvVar databaseEnvVar
	passwordEnvVar databaseEnvVar
}

// mongoidYMLInfo holds connection URL info extracted from config/mongoid.yml.
type mongoidYMLInfo struct {
	connectionURLEnvKey string // e.g. "MONGODB_URL"
	connectionURL       string // e.g. "mongodb://localhost:27017/myapp_test"
}

var knownDatabaseGems = []databaseGem{
	{
		gemName:         "pg",
		adapterName:     "postgresql",
		containerName:   "postgres",
		image:           "postgres:17",
		ports:           []string{"5432:5432"},
		containerEnvKey: "POSTGRES_PASSWORD",
		healthCheck:     `--health-cmd "pg_isready" --health-interval 10s --health-timeout 5s --health-retries 5`,
		isRelationalDB:  true,
	},
	{
		gemName:         "mysql2",
		adapterName:     "mysql2",
		containerName:   "mysql",
		image:           "mysql:8",
		ports:           []string{"3306:3306"},
		containerEnvKey: "MYSQL_ROOT_PASSWORD",
		healthCheck:     `--health-cmd "mysqladmin ping -h 127.0.0.1 -u root --password=$$MYSQL_ROOT_PASSWORD" --health-interval 10s --health-timeout 5s --health-retries 5`,
		isRelationalDB:  true,
		aptPackages:     []string{"libmariadb-dev"},
		hostValue:       "127.0.0.1",
	},
	{
		gemName:             "redis",
		containerName:       "redis",
		image:               "redis:7",
		ports:               []string{"6379:6379"},
		healthCheck:         `--health-cmd "redis-cli ping" --health-interval 10s --health-timeout 5s --health-retries 5`,
		connectionURLEnvKey: "REDIS_URL",
		connectionURL:       "redis://localhost:6379/0",
	},
	{
		gemName:       "mongoid",
		containerName: "mongodb",
		image:         "mongo:8",
		ports:         []string{"27017:27017"},
		healthCheck:   `--health-cmd "mongosh --eval 'db.runCommand({ping:1})'" --health-interval 10s --health-timeout 5s --health-retries 5`,
	},
	{
		gemName:       "mongo",
		containerName: "mongodb",
		image:         "mongo:8",
		ports:         []string{"27017:27017"},
		healthCheck:   `--health-cmd "mongosh --eval 'db.runCommand({ping:1})'" --health-interval 10s --health-timeout 5s --health-retries 5`,
	},
	{
		// SQLite is file-based, no service container needed, but ActiveRecord setup is required
		gemName:        "sqlite3",
		isRelationalDB: true,
	},
}

var (
	gemDeclPattern     = regexp.MustCompile(`^\s*gem\s+['"]([^'"]+)['"]`)
	envFetchPattern    = regexp.MustCompile(`ENV\.fetch\(\s*["'](\w+)["']\s*\)\s*\{\s*["']([^"']*)["']\s*\}`)
	envFetchArgPattern = regexp.MustCompile(`ENV\.fetch\(\s*['"](\w+)['"]\s*,\s*['"]([^'"]*)['"]\s*\)`)
	envBracketPattern  = regexp.MustCompile(`ENV\[["'](\w+)["']\]`)
	// erbTagPattern matches ERB template tags like <%= ... %> that appear in Rails database.yml.
	// It assumes the expression itself does not contain a bare '%>' sequence.
	erbTagPattern = regexp.MustCompile(`<%[^%]*%>`)
)

func detectDatabases(searchDir string) []databaseGem {
	gemfilePath := filepath.Join(searchDir, "Gemfile")
	content, err := fileutil.ReadStringFromFile(gemfilePath)
	if err != nil {
		log.TWarnf("Failed to read Gemfile: %s", err)
		return nil
	}

	databases := detectDatabaseGemsFromContent(content)
	return databases
}

func detectDatabaseGemsFromContent(content string) []databaseGem {
	declaredGems := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		match := gemDeclPattern.FindStringSubmatch(line)
		if len(match) >= 2 {
			declaredGems[match[1]] = true
		}
	}

	var detected []databaseGem
	seen := map[string]bool{}
	for _, dbGem := range knownDatabaseGems {
		dedupKey := dbGem.containerName
		if dedupKey == "" {
			dedupKey = dbGem.gemName
		}
		if declaredGems[dbGem.gemName] && !seen[dedupKey] {
			detected = append(detected, dbGem)
			seen[dedupKey] = true
		}
	}
	return detected
}

// hasRelationalDB returns true if any detected database is relational (pg, mysql).
func hasRelationalDB(databases []databaseGem) bool {
	for _, db := range databases {
		if db.isRelationalDB {
			return true
		}
	}
	return false
}

func parseDatabaseYML(searchDir string, databases []databaseGem) databaseYMLInfo {
	ymlPath := filepath.Join(searchDir, "config", "database.yml")
	content, err := fileutil.ReadStringFromFile(ymlPath)
	if err != nil {
		log.TPrintf("- config/database.yml - not found or not readable")
		return databaseYMLInfo{}
	}

	log.TPrintf("- config/database.yml - found, parsing credentials")
	return parseDatabaseYMLContent(content, databases)
}

// parseDatabaseYMLContent parses the contents of a database.yml file and extracts
// env-var references for the host, username, and password fields.
// It prefers the "test" environment section, then "default", then any other section.
// YAML anchor merges (<<: *default) are resolved automatically by the YAML parser.
// The adapter field is required: if absent or not matching a detected database gem, the result is empty.
func parseDatabaseYMLContent(content string, databases []databaseGem) databaseYMLInfo {
	preprocessed := preprocessERBForYAML(content)

	var rawYML map[string]map[string]interface{}
	if err := yaml.Unmarshal([]byte(preprocessed), &rawYML); err != nil {
		log.TWarnf("- config/database.yml - failed to parse: %s", err)
		return databaseYMLInfo{}
	}

	// Prefer "test", then "default", then the first available section.
	var section map[string]interface{}
	for _, name := range []string{"test", "default"} {
		if s, ok := rawYML[name]; ok {
			section = s
			break
		}
	}
	if section == nil {
		for _, s := range rawYML {
			section = s
			break
		}
	}
	if section == nil {
		return databaseYMLInfo{}
	}

	info := databaseYMLInfo{
		adapter:        asString(section["adapter"]),
		hostEnvVar:     extractEnvVarFromValue(asString(section["host"])),
		usernameEnvVar: extractEnvVarFromValue(asString(section["username"])),
		passwordEnvVar: extractEnvVarFromValue(asString(section["password"])),
	}

	if info.adapter == "" {
		log.TWarnf("database.yml has no adapter field, skipping database.yml config")
		return databaseYMLInfo{}
	}

	for _, db := range databases {
		if db.adapterName == info.adapter {
			return info
		}
	}

	log.TWarnf("database.yml adapter %q does not match any detected database gem, skipping database.yml config", info.adapter)
	return databaseYMLInfo{}
}

// preprocessERBForYAML wraps ERB template tags (e.g. <%= ENV.fetch(...) %>) in
// single quotes so that the surrounding YAML can be parsed by a standard YAML parser.
func preprocessERBForYAML(content string) string {
	return erbTagPattern.ReplaceAllStringFunc(content, func(match string) string {
		// Escape any single quotes inside the ERB expression (YAML single-quote escaping uses '').
		escaped := strings.ReplaceAll(match, "'", "''")
		return "'" + escaped + "'"
	})
}

// asString converts any value from yaml.Unmarshal to its string representation.
func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// extractEnvVarFromValue extracts an env-var name and default from a YAML field value.
// It recognises ENV.fetch("KEY") { "default" }, ENV["KEY"], and plain string values.
func extractEnvVarFromValue(value string) databaseEnvVar {
	// ENV.fetch("KEY") { "default" }
	if match := envFetchPattern.FindStringSubmatch(value); len(match) >= 3 {
		return databaseEnvVar{name: match[1], defaultValue: match[2]}
	}
	// ENV["KEY"]
	if match := envBracketPattern.FindStringSubmatch(value); len(match) >= 2 {
		return databaseEnvVar{name: match[1], defaultValue: ""}
	}
	// Plain value (no ERB reference)
	if value != "" && !strings.Contains(value, "<%") {
		return databaseEnvVar{name: "", defaultValue: value}
	}
	return databaseEnvVar{}
}

// findMongoDBGem returns the first detected non-relational DB gem that has a container (e.g. mongoid/mongo).
func findMongoDBGem(databases []databaseGem) (databaseGem, bool) {
	for _, db := range databases {
		if !db.isRelationalDB && db.containerName != "" {
			return db, true
		}
	}
	return databaseGem{}, false
}

func parseMongoidYML(searchDir string) mongoidYMLInfo {
	ymlPath := filepath.Join(searchDir, "config", "mongoid.yml")
	content, err := fileutil.ReadStringFromFile(ymlPath)
	if err != nil {
		log.TPrintf("- config/mongoid.yml - not found or not readable")
		return mongoidYMLInfo{}
	}

	log.TPrintf("- config/mongoid.yml - found, parsing connection URL")
	return parseMongoidYMLContent(content)
}

func parseMongoidYMLContent(content string) mongoidYMLInfo {
	// Look for ENV.fetch('KEY', 'mongodb://...') pattern anywhere in the file
	match := envFetchArgPattern.FindStringSubmatch(content)
	if len(match) < 3 {
		return mongoidYMLInfo{}
	}

	envKey := match[1]
	defaultURL := match[2]

	// Script steps run on the host machine, not inside Docker, so they connect to service
	// containers via localhost (ports are mapped to the host).
	// Normalize any IP-based localhost references to the hostname form.
	connectionURL := strings.ReplaceAll(defaultURL, "127.0.0.1", "localhost")

	return mongoidYMLInfo{
		connectionURLEnvKey: envKey,
		connectionURL:       connectionURL,
	}
}
