// Command import-srd loads the D&D 5e SRD monster list into npc_templates as
// shared (owner-less) templates.
//
// Data comes from the 5e-bits/5e-database project. The SRD content is licensed
// CC-BY-4.0 by Wizards of the Coast, so the app must show the attribution in
// srdAttribution somewhere users can see it.
//
// The import is idempotent: rows are keyed by (source, source_key), so running
// it again refreshes existing templates instead of duplicating them. The full
// source record is kept in npc_templates.details (JSONB) for later use.
//
// Usage (from backend/, with the same DB_* env vars / .env as the server, and
// after the server has run its migrations at least once):
//
//	go run ./cmd/import-srd                  # SRD 5.2 (2024 rules), downloaded
//	go run ./cmd/import-srd -srd 5.1         # SRD 5.1 (2014 rules)
//	go run ./cmd/import-srd -file monsters.json -dry-run
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"go-initiative-tracker/dao"
)

// srdAttribution is the CC-BY-4.0 credit line WotC asks for; %s is the version.
const srdAttribution = "This work includes material from the System Reference Document %[1]s (\"SRD %[1]s\") " +
	"by Wizards of the Coast LLC, available at https://www.dndbeyond.com/srd. " +
	"The SRD %[1]s is licensed under the Creative Commons Attribution 4.0 International License."

// srdSources maps the -srd flag to the 5e-database folder and our source label.
var srdSources = map[string]struct{ folder, source string }{
	"5.1": {"2014", "srd-5.1"},
	"5.2": {"2024", "srd-5.2"},
}

const dataURL = "https://raw.githubusercontent.com/5e-bits/5e-database/main/src/%s/en/5e-SRD-Monsters.json"

func main() {
	log.SetFlags(0)
	srd := flag.String("srd", "5.2", "SRD version to import: 5.1 or 5.2")
	file := flag.String("file", "", "read monsters from this local JSON file instead of downloading")
	dryRun := flag.Bool("dry-run", false, "parse and print a summary without touching the database")
	flag.Parse()

	src, ok := srdSources[*srd]
	if !ok {
		log.Fatalf("unknown -srd %q (want 5.1 or 5.2)", *srd)
	}

	raw, err := loadData(*file, fmt.Sprintf(dataURL, src.folder))
	if err != nil {
		log.Fatal(err)
	}
	monsters, err := parseMonsters(raw)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("parsed %d monsters for %s", len(monsters), src.source)

	if *dryRun {
		for _, m := range monsters[:min(5, len(monsters))] {
			log.Printf("  %s | %s | AC %d | HP %d | %s", m.Key, m.Template.Name, m.Template.ArmorClass, m.Template.MaxHP, m.Template.Description)
		}
		return
	}

	_ = godotenv.Load()
	sslmode := os.Getenv("SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), sslmode))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	templates := dao.NewNpcTemplateDAO(db)
	var inserted, updated int
	for _, m := range monsters {
		isNew, err := templates.UpsertImported(m.Template, src.source, m.Key, m.Details)
		if err != nil {
			log.Fatalf("importing %s: %v", m.Key, err)
		}
		if isNew {
			inserted++
		} else {
			updated++
		}
	}
	log.Printf("done: %d inserted, %d updated", inserted, updated)
	log.Printf("remember the attribution: "+srdAttribution, *srd)
}

func loadData(file, url string) ([]byte, error) {
	if file != "" {
		return os.ReadFile(file)
	}
	log.Printf("downloading %s", url)
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}
