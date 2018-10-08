package okta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	jsonParser "github.com/hashicorp/hcl/json/parser"
	"github.com/hashicorp/terraform/helper/acctest"

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
		blacklist []string
		// test cases mapped to proper resources
		testCases  map[string][]*resource.TestCase
		assertions map[string]resource.TestCheckFunc
		rawConfig  map[string]interface{}
		config     string
	}

	resourceTest struct {
		checks resource.TestCheckFunc
		// Helper used to build dynamic test config. Will build this dynamically and do runtime conversions to HCL
		jsonConfig *gabs.Container
	}
)

// Hashicorp helper for this?
var primitives = []schema.ValueType{
	schema.TypeString,
	schema.TypeBool,
	schema.TypeInt,
	schema.TypeFloat,
}

// We want to grab everything in between quotes after example
// Don't care about case. We grab multiples as I would like to expand this in the future.
var exampleRx = regexp.MustCompile(`(?i).+?example:?\W?"(.+?)"`)

func (manager *accTestManager) Blacklist(resourceName string) {
	manager.blacklist = append(manager.blacklist, resourceName)
}

func (manager *accTestManager) Init(provider *schema.Provider) {
	for resourceKey, resourceObj := range provider.ResourcesMap {
		// Ignore blacklisted resources, good ol' opt out strategy, gotta test your code bro.
		if contains(manager.blacklist, resourceKey) {
			continue
		}

		configWalker := NewConfigWalker()
		configWalker.Run(resourceObj.Schema)
		conf, err := configWalker.GetConfig(resourceKey, resourceName)

		if err != nil {
			panic(fmt.Sprintf("failed to build automated ACC tests. Error %v", err))
		}
		assertions := configWalker.GetAssertions(resourceName)
	}
}

func getExample(desc string) string {
	results := exampleRx.FindAllString(desc, 0)

	// Perhaps allow multiple tests in a predictable way.
	if results != nil {
		return results[0]
	}

	return ""
}

func getStringExample(v *schema.Schema) string {
	str := getExample(v.Description)

	// Defaulting logic, not guaranteed to work. It is preferrable to pass me an example.
	if str == "" {
		switch v.Type {
		case schema.TypeString:
			return acctest.RandString(10)
		case schema.TypeBool:
			if v.Default == true {
				return "false"
			}
			return "true"
		case schema.TypeInt:
		case schema.TypeFloat:
			return string(acctest.RandInt())
		}
	}

	return str
}

func buildBasicTestCase(resourceSchema map[string]*schema.Schema) {
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
}`, resourceType, resourceName, config)
}
