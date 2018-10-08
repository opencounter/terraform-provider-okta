package okta

import (
	"strings"
	"testing"

	"github.com/Jeffail/gabs"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

type (
	configWalker struct {
		keys   [][]string
		config *configMap
		index  int
	}

	configMap struct {
		basic *gabs.Container
		full  *gabs.Container
	}
)

// NewConfigWalker creates a new config walker
func NewConfigWalker() *configWalker {
	return &configWalker{
		keys: [][]string{},
		config: &configMap{
			basic: gabs.New(),
			full:  gabs.New(),
		},
		index: 0,
	}
}

func (c *configWalker) GetTestCases(resourceType string, resourceName string) ([]resource.TestCase, error) {
	testCases := []resource.TestCase{}
	basic, err := toHCL(c.config.basic.String())
	if err != nil {
		return configObj, err
	}
	basic = wrapHCL(basic, resourceType, resourceName)
	basicAssertions := c.getAssertion(resourceName, c.config.basic)
	full, err := toHCL(c.config.full.String())
	if err != nil {
		return configObj, err
	}
	full = wrapHCL(full, resourceType, resourceName)
	fullAssertions := c.getAssertion(resourceName, c.config.full)

	basicTestStep := buildTestStep(basic, resourceName, basicAssertions)
	fullTestStep := buildTestStep(full, resourceName, fullAssertions)

	basicToFull := []resource.TestStep{
		basicTestStep,
		fullTestStep,
	}
	fullToBasic := []resource.TestStep{
		basicTestStep,
		fullTestStep,
	}
	testCases := buildTestCase()

	return testCases, err
}

func buildTestStep(config string, resourceName string, check resource.TestCheckFunc) resource.TestStep {
	return resource.TestStep{
		Config: config,
		Check:  check,
	}
}

func buildTestCase(t testing.T, resourceType string, checkDestroy resource.TestCheckFunc, steps []resource.TestStep) resource.TestCase {
	ri := acctest.RandInt()
	resourceName := buildResourceFQN(resourceType, ri)

	return resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: checkDestroy,
		Steps:        steps,
	}
}

func (c *configWalker) GetAssertions(resourceName string) map[string]resource.TestCheckFunc {
	return map[string]resource.TestCheckFunc{
		"basic": ,
		"full":  c.getAssertion(resourceName, c.config.full),
	}
}

func (c *configWalker) getAssertion(resourceName string, config *gabs.Container) resource.TestCheckFunc {
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
func (c *configWalker) Run(resourceSchema map[string]*schema.Schema) {
	for key, value := range resourceSchema {
		c.keys[c.index] = []string{key}
		c.Walk(value)
	}
}

// Walk recursively walks config. Builds keys based on current index, every branch of possible config value gets
// a new key. Each key is a slice of strings which eventually will be joined with ".".
func (c *configWalker) Walk(value *schema.Schema) {
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
			for key, nestedValue := range value.Elem.(map[string]*schema.Schema) {
				c.keys[c.index] = append(c.keys[c.index], key)
				c.Walk(nestedValue)
				c.index++
			}
			break
		case schema.TypeList:
		case schema.TypeSet:
			// Just default to one item in the list
			c.keys[c.index] = append(c.keys[c.index], "0")
			c.Walk(value.Elem.(*schema.Schema))
			c.index++
		}
	}
}
