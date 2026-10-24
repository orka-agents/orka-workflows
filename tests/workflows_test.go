package workflows_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"go.yaml.in/yaml/v3"
)

var writePaths = map[string]string{
	"documentation-updater": "demo/README.md",
	"test-improver":         "demo/test.sh",
	"code-simplifier":       "demo/textkit.sh",
}

var workflowNames = []string{
	"repository-analyst", "documentation-updater", "test-improver", "code-simplifier",
}

func TestWorkflowPolicies(t *testing.T) {
	for _, name := range workflowNames {
		t.Run(name, func(t *testing.T) {
			task := readManifest(t, "workflows", name, "task.yaml")
			if err := taskPolicy(name, task); err != nil {
				t.Fatal(err)
			}
		})
	}

	agent := readManifest(t, "agents", "codex.yaml")
	if err := agentPolicy(agent); err != nil {
		t.Fatal(err)
	}
}

func agentPolicy(agent map[string]any) error {
	spec := object(agent, "spec")
	for path, want := range map[string]any{
		"apiVersion": "core.orka.ai/v1alpha1", "kind": "Agent",
		"metadata.name":   "workflows-codex",
		"spec.model.name": "gpt-5.4", "spec.tools": []any{},
		"spec.coordination.enabled":        false,
		"spec.runtime.type":                "codex",
		"spec.runtime.contractVersion":     "orka.harness.v2",
		"spec.runtime.defaultMaxTurns":     float64(24),
		"spec.runtime.defaultAllowBash":    true,
		"spec.runtime.defaultAllowedTools": nil,
	} {
		if got := value(agent, path); !reflect.DeepEqual(got, want) {
			return fmt.Errorf("agent %s = %v, want %v", path, got, want)
		}
	}
	for _, path := range []string{"providerRef", "secretRef", "model.apiKeySecretRef", "runtime.credentials"} {
		if value(spec, path) != nil {
			return fmt.Errorf("ACP Agent must not carry %s", path)
		}
	}
	return nil
}

func taskPolicy(name string, task map[string]any) error {
	for path, want := range map[string]any{
		"apiVersion": "core.orka.ai/v1alpha1", "kind": "Task",
		"spec.type": "agent", "spec.agentRef.name": "workflows-codex",
		"spec.timeout": "10m", "spec.retryPolicy.maxRetries": float64(0),
	} {
		if got := value(task, path); !reflect.DeepEqual(got, want) {
			return fmt.Errorf("%s = %v, want %v", path, got, want)
		}
	}
	for _, path := range []string{"spec.schedule", "spec.secretRef", "spec.agentRuntime.workspace", "spec.sessionRef", "spec.priorTaskRef"} {
		if value(task, path) != nil {
			return fmt.Errorf("one-shot sample must not set %s", path)
		}
	}
	prompt, _ := value(task, "spec.prompt").(string)
	if strings.TrimSpace(prompt) == "" || strings.Contains(prompt, "${{") || strings.Contains(prompt, "safe-outputs") {
		return fmt.Errorf("prompt must be nonempty and Orka-native")
	}
	w := object(task, "spec", "workspace")
	if name == "repository-analyst" {
		want := map[string]any{
			"intent": "read", "gitRepo": "https://github.com/orka-agents/orka-workflows.git",
		}
		if !reflect.DeepEqual(w, want) {
			return fmt.Errorf("analyst must use public read-only workspace without publication fields")
		}
		if value(task, "metadata.name") != name || value(task, "metadata.generateName") != nil ||
			value(task, "spec.agentRuntime.maxTurns") != float64(16) ||
			value(task, "spec.agentRuntime.allowBash") != false ||
			!reflect.DeepEqual(value(task, "spec.agentRuntime.allowedTools"), []any{"Glob", "Grep", "Read"}) {
			return fmt.Errorf("analyst identity or runtime changed")
		}
		return nil
	}
	allowed, ok := writePaths[name]
	if !ok {
		return fmt.Errorf("unknown workflow %q", name)
	}
	if value(task, "metadata.generateName") != name+"-" || value(task, "metadata.name") != nil ||
		value(task, "spec.agentRuntime.maxTurns") != float64(24) {
		return fmt.Errorf("writer must use generated name and bounded turns")
	}
	for _, key := range []string{"allowedTools", "disallowedTools", "allowBash"} {
		if value(task, "spec.agentRuntime."+key) != nil {
			return fmt.Errorf("writer must inherit the unrestricted Codex native policy, not override %s", key)
		}
	}
	for path, want := range map[string]any{
		"intent": "write", "createPR": true,
		"gitRepo":            "https://github.com/YOUR-OWNER/orka-workflows.git",
		"publicationGitRepo": "https://github.com/YOUR-OWNER/orka-workflows.git",
		"branch":             "YOUR-BASE-BRANCH", "prBaseBranch": "YOUR-BASE-BRANCH",
		"readCredentialRef.name":            "workflows-read",
		"publicationReadCredentialRef.name": "workflows-read",
		"publicationCredentialRef.name":     "workflows-publish",
		"forgeCredentialRef.name":           "workflows-forge",
		"allowedPaths":                      []any{allowed}, "maxChangedFiles": float64(1),
		"denyRepositoryControlPaths": true, "rejectBinaryFiles": true, "rejectSecretLikeContent": true,
	} {
		if got := value(w, path); !reflect.DeepEqual(got, want) {
			return fmt.Errorf("workspace.%s = %v, want %v", path, got, want)
		}
	}
	for _, key := range []string{"pushBranch", "expectedRemoteSHA", "subPath"} {
		if w[key] != nil {
			return fmt.Errorf("writer must not set workspace.%s", key)
		}
	}
	if !strings.Contains(prompt, allowed) || !strings.Contains(prompt, "sh demo/test.sh") {
		return fmt.Errorf("writer prompt must name its allowed file and validation command")
	}
	return nil
}

// The pinned Orka controller rejects partial Codex write allowlists and accepts
// exactly Glob/Grep/Read for restricted read intent. Schema validation misses this.
// See internal/controller/acp_agent_configuration.go in the upstream revision.
func TestCodexPolicyRegressions(t *testing.T) {
	t.Run("partial writer allowlist", func(t *testing.T) {
		agent := readManifest(t, "agents", "codex.yaml")
		object(agent, "spec", "runtime")["defaultAllowedTools"] = []any{"Read", "Write", "Edit", "Bash"}
		if err := agentPolicy(agent); err == nil {
			t.Fatal("unsupported partial Codex writer policy accepted")
		}
	})
	t.Run("partial analyst allowlist", func(t *testing.T) {
		task := readManifest(t, "workflows", "repository-analyst", "task.yaml")
		object(task, "spec", "agentRuntime")["allowedTools"] = []any{"Read", "Bash"}
		if err := taskPolicy("repository-analyst", task); err == nil {
			t.Fatal("unsupported partial Codex analyst policy accepted")
		}
	})
	t.Run("writer disables bash", func(t *testing.T) {
		task := readManifest(t, "workflows", "test-improver", "task.yaml")
		object(task, "spec", "agentRuntime")["allowBash"] = false
		if err := taskPolicy("test-improver", task); err == nil {
			t.Fatal("unsupported restricted Codex writer policy accepted")
		}
	})
}

func TestPolicyRejectsUnsafeChanges(t *testing.T) {
	for _, tc := range []struct {
		name, workflow, key string
		unsafe              any
	}{
		{"read publication", "repository-analyst", "createPR", true},
		{"read credential", "repository-analyst", "forgeCredentialRef", map[string]any{"name": "token"}},
		{"broad path", "test-improver", "allowedPaths", []any{"**"}},
		{"missing base", "test-improver", "prBaseBranch", ""},
		{"fixed branch", "code-simplifier", "pushBranch", "main"},
		{"secret checks disabled", "documentation-updater", "rejectSecretLikeContent", false},
		{"shared publication", "documentation-updater", "publicationGitRepo", "https://github.com/orka-agents/orka-workflows.git"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := readManifest(t, "workflows", tc.workflow, "task.yaml")
			object(task, "spec", "workspace")[tc.key] = tc.unsafe
			if err := taskPolicy(tc.workflow, task); err == nil {
				t.Fatal("unsafe change accepted")
			}
		})
	}
}

func TestSchedule(t *testing.T) {
	scheduled := schedule(t)
	want := readManifest(t, "workflows", "repository-analyst", "task.yaml")
	object(want, "metadata")["name"] = "repository-analyst-weekly"
	spec := object(want, "spec")
	for k, v := range map[string]any{
		"schedule": "0 9 * * 1", "timeZone": "UTC", "concurrencyPolicy": "Forbid",
		"suspend": true, "startingDeadlineSeconds": float64(300),
		"successfulRunsHistoryLimit": float64(3), "failedRunsHistoryLimit": float64(1),
	} {
		spec[k] = v
	}
	if !reflect.DeepEqual(scheduled, want) {
		t.Fatal("schedule must reuse the analyst unchanged except for its name and scheduling fields")
	}
}

func TestShellDemo(t *testing.T) {
	run(t, "sh", "-n", "demo/textkit.sh")
	run(t, "sh", "-n", "demo/test.sh")
	run(t, "sh", "demo/test.sh")
}

// These are immutable upstream commits, not release tags or moving branches.
// Kubernetes CEL and admission rules still require a server-side dry run.
func TestOrkaSchemas(t *testing.T) {
	resources := []map[string]any{readManifest(t, "agents", "codex.yaml"), schedule(t)}
	for _, name := range workflowNames {
		resources = append(resources, readManifest(t, "workflows", name, "task.yaml"))
	}
	for _, pin := range []struct{ name, commit string }{
		{"v0.2.0", "5f4eb543b2b35a3afb8e7ea01f5f53985e25d4c1"},
		{"2026-10-01", "1cd16c88b43410e0ea46463f8a39b473d8e5250e"},
	} {
		t.Run(pin.name, func(t *testing.T) {
			for _, kind := range []string{"Agent", "Task"} {
				schema := upstreamSchema(t, pin.commit, kind)
				for _, resource := range resources {
					if resource["kind"] == kind {
						if err := schema.Validate(resource); err != nil {
							t.Errorf("%s %v: %v", kind, resource["metadata"], err)
						}
					}
				}
				if kind == "Task" {
					bad := readManifest(t, "workflows", "test-improver", "task.yaml")
					workspace := object(bad, "spec", "workspace")
					workspace["allowPaths"] = workspace["allowedPaths"]
					delete(workspace, "allowedPaths")
					if err := schema.Validate(bad); err == nil {
						t.Fatal("schema accepted misspelled allowedPaths")
					}
					delete(workspace, "allowPaths")
					workspace["intent"] = "read-write"
					if err := schema.Validate(bad); err == nil {
						t.Fatal("schema accepted invalid workspace intent")
					}
				}
			}
		})
	}
}

func upstreamSchema(t *testing.T, commit, kind string) *jsonschema.Resolved {
	t.Helper()
	url := "https://raw.githubusercontent.com/orka-agents/orka/" + commit +
		"/config/crd/bases/core.orka.ai_" + strings.ToLower(kind) + "s.yaml"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("fetch pinned CRD: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("fetch pinned CRD: HTTP %d", resp.StatusCode)
	}
	var crd struct {
		Spec struct {
			Versions []struct {
				Name   string
				Schema struct {
					OpenAPIV3Schema map[string]any `yaml:"openAPIV3Schema"`
				}
			}
		}
	}
	if err := yaml.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&crd); err != nil {
		t.Fatal(err)
	}
	for _, version := range crd.Spec.Versions {
		if version.Name != "v1alpha1" {
			continue
		}
		strictObjects(version.Schema.OpenAPIV3Schema)
		data, err := json.Marshal(version.Schema.OpenAPIV3Schema)
		if err != nil {
			t.Fatal(err)
		}
		var schema jsonschema.Schema
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		resolved, err := schema.Resolve(nil)
		if err != nil {
			t.Fatal(err)
		}
		return resolved
	}
	t.Fatal("pinned CRD has no v1alpha1 schema")
	return nil
}

// Reject typos where the CRD declares fields, but retain free-form maps.
func strictObjects(node any) {
	switch v := node.(type) {
	case map[string]any:
		if v["properties"] != nil && v["additionalProperties"] == nil {
			v["additionalProperties"] = false
		}
		for _, child := range v {
			strictObjects(child)
		}
	case []any:
		for _, child := range v {
			strictObjects(child)
		}
	}
}

func readManifest(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return manifest(t, data)
}

func manifest(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatal("expected exactly one YAML document")
	}
	// Match JSON Schema's JSON number representation.
	normalized, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(normalized, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func schedule(t *testing.T) map[string]any {
	t.Helper()
	return manifest(t, run(t, "kubectl", "kustomize", "schedules/repository-analyst"))
}

func run(t *testing.T, name string, args ...string) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = ".."
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, output)
	}
	return output
}

func value(doc map[string]any, path string) any {
	var v any = doc
	for _, part := range strings.Split(path, ".") {
		m, _ := v.(map[string]any)
		v = m[part]
	}
	return v
}

func object(doc map[string]any, path ...string) map[string]any {
	obj, _ := value(doc, strings.Join(path, ".")).(map[string]any)
	return obj
}
