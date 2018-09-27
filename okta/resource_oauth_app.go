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
			"label": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Application label.",
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validation.StringInSlice(validApplicationTypeList, false),
				Required:     true,
				Description:  "The type of client application.",
			},
			"client_id": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "OAuth client ID.",
			},
			"client_secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
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
			// "Early access" properties.. looks to be in beta which requires opt-in per account
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
	_, _, err := client.Application.CreateApplication(app, params)

	if err != nil {
		return err
	}

	d.SetId(app.Id)

	return resourceGroupRead(d, m)
}

func resourceOAuthAppRead(d *schema.ResourceData, m interface{}) error {
	app, err := fetchApp(d, m)

	if err != nil {
		return err
	}

	d.Set("name", app.Name)
	d.Set("label", app.Label)
	d.Set("type", app.Settings.OauthClient.ApplicationType)
	d.Set("client_id", app.Credentials.OauthClient.ClientId)
	d.Set("client_secret", app.Credentials.OauthClient.ClientSecret)
	d.Set("token_endpoint_auth_method", app.Credentials.OauthClient.TokenEndpointAuthMethod)
	d.Set("auto_key_rotation", app.Credentials.OauthClient.AutoKeyRotation)
	d.Set("consent_method", app.Settings.OauthClient.ConsentMethod)
	d.Set("client_uri", app.Settings.OauthClient.ClientUri)
	d.Set("logo_uri", app.Settings.OauthClient.LogoUri)
	d.Set("tos_uri", app.Settings.OauthClient.TosUri)
	d.Set("policy_uri", app.Settings.OauthClient.PolicyUri)

	return setNonPrimitives(d, map[string]interface{}{
		"redirect_uris":  app.Settings.OauthClient.RedirectUris,
		"response_types": app.Settings.OauthClient.ResponseTypes,
		"grant_types":    app.Settings.OauthClient.GrantTypes,
	})
}

func resourceOAuthAppUpdate(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	app := buildOAuthApp(d, m)

	_, _, err := client.Application.UpdateApplication(d.Id(), app)
	if err != nil {
		return err
	}

	return resourceGroupRead(d, m)
}

func resourceOAuthAppDelete(d *schema.ResourceData, m interface{}) error {
	client := getOktaClientFromMetadata(m)
	_, err := client.Application.DeleteApplication(d.Id())

	return err
}

func fetchApp(d *schema.ResourceData, m interface{}) (*okta.OpenIdConnectApplication, error) {
	client := getOktaClientFromMetadata(m)
	params := &query.Params{}
	newApp := &okta.OpenIdConnectApplication{}
	_, response, err := client.Application.GetApplication(d.Id(), newApp, params)

	// We don't want to consider a 404 an error in some cases and thus the delineation
	if response.StatusCode == 404 {
		return nil, nil
	}

	return newApp, err
}

func buildOAuthApp(d *schema.ResourceData, m interface{}) *okta.OpenIdConnectApplication {
	app := &okta.OpenIdConnectApplication{
		Name:  d.Get("name").(string),
		Label: d.Get("label").(string),
		Credentials: &okta.OAuthApplicationCredentials{
			OauthClient: &okta.ApplicationCredentialsOAuthClient{
				AutoKeyRotation:         d.Get("auto_key_rotation").(*bool),
				ClientId:                d.Get("client_id").(string),
				ClientSecret:            d.Get("client_secret").(string),
				TokenEndpointAuthMethod: d.Get("token_endpoint_auth_method").(string),
			},
		},
		Settings: &okta.OpenIdConnectApplicationSettings{
			OauthClient: &okta.OpenIdConnectApplicationSettingsClient{
				ApplicationType: d.Get("type").(string),
				ClientUri:       d.Get("client_uri").(string),
				ConsentMethod:   d.Get("consent_method").(string),
				GrantTypes:      convertInterfaceToStringArr(d.Get("GrantTypes").(string)),
				LogoUri:         d.Get("logo_uri").(string),
				PolicyUri:       d.Get("policy_uri").(string),
				RedirectUris:    convertInterfaceToStringArr(d.Get("redirect_uris")),
				ResponseTypes:   convertInterfaceToStringArr(d.Get("response_types")),
				TosUri:          d.Get("tos_uri").(string),
			},
		},
	}

	return app
}
