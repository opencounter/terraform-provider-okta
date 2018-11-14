package okta

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceUserSchemas() *schema.Resource {
	return &schema.Resource{
		Create: resourceUserSchemaCreate,
		Delete: resourceUserSchemaDelete,
		Exists: resourceUserSchemaExists,
		Read:   resourceUserSchemaRead,
		Update: resourceUserSchemaUpdate,
		Schema: map[string]*schema.Schema{
			"subschema": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"base", "custom"}, false),
				Description:  "SubSchema Type: base or custom",
			},
			"index": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Subschema unique string identifier",
			},
			"title": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Subschema title (display name)",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"string", "boolean", "number", "integer", "array"}, false),
				Description:  "Subschema type: string, boolean, number, integer, or array",
			},
			"arraytype": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"string", "number", "interger", "reference"}, false),
				Description:  "Subschema array type: string, number, interger, reference. Type field must be an array.",
			},
			"description": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom Subschema description",
			},
			"required": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "whether the Subschema is required, true or false. Default = false",
			},
			"minlength": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Subschema of type string minlength",
			},
			"maxlength": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Subschema of type string maxlength",
			},
			"enum": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Custom Subschema enumerated value of the property. see: developer.okta.com/docs/api/resources/schemas#user-profile-schema-property-object",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"oneof": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Custom Subschema json schemas. see: developer.okta.com/docs/api/resources/schemas#user-profile-schema-property-object",
				ValidateFunc: validateJSONString,
			},
			"permissions": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "READ_ONLY",
				ValidateFunc: validation.StringInSlice([]string{"HIDE", "READ_ONLY", "READ_WRITE"}, false),
				Description:  "SubSchema permissions: HIDE, READ_ONLY, or READ_WRITE. Default = READ_ONLY",
			},
			"master": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "PROFILE_MASTER",
				ValidateFunc: validation.StringInSlice([]string{"PROFILE_MASTER", "OKTA"}, false),
				Description:  "SubSchema profile manager: PROFILE_MASTER or OKTA. Default = PROFILE_MASTER",
			},
		},
	}
}

func resourceUserSchemaCreate(d *schema.ResourceData, m interface{}) error {
	index := d.Get("index").(string)
	log.Printf("[INFO] Creating User Schema %v", index)

	schema := d.Get("subschema").(string)
	if schema == "custom" {
		err := createUserSchema(d, m)
		if err != nil {
			return err
		}
	}
	d.SetId(index)

	return nil
}

func resourceUserSchemaDelete(d *schema.ResourceData, m interface{}) error {
	index := d.Get("index").(string)
	log.Printf("[INFO] Delete User Schema %v", index)
	client := m.(*Config).articulateOktaClient

	schema := d.Get("subschema").(string)

	if schema == "base" {
		return fmt.Errorf("[ERROR] Error you cannot delete a base subschema")
	}

	_, _, err := client.Schemas.DeleteUserCustomSubSchema(index)

	return err
}

func resourceUserSchemaExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := getClientFromMetadata(m)

	subschemas, resp, err := client.Schemas.GetUserSubSchemaIndex(d.Get("subschema").(string))
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return contains(subschemas, d.Get("index").(string)), err
}

func resourceUserSchemaRead(d *schema.ResourceData, m interface{}) error {
	index := d.Get("index").(string)
	log.Printf("[INFO] List User Schema %v", index)
	client := getClientFromMetadata(m)
	subschemas, _, err := client.Schemas.GetUserSubSchemaPropMap(d.Get("subschema").(string), index)
	if err != nil {
		return err
	}
	d.Set("subschema", subschemas)

	return nil
}

func resourceUserSchemaUpdate(d *schema.ResourceData, m interface{}) error {
	var err error
	log.Printf("[INFO] Update User Schema %v", d.Get("index").(string))

	d.Partial(true)
	schema := d.Get("subschema").(string)
	if schema == "custom" {
		err = createUserSchema(d, m)
	}
	d.Partial(false)

	return err
}

// create or modify a custom subschema
func createUserSchema(d *schema.ResourceData, m interface{}) error {
	client := getClientFromMetadata(m)

	perms := client.Schemas.Permissions()
	perms.Principal = "SELF"
	perms.Action = "READ_ONLY"

	template := client.Schemas.CustomSubSchema()
	template.Index = d.Get("index").(string)
	template.Title = d.Get("title").(string)
	template.Type = d.Get("type").(string)
	template.Master.Type = "PROFILE_MASTER"
	template.Items.Type = d.Get("arraytype").(string)
	template.Description = d.Get("description").(string)
	template.Required = d.Get("required").(bool)
	template.MinLength = d.Get("minlength").(int)
	template.MaxLength = d.Get("maxlength").(int)
	template.Master.Type = d.Get("master").(string)
	perms.Action = d.Get("permissions").(string)

	template.Permissions = append(template.Permissions, perms)
	if enum, ok := d.GetOk("enum"); ok {
		template.Enum = convertInterfaceToStringArr(enum)
	}
	if oneOf, ok := d.GetOk("oneof"); ok {
		var obj []interface{}
		err := json.Unmarshal([]byte(oneOf.(string)), &obj)
		if err != nil {
			return fmt.Errorf("[ERROR] Error decoding oneof json string %v", err)
		}
		for _, v := range obj {
			oneof := client.Schemas.OneOf()
			for k2, v2 := range v.(map[string]interface{}) {
				switch k2 {
				case "const":
					oneof.Const = v2.(string)
				case "title":
					oneof.Title = v2.(string)
				}
			}
			template.OneOf = append(template.OneOf, oneof)
		}
	}

	_, _, err := client.Schemas.UpdateUserCustomSubSchema(template)

	return err
}

// validate if oneof value is a json string
// this function lovingly lifted from the aws terraform provider
func validateJSONString(v interface{}, k string) (ws []string, errors []error) {
	if _, err := structure.NormalizeJsonString(v); err != nil {
		errors = append(errors, fmt.Errorf("[ERROR] %q contains an invalid JSON: %s", k, err))
	}
	return
}
