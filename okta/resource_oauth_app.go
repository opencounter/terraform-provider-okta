package okta

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/okta/okta-sdk-golang/okta"
	"github.com/okta/okta-sdk-golang/okta/query"
)

var validAuthMethodList = []string{"none", "client_secret_post", "client_secret_basic", "client_secret_jwt"}
var validResponseTypeList = []string{"code", "token", "id_token"}
var validGrantTypeList = []string{"authorization_code", "implicit", "password", "refresh_token", "client_credentials"}
var validApplicationTypeList = []string{"web", "native", "browser", "service"}

func resourceOAuthApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceOAuthAppCreate,
		Read:   resourceOAuthAppRead,
		Update: resourceOAuthAppUpdate,
		Delete: resourceOAuthAppDelete,
		Exists: resourceOAuthAppExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of resource.",
			},
			"type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Type of OAuth application this resource represents.",
			},
			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OAuth client ID.",
			},
			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OAuth client secret key.",
			},
			"token_endpoint_auth_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(validAuthMethodList, false),
				Default:      "client_secret_basic",
				Description:  "Requested authentication method for the token endpoint.",
			},
			"auto_key_rotation": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Requested key rotation mode.",
			},
			"client_uri": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "URI to a web page providing information about the client.",
			},
			"logo_uri": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "URI that references a logo for the client.",
			},
			"redirect_uris": &schema.Schema{
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "List of URIs for use in the redirect-based flow.",
			},
			"response_types": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(validResponseTypeList, false)},
				Optional:    true,
				Description: "List of OAuth 2.0 response type strings.",
			},
			"grant_types": &schema.Schema{
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringInSlice(validResponseTypeList, false)},
				Required:    true,
				Description: "List of OAuth 2.0 grant type strings.",
			},
			"initiate_login_uri": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "URI that a third party can use to initiate a login by the client.",
			},
			"application_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice(validApplicationTypeList, false),
				Description:  "The type of client application.",
			},
			// "Early access" properties.. looks to be in beta which requires opt-in per account
			"issuer_mode": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"CUSTOM_URL", "ORG_URL"}, false),
				Description:  "*Beta Okta Property*. Indicates whether the Okta Authorization Server uses the original Okta org domain URL or a custom domain URL as the issuer of ID token for this client.",
			},
			"tos_uri": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "*Beta Okta Property*. URI to web page providing client tos (terms of service).",
			},
			"policy_uri": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "*Beta Okta Property*. URI to web page providing client policy document.",
			},
			"consent_method": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"REQUIRED", "TRUSTED"}, false),
				Default:      "TRUSTED",
				Description:  "*Beta Okta Property*. Indicates whether user consent is required or implicit. Valid values: REQUIRED, TRUSTED. Default value is TRUSTED",
			},
		},
	}
}

func resourceOAuthAppExists(d *schema.ResourceData, m interface{}) (bool, error) {
	app, err := fetchApp(d, m)

	// Not sure if a non-nil app with an empty ID is possible but checking to avoid false positives.
	return app != nil && app.Id != "", err
}

func resourceOAuthAppCreate(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	params := &query.Params{}
	app := buildOAuthApp(d, m)
	app, _, err := client.Application.CreateApplication(*app, params)

	if err != nil {
		return err
	}

	return resourceGroupRead(d, m)
}

func resourceOAuthAppRead(d *schema.ResourceData, m interface{}) error {
	app, err := fetchApp(d, m)

	if err != nil {
		return err
	}

	if app == nil || app.Id == "" {
		d.SetId("")
		return nil
	}

	return err
}

func resourceOAuthAppUpdate(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	params := &query.Params{}
	app := buildOAuthApp(d, m)

	client.Application.UpdateApplication(d.Get("id").(string), *app, params)
	return resourceGroupRead(d, m)
}

func resourceOAuthAppDelete(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	params := &query.Params{}
	_, err := client.Application.DeleteApplication(d.Get("id").(string), params)

	return err
}

func fetchApp(d *schema.ResourceData, m interface{}) (*okta.Application, error) {
	client := getOktaClientFromMetadata(m)
	params := &query.Params{}
	app, response, err := client.Application.GetApplication(d.Get("id").(string), params)

	// We don't want to consider a 404 an error in some cases and thus the delineation
	if response.StatusCode == 404 {
		return nil, nil
	}

	return app, err
}

func buildOAuthApp(d *schema.ResourceData, m interface{}) *okta.Application {

}
