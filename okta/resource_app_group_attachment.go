package okta

import (
	"encoding/json"
	"fmt"

	"github.com/okta/okta-sdk-golang/okta"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAppGroupAttachment() *schema.Resource {
	return &schema.Resource{
		Create:   resourceAppGroupAttachmentCreate,
		Read:     resourceAppGroupAttachmentRead,
		Delete:   resourceAppGroupAttachmentDelete,
		Exists:   resourceAppGroupAttachmentExists,
		Importer: createNestedResourceImporter([]string{"app_id", "group_id"}),
		// For those familiar with Terraform schemas be sure to check the base application schema and/or
		// the examples in the documentation
		Schema: map[string]*schema.Schema{
			"app_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of application to associate group with.",
				Required:    true,
				ForceNew:    true,
			},
			"group_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of group to associate with app.",
				Required:    true,
				ForceNew:    true,
			},
			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"profile_json": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Profile settings associated with the group/app association. Due to the great variety of possible settings here, just using JSON for this.",
				ForceNew:     true,
				ValidateFunc: validateDataJSON,
				StateFunc:    normalizeDataJSON,
			},
		},
	}
}

func resourceAppGroupAttachmentExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := getOktaClientFromMetadata(m)
	resource, _, err := client.Application.GetApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string), nil)

	return resource != nil && resource.Id != "", err
}

func resourceAppGroupAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	var profile interface{}
	if val, ok := d.GetOk("profile"); ok {
		// Error handling done in the schema, no need here.
		json.Unmarshal([]byte(val.(string)), &profile)
	}
	body := okta.ApplicationGroupAssignment{
		Profile: profile,
	}
	client := getOktaClientFromMetadata(m)
	_, _, err := client.Application.CreateApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string), body)
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprintf("%s/%s", d.Get("app_id").(string), d.Get("group_id").(string)))
	return resourceAppGroupAttachmentRead(d, m)
}

func resourceAppGroupAttachmentRead(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	grp, _, err := client.Application.GetApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string), nil)
	if err != nil {
		return err
	}
	if grp.Profile != nil {
		jsonProfile, err := json.Marshal(grp.Profile)
		if err != nil {
			return fmt.Errorf("Failed to parse app group attachment profile, error: %s", err)
		}
		d.Set("profile", string(jsonProfile))
	}
	d.Set("priority", grp.Priority)

	return nil
}

func resourceAppGroupAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	_, err := client.Application.DeleteApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string))
	return err
}
