package okta

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

type (
	// TestManager interface for setting up and running tests.
	TestManager interface {
		AddStep(string, *regexp.Regexp) TestManager
		AddStepFromPath(string, *regexp.Regexp) TestManager
		Run(*testing.T) TestManager
		Reset() TestManager
		buildTestCase([]*TestStep, *testing.T) resource.TestCase
	}

	// AccTestManager manages ACC tests. This was built in an effort to decrease repeat code
	// and make writing ACC tests dead simple.
	AccTestManager struct {
		ResourceName string
		ResourceType string
		Steps        []*TestStep
		TestID       int
		CheckDestory resource.TestCheckFunc
	}

	TestStep struct {
		Config           string
		ExpectError      *regexp.Regexp
		MarshalledConfig map[string]interface{}
	}

	AssertionTuple struct {
		Key   string
		Value string
	}
)

func NewTestManager(resourceType string, checkUpstream CheckUpstream) TestManager {
	testID := acctest.RandInt()
	resourceName := buildResourceFQN(resourceType, testID)
	checkDestroy := createCheckResourceDestroy(resourceType, checkUpstream)

	return &AccTestManager{
		ResourceName: resourceName,
		ResourceType: resourceType,
		TestID:       testID,
		CheckDestory: checkDestroy,
	}
}

func (step *TestStep) Run(t *testing.T) {
	testCase := resource.TestCase{}
	resource.Test(t, testCase)
}

func (step *TestStep) toResourceStep(resourceName string) {
	resourceStep := resource.TestStep{
		Config: step.Config,
	}

	if step.ExpectError != nil {
		resourceStep.ExpectError = step.ExpectError
	}

	resourceStep.Check = step.buildCheck(resourceName)
}

// Uses reflection on marshalled config to do resource attribute checks
func (step *TestStep) buildCheck(resourceName string) resource.TestCheckFunc {
	assertions := []*AssertionTuple{}

	config := reflect.ValueOf(step.MarshalledConfig)
	buildAssertions(assertions, config)
	args := make([]resource.TestCheckFunc, len(assertions))

	for i, tuple := range assertions {
		args[i] = resource.TestCheckResourceAttr(resourceName, tuple.Key, tuple.Value)
	}

	return resource.ComposeTestCheckFunc(args...)
}

func buildAssertions(assertions *AssertionTuple, origingalVal interface{}) {
	value := reflect.ValueOf(originalVal)
	switch value.Kind() {
	case reflect.Slice:
		for i := 0; i < original.Len(); i += 1 {
		}

	// If it is a map we create a new map and translate each value
	case reflect.Map:
		copy.Set(reflect.MakeMap(original.Type()))
		for _, key := range original.MapKeys() {
		}

	case reflect.String:
		assertions
	}
}

func (manager *AccTestManager) buildTestCase(steps []*TestStep, t *testing.T) resource.TestCase {
	testCase := resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: manager.CheckDestory,
		Steps:        []resource.TestStep{},
	}

	for _, step := range manager.Steps {
		testCase.Steps = append(testCase.Steps, step.toResourceStep(manager.ResourceName))
	}

	return testCase
}

func (manager *AccTestManager) Run(t *testing.T) TestManager {
	testCase := manager.buildTestCase()
	resource.Test(t, testCase)

	return manager
}

func (manager *AccTestManager) Reset() TestManager {
	manager.Steps = make([]*TestStep, 0)

	return manager
}

// AddStep adds test step with provided HCL config. If an error is provided it will not assert
func (manager *AccTestManager) AddStep(rawConfig string, expectErr *regexp.Regexp) TestManager {
	var config interface{}
	jsonConfig, err := toJSON(rawConfig)

	if err != nil {
		panic(fmt.Sprintf("failed to marshall HCL config to JSON config for test type: %s", manager.ResourceType))
	}

	err = json.Unmarshal(jsonConfig, config)

	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal JSON from HCL config, test type %s", manager.ResourceType))
	}
	testStep := TestStep{
		Config:           rawConfig,
		MarshalledConfig: config,
	}

	if expectErr != nil {
		testStep.ExpectError = expectErr
	}

	manager.Steps = append(manager.Steps, &testStep)

	return manager
}

// AddStepFromPath supports JSON or HCL files.
func (manager *AccTestManager) AddStepFromPath(path string, expectErr *regexp.Regexp) TestManager {
	isHCL := true

	if strings.HasSuffix(".json") {
		isHCL = false
	} else if !strings.HasSuffix(".hcl") {
		panic(fmt.Sprintf("test manager only supports *.json and *.hcl files, was provided: %s", path))
	}

	rawConfig, err := ioutil.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("failed to read provided test config file: %s, error: %s", path, err))
	}

	if !isHCL {
		rawConfig, err = toHCL(string(rawConfig))

		if err != nil {
			panic(fmt.Sprintf("failed to marshal JSON contents to HCL: %s, error: %s", path, err))
		}
	}

	return manager.AddStep(string(rawConfig), testErr)
}

func toJSON(input string) (string, error) {
	var v interface{}
	err = hcl.Unmarshal(input, &v)
	if err != nil {
		return fmt.Errorf("unable to parse HCL: %s", err)
	}

	json, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("unable to marshal JSON: %s", err)
	}

	return string(json), nil
}

func toHCL(input string) (string, error) {
	ast, err := jsonParser.Parse([]byte(input))
	if err != nil {
		return fmt.Errorf("unable to parse JSON: %s", err)
	}

	return ast, nil
}
