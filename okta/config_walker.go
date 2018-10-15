package okta

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/hashicorp/terraform/helper/acctest"

	"github.com/Jeffail/gabs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type (
	ConfigWalker struct {
		keys       [][]string
		config     *configMap
		index      int
		testConfig *testConfig
	}

	configMap struct {
		basic        *gabs.Container
		full         *gabs.Container
		dynamicTests []*gabs.Container
	}

	testCase struct {
		name         string
		resourceName string
		testCase     resource.TestCase
	}
)

// NewConfigWalker creates a new config walker
func NewConfigWalker(config *testConfig) *ConfigWalker {
	return &ConfigWalker{
		keys: [][]string{},
		config: &configMap{
			basic: gabs.New(),
			full:  gabs.New(),
		},
		index:      0,
		testConfig: config,
	}
}

// GetTestCases builds test cases after walking resource schemas
func (c *ConfigWalker) GetTestCases(resourceType string) ([]testCase, error) {
	var cases []testCase

	for _, config := range c.testConfig.Tests {
		tCase, err := c.buildTestCase(resourceType, config, c.testConfig.Properties)
		if err != nil {
			return cases, err
		}
		cases = append(cases, tCase)
	}

	return cases, nil
}

func (c *ConfigWalker) buildTestCase(resourceType string, conf *testCaseConfig, propOverrides map[string]interface{}) (testCase, error) {
	var (
		test  testCase
		steps []resource.TestStep
	)
	id := acctest.RandInt()
	resourceName := fmt.Sprintf("%s.test-acc-%v", resourceType, id)

	for _, step := range conf.Steps {
		copiedConf, err := gabs.ParseJSON([]byte(c.config.full.String()))
		if err != nil {
			return test, err
		}
		setOverrides(copiedConf, step, propOverrides)

		raw, err := toHCL(copiedConf.String())
		if err != nil {
			return test, err
		}
		raw = wrapHCL(raw, resourceType, resourceName)
		steps = append(steps, resource.TestStep{
			Config: raw,
			Check:  c.getAssertion(resourceName, copiedConf),
		})
	}

	return testCase{
		name:         fmt.Sprintf("TestAutomatedAcc_%s", resourceType),
		resourceName: resourceName,
		testCase: resource.TestCase{
			Steps: steps,
		},
	}, nil
}

func setOverrides(conf *gabs.Container, step *stepConfig, propOverrides map[string]interface{}) {
	for key, val := range propOverrides {
		conf.Set(val, key)
	}

	for key, val := range step.Properties {
		conf.Set(val, key)
	}

	a := conf.Path("login").String()
	fmt.Print(a)
	runtime.Breakpoint()
}

func buildTestStep(config string, check resource.TestCheckFunc) resource.TestStep {
	return resource.TestStep{
		Config: config,
		Check:  check,
	}
}

func (c *ConfigWalker) getAssertion(resourceName string, config *gabs.Container) resource.TestCheckFunc {
	arr := make([]resource.TestCheckFunc, len(c.keys))

	for i, arrKey := range c.keys {
		strKey := strings.Join(arrKey, ".")
		var val string
		_ = json.Unmarshal([]byte(config.Path(strKey).String()), &val)

		if val != "" {
			arr[i] = resource.TestCheckResourceAttr(resourceName, strKey, val)
		}
	}

	return resource.ComposeTestCheckFunc(arr...)
}

// Run starts configuration test process
func (c *ConfigWalker) Run(resourceSchema map[string]*schema.Schema) {
	for key, value := range resourceSchema {
		c.keys = append(c.keys, []string{key})
		c.Walk(value)
		c.index++
	}
}

// Walk recursively walks config. Builds keys based on current index, every branch of possible config value gets
// a new key. Each key is a slice of strings which eventually will be joined with ".".
func (c *ConfigWalker) Walk(value *schema.Schema) {
	if isPrimitive(value.Type) {
		if value.Computed != true {
			exampleValue := getStringExample(value)
			c.config.full.Set(exampleValue, c.keys[c.index]...)

			if value.Required == true {
				c.config.basic.Set(exampleValue, c.keys[c.index]...)
			}
		}
	} else {
		if value.Type == schema.TypeMap {
			// And it begins exponentially growing!
			if value.Elem != nil {
				for key, nestedValue := range value.Elem.(schema.Resource).Schema {
					c.keys[c.index] = append(c.keys[c.index], key)
					c.Walk(nestedValue)
					c.index++
					c.keys = append(c.keys, []string{})
				}
			}
		} else if value.Type == schema.TypeList || value.Type == schema.TypeSet {
			// // Just default to one item in the list
			// c.config.full.Array(c.keys[c.index]...)
			// c.config.basic.Array(c.keys[c.index]...)
			// c.keys[c.index] = append(c.keys[c.index], "0")
			// c.Walk(value.Elem.(*schema.Schema))
			// c.index++
			// c.keys = append(c.keys, []string{})
		}
	}
}

func getStringExample(v *schema.Schema) string {
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

	return ""
}
