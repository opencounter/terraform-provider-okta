package okta

import (
	"fmt"
	"strings"
	"testing"

	articulateOkta "github.com/articulate/oktasdk-go/okta"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/okta/okta-sdk-golang/okta"
	"github.com/okta/okta-sdk-golang/okta/query"
)

func deleteOAuthApps(artClient *articulateOkta.Client, client *okta.Client) error {
	appList, _, err := client.Application.ListApplications(nil)

	if err != nil {
		return err
	}

	for _, app := range appList {
		if app, ok := app.(*okta.OpenIdConnectApplication); ok {
			if strings.HasPrefix(app.Name, testResourcePrefix) {
				_, appErr := client.Application.DeleteApplication(app.Id)

				if appErr != nil {
					err = appErr
				}
			}
		}
	}

	return err
}

func TestAccOktaOAuthApplication(t *testing.T) {
	ri := acctest.RandInt()
	config := testOktaOAuthApplication(ri)
	updatedConfig := testOktaOAuthApplicationUpdated(ri)
	resourceName := buildResourceFQN(oAuthApp, ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: createCheckResourceDestroy(oAuthApp, doesAppExistUpstream),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					ensureResourceExists(resourceName, doesAppExistUpstream),
					resource.TestCheckResourceAttr(resourceName, "name", buildResourceName(ri)),
					resource.TestCheckResourceAttr(resourceName, "status", "ACTIVE"),
					resource.TestCheckResourceAttr(resourceName, "label", "Rise and shine"),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					ensureResourceExists(resourceName, doesAppExistUpstream),
					resource.TestCheckResourceAttr(resourceName, "name", buildResourceName(ri)),
					resource.TestCheckResourceAttr(resourceName, "status", "INACTIVE"),
				),
			},
		},
	})
}

func doesAppExistUpstream(id string) (bool, error) {
	client := getOktaClientFromMetadata(testAccProvider.Meta())
	newApp := &okta.OpenIdConnectApplication{}
	_, response, err := client.Application.GetApplication(id, newApp, &query.Params{})

	if err != nil {
		return false, err
	}

	// We don't want to consider a 404 an error in some cases and thus the delineation
	if response.StatusCode == 404 {
		return false, nil
	}

	return true, err
}

func testOktaOAuthApplication(rInt int) string {
	name := buildResourceName(rInt)

	return fmt.Sprintf(`
resource "%s" "%s" {
  name        = "%s"
  type		  = "web"
  status      = "ACTIVE"
  label = "Rise and shine"
}
`, oAuthApp, name, name)
}

func testOktaOAuthApplicationUpdated(rInt int) string {
	name := buildResourceName(rInt)

	return fmt.Sprintf(`
resource "%s" "%s" {
  name        = "%s"
  type		  = "browser"
  status      = "INACTIVE"
  label = "Rise and shine UPDATED"
}
`, oAuthApp, name, name)
}
