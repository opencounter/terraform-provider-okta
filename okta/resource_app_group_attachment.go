package okta

import (
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
		Schema: buildAppSchema(map[string]*schema.Schema{
			"app_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of application to associate group with.",
				ForceNew:    true,
			},
			"group_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of group to associate with app.",
				ForceNew:    true,
			},
			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"profile": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Profile settings associated with the group/app association",
				ForceNew:     true,
				ValidateFunc: validateDataJSON,
				StateFunc:    normalizeDataJSON,
			},
		}),
	}
}

func resourceAppGroupAttachmentExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := getOktaClientFromMetadata(m)
	resource, _, err := client.Application.GetApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string), nil)

	return resource != nil && resource.Id != "", err
}

func resourceAppGroupAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	return resourceAppGroupAttachmentRead(d, m)
}

func resourceAppGroupAttachmentRead(d *schema.ResourceData, m interface{}) error {
	return nil
}

func resourceAppGroupAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	_, err := client.Application.DeleteApplicationGroupAssignment(d.Get("app_id").(string), d.Get("group_id").(string))
	return err
}
