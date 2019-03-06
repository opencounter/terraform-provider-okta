package okta

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/okta/okta-sdk-golang/okta"
)

func resourceAppUserAttachment() *schema.Resource {
	return &schema.Resource{
		Create:   resourceAppUserAttachmentCreate,
		Read:     resourceAppUserAttachmentRead,
		Delete:   resourceAppUserAttachmentDelete,
		Exists:   resourceAppUserAttachmentExists,
		Update:   resourceAppUserAttachmentUpdate,
		Importer: createNestedResourceImporter([]string{"app_id", "user_id"}),
		// For those familiar with Terraform schemas be sure to check the base application schema and/or
		// the examples in the documentation
		Schema: map[string]*schema.Schema{
			"app_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of application to associate user with.",
				Required:    true,
				ForceNew:    true,
			},
			"user_id": &schema.Schema{
				Type:        schema.TypeString,
				Description: "ID of user to associate with app.",
				Required:    true,
				ForceNew:    true,
			},
			"scope": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"username": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "This will only be set if it is configured. It will be stored in clear text in state.",
			},
		},
	}
}

func resourceAppUserAttachmentExists(d *schema.ResourceData, m interface{}) (bool, error) {
	client := getOktaClientFromMetadata(m)
	resource, _, err := client.Application.GetApplicationUser(d.Get("app_id").(string), d.Get("user_id").(string), nil)

	return resource != nil && resource.Id != "", err
}

func resourceAppUserAttachmentCreate(d *schema.ResourceData, m interface{}) error {
	err := assignUserToApp(
		d.Get("app_id").(string),
		d.Get("user_id").(string),
		d.Get("username").(string),
		d.Get("password").(string),
		getOktaClientFromMetadata(m),
	)
	if err != nil {
		return err
	}
	d.SetId(fmt.Sprintf("%s/%s", d.Get("app_id").(string), d.Get("user_id").(string)))
	return resourceAppUserAttachmentRead(d, m)
}

func resourceAppUserAttachmentRead(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	user, _, err := client.Application.GetApplicationUser(d.Get("app_id").(string), d.Get("user_id").(string), nil)
	if err != nil {
		return err
	}
	d.Set("scope", user.Scope)
	d.Set("username", user.Credentials.UserName)
	if user.Credentials.Password != nil && user.Credentials.Password.Value != "" {
		d.Set("password", user.Credentials.Password)
	}

	return nil
}

func resourceAppUserAttachmentUpdate(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	_, _, err := client.Application.UpdateApplicationUser(
		d.Get("app_id").(string),
		d.Get("user_id").(string),
		okta.AppUser{
			Id: d.Get("user_id").(string),
			Credentials: &okta.AppUserCredentials{
				UserName: d.Get("username").(string),
				Password: &okta.AppUserPasswordCredential{
					Value: d.Get("password").(string),
				},
			},
		},
	)

	if err != nil {
		return err
	}

	return resourceAppUserAttachmentRead(d, m)
}

func resourceAppUserAttachmentDelete(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	_, err := client.Application.DeleteApplicationUser(d.Get("app_id").(string), d.Get("user_id").(string))
	return err
}
