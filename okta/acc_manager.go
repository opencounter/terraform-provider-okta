package okta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-yaml/yaml"

	"github.com/hashicorp/terraform/terraform"

	jsonParser "github.com/hashicorp/hcl/json/parser"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/Jeffail/gabs"
)

// The ACC test manager is a way to prevent the code duplication we are often seeing with ACC tests. We always want.
// The same happy path tests for every resource and therefore why not automate it. This provides an interface for doing
// just that. Here are the "happy path" test cases this covers.
// * Minimal required config for a resource
// * Update with ALL config specified. Will rely on "Example:xyz" for format. If none is provided with will
// generate test data of the proper type.

type (
	accTestManager struct {
		// You have to opt out of this testing.
		resourceProviders map[string]terraform.ResourceProvider
		testableResources map[string]*schema.Resource
		testConfig        map[string]*testConfig
		preCheck          func(*testing.T)
	}

	resourceTest struct {
		checks resource.TestCheckFunc
		// Helper used to build dynamic test config. Will build this dynamically and do runtime conversions to HCL
		jsonConfig *gabs.Container
	}

	testConfig struct {
		Properties map[string]interface{}     `json:"properties,omitempty"`
		Tests      map[string]*testCaseConfig `json:"tests,omitempty"`
	}

	testCaseConfig struct {
		Description string        `json:"description,omitempty"`
		Steps       []*stepConfig `json:"steps,omitempty"`
	}

	stepConfig struct {
		Type       string                 `json:"type,omitempty"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}
)

var automatedTestTypes = []string{
	"basic",
	"infer",
	"custom",
}

var primitives = []schema.ValueType{
	schema.TypeString,
	schema.TypeBool,
	schema.TypeInt,
	schema.TypeFloat,
}

// NewAccTestManager sets up a per provider ACC test automation manager
func NewAccTestManager(resourceProviders map[string]terraform.ResourceProvider, preCheck func(*testing.T)) *accTestManager {
	return &accTestManager{
		resourceProviders: resourceProviders,
		preCheck:          preCheck,
		testConfig:        map[string]*testConfig{},
		testableResources: map[string]*schema.Resource{},
	}
}

func (manager *accTestManager) Run(t *testing.T) {
	manager.loadManifest(t)
	for resourceKey, resourceObj := range manager.testableResources {
		configWalker := NewConfigWalker(manager.testConfig[resourceKey])
		configWalker.Run(resourceObj.Schema)
		testCases, err := configWalker.GetTestCases(resourceKey)

		for _, test := range testCases {
			t.Run(test.name, func(nested *testing.T) {
				test.testCase.PreCheck = func() { manager.preCheck(nested) }
				test.testCase.Providers = manager.resourceProviders
				resource.Test(nested, test.testCase)
			})
		}

		if err != nil {
			t.Fatalf("failed to build automated ACC tests. Error %v", err)
		}
	}
}

func loadAndParse(file string, thing interface{}, t *testing.T) {
	absPath, _ := filepath.Abs(filepath.Join("./okta/tests", file))
	raw, err := ioutil.ReadFile(absPath)

	if err != nil {
		t.Fatalf("failed to load file. Error %v", err)
	}

	err = yaml.Unmarshal(raw, thing)

	if err != nil {
		t.Fatalf("failed to parse config file. Error %v", err)
	}
}

func (manager *accTestManager) loadManifest(t *testing.T) {
	var (
		manifest map[string]string
		test     *testConfig
	)
	loadAndParse("./manifest.yaml", &manifest, t)

	for key, fileName := range manifest {
		p := manager.resourceProviders["okta"].(*schema.Provider)
		if p.ResourcesMap[key] != nil {
			manager.testableResources[key] = p.ResourcesMap[key]
			loadAndParse(fileName, &test, t)
			manager.testConfig[key] = test
		}
	}
}

func isPrimitive(propType schema.ValueType) bool {
	for _, a := range primitives {
		if a == propType {
			return true
		}
	}

	return false
}

func toJSON(input string) (string, error) {
	var v interface{}
	err := hcl.Unmarshal([]byte(input), &v)

	if err != nil {
		return "", fmt.Errorf("unable to parse HCL: %s", err)
	}

	json, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("unable to marshal JSON: %s", err)
	}

	return string(json), nil
}

func toHCL(input string) (string, error) {
	ast, err := jsonParser.Parse([]byte(input))
	if err != nil {
		return "", fmt.Errorf("unable to parse JSON: %s", err)
	}

	buf := new(bytes.Buffer)
	err = printer.Fprint(buf, ast.Node)

	return buf.String(), err
}

func wrapHCL(config string, resourceType string, resourceName string) string {
	return fmt.Sprintf(`
resource "%s" "%s" {
	%s
}`, resourceType, strings.Replace(resourceName, fmt.Sprintf("%s.", resourceType), "", -1), config)
}
