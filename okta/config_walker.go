package okta

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/acctest"

	"github.com/Jeffail/gabs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type (
	ConfigWalker struct {
		keys   [][]string
		config *configMap
		index  int
	}

	configMap struct {
		basic *gabs.Container
		full  *gabs.Container
	}

	testCase struct {
		name         string
		resourceName string
		testCase     resource.TestCase
	}
)

// NewConfigWalker creates a new config walker
func NewConfigWalker() *ConfigWalker {
	return &ConfigWalker{
		keys: [][]string{[]string{}},
		config: &configMap{
			basic: gabs.New(),
			full:  gabs.New(),
		},
		index: 0,
	}
}

func (c *ConfigWalker) GetTestCases(resourceType string) ([]testCase, error) {
	basicToFull, err := c.buildTestCase(resourceType, c.config.basic, c.config.full)
	if err != nil {
		return nil, err
	}

	fullToBasic, err := c.buildTestCase(resourceType, c.config.full, c.config.basic)
	if err != nil {
		return nil, err
	}

	return []testCase{basicToFull, fullToBasic}, nil
}

func (c *ConfigWalker) buildTestCase(resourceType string, configList ...*gabs.Container) (testCase, error) {
	var (
		test  testCase
		steps []resource.TestStep
	)
	id := acctest.RandInt()
	resourceName := fmt.Sprintf("%s.test-acc-%v", resourceType, id)

	for _, conf := range configList {
		raw, err := toHCL(conf.String())
		if err != nil {
			return test, err
		}
		raw = wrapHCL(raw, resourceType, resourceName)
		steps = append(steps, resource.TestStep{
			Config: raw,
			Check:  c.getAssertion(resourceName, conf),
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
		val := config.Path(strKey).String()

		if val != "" {
			arr[i] = resource.TestCheckResourceAttr(resourceName, strKey, val)
		}
	}

	return resource.ComposeTestCheckFunc(arr...)
}

// Run starts configuration test process
func (c *ConfigWalker) Run(resourceSchema map[string]*schema.Schema) {
	for key, value := range resourceSchema {
		c.keys[c.index] = []string{key}
		c.Walk(value)
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
		switch value.Type {
		case schema.TypeMap:
			// And it begins exponentially growing!
			if value.Elem != nil {
				for key, nestedValue := range value.Elem.(map[string]*schema.Schema) {
					c.keys[c.index] = append(c.keys[c.index], key)
					c.Walk(nestedValue)
					c.index++
					c.keys = append(c.keys, []string{})
				}
			}
			break
		case schema.TypeList:
		case schema.TypeSet:
			// Just default to one item in the list
			c.keys[c.index] = append(c.keys[c.index], "0")
			c.Walk(value.Elem.(*schema.Schema))
			c.index++
			c.keys = append(c.keys, []string{})
		}
	}
}
