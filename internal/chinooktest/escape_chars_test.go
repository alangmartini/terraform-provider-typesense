//go:build e2e

package chinooktest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestEscapeChars exercises a synonym and an override whose IDs contain
// a space. Typesense addresses these via URL path segments, so the
// regression we guard against is forgetting to url.PathEscape the ID
// (which would yield a 404 on every read/update/delete).
func TestEscapeChars(t *testing.T) {
	cluster := StartCluster(t, "30.1")

	dir := t.TempDir()
	if err := writeEscapeFixture(dir); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	tf := NewTerraform(t, dir)
	vars := map[string]string{
		"typesense_host":     cluster.Host,
		"typesense_port":     fmt.Sprintf("%d", cluster.Port),
		"typesense_protocol": "http",
		"typesense_api_key":  cluster.APIKey,
	}

	if err := tf.Apply(vars); err != nil {
		t.Fatalf("apply: %v", err)
	}

	code, planOut, err := tf.PlanWithOutput(vars)
	if err != nil {
		t.Fatalf("plan after apply: %v", err)
	}
	if code != 0 {
		t.Errorf("plan exit code = %d after apply, want 0 (no changes)\noutput:\n%s", code, planOut)
	}

	cli := cluster.Client()
	ctx := context.Background()

	synSets, err := cli.ListSynonymSets(ctx)
	if err != nil {
		t.Fatalf("list synonym sets: %v", err)
	}
	found := false
	for _, set := range synSets {
		for _, item := range set.Synonyms {
			if item.ID == "rock and roll" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("synonym item with space in ID was not retrievable")
	}

	curSets, err := cli.ListCurationSets(ctx)
	if err != nil {
		t.Fatalf("list curation sets: %v", err)
	}
	found = false
	for _, set := range curSets {
		for _, item := range set.Curations {
			if item.ID == "promote acoustic" {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("override item with space in ID was not retrievable")
	}

	if err := tf.Destroy(vars); err != nil {
		t.Fatalf("destroy: %v", err)
	}
}

const escapeFixture = `terraform {
  required_providers {
    typesense = {
      source = "alanm/typesense"
    }
  }
}

variable "typesense_host"     { type = string }
variable "typesense_port"     { type = string }
variable "typesense_protocol" { type = string }
variable "typesense_api_key"  { type = string }

provider "typesense" {
  server_host     = var.typesense_host
  server_port     = tonumber(var.typesense_port)
  server_protocol = var.typesense_protocol
  server_api_key  = var.typesense_api_key
}

resource "typesense_collection" "products" {
  name = "products"

  field {
    name = "id"
    type = "string"
  }
  field {
    name = "name"
    type = "string"
  }
}

resource "typesense_synonym" "rock_and_roll" {
  collection = typesense_collection.products.name
  name       = "rock and roll"
  synonyms   = ["rock", "roll"]
}

resource "typesense_override" "promote_acoustic" {
  collection = typesense_collection.products.name
  name       = "promote acoustic"

  rule = {
    query = "acoustic"
    match = "exact"
  }
  filter_by             = "name:Acoustic"
  remove_matched_tokens = false
}
`

func writeEscapeFixture(dir string) error {
	return os.WriteFile(filepath.Join(dir, "main.tf"), []byte(escapeFixture), 0o600)
}
