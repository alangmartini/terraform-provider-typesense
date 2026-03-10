package generator

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/alanm/terraform-provider-typesense/internal/client"
	"github.com/alanm/terraform-provider-typesense/internal/tfnames"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func repoRootForTerraformValidate(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine test file path")
	}

	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func providerBinaryNameForTerraformValidate() string {
	if runtime.GOOS == "windows" {
		return "terraform-provider-typesense.exe"
	}

	return "terraform-provider-typesense"
}

func buildProviderBinaryForTerraformValidate(t *testing.T, repoRoot string) string {
	t.Helper()

	buildDir := t.TempDir()
	binaryPath := filepath.Join(buildDir, providerBinaryNameForTerraformValidate())

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build provider binary: %v\n%s", err, string(output))
	}

	return buildDir
}

func TestGenerateStopwordsUsesStopwordsSetResourceType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stopwords" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("X-TYPESENSE-API-KEY"); got != "test-key" {
			t.Fatalf("unexpected API key header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"stopwords":[{"id":"english","stopwords":["the","a"],"locale":"en"}]}`))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")
	host, portStr, err := net.SplitHostPort(serverURL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse test server port: %v", err)
	}

	g := New(&Config{
		Host:     host,
		Port:     port,
		Protocol: "http",
		APIKey:   "test-key",
	})

	f := hclwrite.NewEmptyFile()
	resourceNames := make(map[string]bool)
	var importCommands []ImportCommand

	if err := g.generateStopwords(context.Background(), f, resourceNames, &importCommands); err != nil {
		t.Fatalf("generateStopwords() returned error: %v", err)
	}

	hcl := string(f.Bytes())
	if !strings.Contains(hcl, `resource "`+tfnames.FullTypeName(tfnames.ResourceStopwordsSet)+`" "english"`) {
		t.Fatalf("generated HCL did not contain stopwords_set resource:\n%s", hcl)
	}

	if len(importCommands) != 1 {
		t.Fatalf("generateStopwords() produced %d import commands, want 1", len(importCommands))
	}
	if importCommands[0].ResourceType != tfnames.FullTypeName(tfnames.ResourceStopwordsSet) {
		t.Fatalf("generateStopwords() import resource type = %q, want %q", importCommands[0].ResourceType, tfnames.FullTypeName(tfnames.ResourceStopwordsSet))
	}
}

func TestGeneratedHCLValidatesWithTerraform(t *testing.T) {
	terraformPath, err := exec.LookPath("terraform")
	if err != nil {
		t.Skip("terraform binary not found in PATH")
	}

	root := repoRootForTerraformValidate(t)
	providerDir := buildProviderBinaryForTerraformValidate(t, root)

	baseCollection := &client.Collection{
		Name: "products",
		Fields: []client.CollectionField{
			{
				Name: "id",
				Type: "string",
			},
			{
				Name:    "embedding",
				Type:    "float[]",
				NumDim:  384,
				VecDist: "cosine",
				Embed: &client.FieldEmbed{
					From: []string{"title", "description"},
					ModelConfig: client.FieldModelConfig{
						ModelName: "ts/all-MiniLM-L12-v2",
					},
				},
				HnswParams: &client.FieldHnswParams{
					EfConstruction: 200,
					M:              16,
				},
			},
		},
	}

	type terraformValidateCase struct {
		name         string
		resourceName string
		appendBlocks func(body *hclwrite.Body)
	}

	cases := []terraformValidateCase{
		{
			name:         "cluster",
			resourceName: tfnames.ResourceCluster,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateClusterBlock(&client.Cluster{
					Name:                   "smoke-cluster",
					Memory:                 "2_gb",
					VCPU:                   "2_vcpus_4_hr_burst_per_day",
					TypesenseServerVersion: "30.2",
					Regions:                []string{"hyderabad"},
				}, "smoke_cluster"))
				body.AppendNewline()
			},
		},
		{
			name:         "collection",
			resourceName: tfnames.ResourceCollection,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateCollectionBlock(baseCollection, "products"))
				body.AppendNewline()
			},
		},
		{
			name:         "stopwords",
			resourceName: tfnames.ResourceStopwordsSet,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateStopwordsBlock(&client.StopwordsSet{
					ID:        "english",
					Stopwords: []string{"the", "a", "an"},
					Locale:    "en",
				}, "english"))
				body.AppendNewline()
			},
		},
		{
			name:         "synonym",
			resourceName: tfnames.ResourceSynonym,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateCollectionBlock(baseCollection, "products"))
				body.AppendNewline()
				body.AppendBlock(generateSynonymBlock(&client.Synonym{
					ID:       "shoe_terms",
					Synonyms: []string{"shoe", "sneaker"},
				}, "products", "shoe_terms"))
				body.AppendNewline()
			},
		},
		{
			name:         "override",
			resourceName: tfnames.ResourceOverride,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateCollectionBlock(baseCollection, "products"))
				body.AppendNewline()
				body.AppendBlock(generateOverrideBlock(&client.Override{
					ID: "featured",
					Rule: client.OverrideRule{
						Query: "featured",
						Match: "exact",
					},
				}, "products", "featured"))
				body.AppendNewline()
			},
		},
		{
			name:         "analytics_rule",
			resourceName: tfnames.ResourceAnalyticsRule,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateAnalyticsRuleBlock(&client.AnalyticsRule{
					Name:       "popular_searches",
					Type:       "popular_queries",
					Collection: "products",
					EventType:  "search",
					Params: map[string]any{
						"destination_collection": "product_queries",
						"limit":                  float64(1000),
					},
				}, "popular_searches"))
				body.AppendNewline()
			},
		},
		{
			name:         "api_key",
			resourceName: tfnames.ResourceAPIKey,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateAPIKeyBlock(&client.APIKey{
					Description: "search-only",
					Actions:     []string{"documents:search"},
					Collections: []string{"products"},
				}, "search_only"))
				body.AppendNewline()
			},
		},
		{
			name:         "nl_search_model",
			resourceName: tfnames.ResourceNLSearchModel,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateNLSearchModelBlock(&client.NLSearchModel{
					ID:        "nl-model",
					ModelName: "openai/gpt-4o-mini",
				}, "nl_model"))
				body.AppendNewline()
			},
		},
		{
			name:         "conversation_model",
			resourceName: tfnames.ResourceConversationModel,
			appendBlocks: func(body *hclwrite.Body) {
				body.AppendBlock(generateConversationModelBlock(&client.ConversationModel{
					ID:                "conversation-model",
					ModelName:         "openai/gpt-4o-mini",
					HistoryCollection: "conversation_history",
					SystemPrompt:      "Answer based on indexed content.",
				}, "conversation_model"))
				body.AppendNewline()
			},
		},
	}

	coveredResourceNames := make(map[string]bool, len(cases))
	for _, tc := range cases {
		coveredResourceNames[tc.resourceName] = true
	}
	if len(coveredResourceNames) != len(tfnames.GeneratedResourceNames) {
		t.Fatalf("terraform validate cases cover %d generated resource names, want %d", len(coveredResourceNames), len(tfnames.GeneratedResourceNames))
	}
	for _, resourceName := range tfnames.GeneratedResourceNames {
		if !coveredResourceNames[resourceName] {
			t.Fatalf("missing terraform validate case for generated resource %q", resourceName)
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tfDir := t.TempDir()
			cliConfigPath := filepath.Join(tfDir, "terraform.rc")
			cliConfig := fmt.Sprintf(`provider_installation {
  dev_overrides {
    %q = %q
  }
  direct {}
}
`, tfnames.ProviderSource, filepath.ToSlash(providerDir))
			if err := os.WriteFile(cliConfigPath, []byte(cliConfig), 0644); err != nil {
				t.Fatalf("failed to write terraform CLI config: %v", err)
			}

			f := hclwrite.NewEmptyFile()
			generateTerraformBlock(f)
			generateProviderBlock(f, "example.a1.typesense.net", 443, "https", true, true)
			tc.appendBlocks(f.Body())

			mainTFPath := filepath.Join(tfDir, "main.tf")
			if err := os.WriteFile(mainTFPath, f.Bytes(), 0644); err != nil {
				t.Fatalf("failed to write generated Terraform config: %v", err)
			}

			variablesTFPath := filepath.Join(tfDir, "variables.tf")
			if err := os.WriteFile(variablesTFPath, []byte("variable \"openai_api_key\" {\n  type = string\n}\n"), 0644); err != nil {
				t.Fatalf("failed to write Terraform variables file: %v", err)
			}

			cmd := exec.Command(terraformPath, "validate")
			cmd.Dir = tfDir
			cmd.Env = append(os.Environ(),
				"TF_CLI_CONFIG_FILE="+cliConfigPath,
				"TF_IN_AUTOMATION=1",
				"CHECKPOINT_DISABLE=1",
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("terraform validate failed: %v\n%s", err, string(output))
			}
		})
	}
}
