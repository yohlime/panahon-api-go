// Package public builds the filtered Swagger document exposed to external API
// consumers. The full generated document remains available at /docs.
package public

import (
	"encoding/json"
	"fmt"
	"strings"

	apiDocs "github.com/emiliogozo/panahon-api-go/internal/docs/api"
	"github.com/swaggo/swag"
)

var publicPaths = map[string]map[string]bool{
	"/stations":                                  {"get": true},
	"/stations/{station_id}":                     {"get": true},
	"/stations/nearest/observations/latest":      {"get": true},
	"/stations/{station_id}/observations":        {"get": true},
	"/stations/{station_id}/observations/{id}":   {"get": true},
	"/stations/{station_id}/observations/latest": {"get": true},
	"/observations":                              {"get": true},
	"/observations/latest":                       {"get": true},
}

var apiKeyPaths = map[string]bool{
	"/stations/{station_id}/observations":        true,
	"/stations/{station_id}/observations/{id}":   true,
	"/stations/{station_id}/observations/latest": true,
	"/observations": true,
}

const publicDescription = "Public station and observation endpoints. " +
	"Pass an API key as the `api_key` query parameter for observation queries. " +
	"Keys are scoped by station, variable, and history depth."

// PublicSpec returns the generated Swagger document filtered to the read-only
// station/observation API, with only APITokenAuth advertised.
func PublicSpec() ([]byte, error) {
	raw, err := swag.ReadDoc(apiDocs.SwaggerInfo.InstanceName())
	if err != nil {
		return nil, err
	}
	return filter([]byte(raw))
}

func filter(raw []byte) ([]byte, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("swagger document has no paths")
	}

	filteredPaths := make(map[string]any)
	refs := make(map[string]bool)

	for path, methodsAny := range paths {
		allowed, ok := publicPaths[path]
		if !ok {
			continue
		}

		methods, ok := methodsAny.(map[string]any)
		if !ok {
			continue
		}

		kept := make(map[string]any)
		for method, operationAny := range methods {
			if !allowed[strings.ToLower(method)] {
				continue
			}
			operation, ok := operationAny.(map[string]any)
			if !ok {
				continue
			}

			// The public docs only advertise API-key auth. Public read
			// endpoints do not show the admin/session Bearer scheme.
			if apiKeyPaths[path] {
				operation["security"] = []any{
					map[string]any{"APITokenAuth": []any{}},
				}
			} else {
				delete(operation, "security")
			}

			kept[method] = operation
			collectRefs(operation, refs)
		}

		if len(kept) > 0 {
			filteredPaths[path] = kept
		}
	}
	doc["paths"] = filteredPaths

	definitions, _ := doc["definitions"].(map[string]any)
	doc["definitions"] = pruneDefinitions(definitions, refs)

	securityDefinitions, _ := doc["securityDefinitions"].(map[string]any)
	if apiKey, ok := securityDefinitions["APITokenAuth"]; ok {
		doc["securityDefinitions"] = map[string]any{"APITokenAuth": apiKey}
	} else {
		delete(doc, "securityDefinitions")
	}

	if info, ok := doc["info"].(map[string]any); ok {
		info["title"] = "Panahon Public API"
		info["description"] = publicDescription
		doc["info"] = info
	}

	return json.MarshalIndent(doc, "", "  ")
}

func pruneDefinitions(definitions map[string]any, refs map[string]bool) map[string]any {
	pruned := make(map[string]any)
	pending := make([]string, 0, len(refs))
	for ref := range refs {
		pending = append(pending, ref)
	}

	seen := make(map[string]bool)
	for len(pending) > 0 {
		name := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[name] {
			continue
		}
		seen[name] = true

		definition, ok := definitions[name]
		if !ok {
			continue
		}
		pruned[name] = definition

		childRefs := make(map[string]bool)
		collectRefs(definition, childRefs)
		for ref := range childRefs {
			if !seen[ref] {
				pending = append(pending, ref)
			}
		}
	}

	return pruned
}

func collectRefs(value any, refs map[string]bool) {
	switch v := value.(type) {
	case map[string]any:
		if ref, ok := v["$ref"].(string); ok {
			if strings.HasPrefix(ref, "#/definitions/") {
				refs[strings.TrimPrefix(ref, "#/definitions/")] = true
			}
		}
		for _, child := range v {
			collectRefs(child, refs)
		}
	case []any:
		for _, child := range v {
			collectRefs(child, refs)
		}
	}
}
