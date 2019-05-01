package okta

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/okta/okta-sdk-golang/okta"
)

var userSearchSchema = &schema.Schema{
	Type:        schema.TypeSet,
	Required:    true,
	Description: "Filter to find a user, each filter will be concatenated with an AND clause. Please be aware profile properties must match what is in Okta, which is likely camel case",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Property name to search for. This requires the search feature be on. Please see Okta documentation on their filter API for users. https://developer.okta.com/docs/api/resources/users#list-users-with-search",
			},
			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"comparison": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "eq",
				ValidateFunc: validation.StringInSlice([]string{"eq", "lt", "gt", "sw"}, true),
			},
		},
	},
}

var userProfileDataSchema = map[string]*schema.Schema{
	"admin_roles": &schema.Schema{
		Type:     schema.TypeSet,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"city": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"cost_center": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"country_code": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"custom_profile_attributes": &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
	},
	"department": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"display_name": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"division": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"email": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"employee_number": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"first_name": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"group_memberships": {
		Type:     schema.TypeSet,
		Computed: true,
		Elem:     &schema.Schema{Type: schema.TypeString},
	},
	"honorific_prefix": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"honorific_suffix": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"last_name": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"locale": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"login": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"manager": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"manager_id": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"middle_name": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"mobile_phone": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"nick_name": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"organization": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"postal_address": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"preferred_language": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"primary_phone": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"profile_url": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"second_email": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"state": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"status": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"street_address": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"timezone": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"title": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"user_type": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
	"zip_code": &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	},
}

func buildUserDataSourceSchema(target map[string]*schema.Schema) map[string]*schema.Schema {
	return buildSchema(userProfileDataSchema, target)
}

func assignAdminRolesToUser(u string, r []string, c *okta.Client) error {
	validRoles := []string{"SUPER_ADMIN", "ORG_ADMIN", "API_ACCESS_MANAGEMENT_ADMIN", "APP_ADMIN", "USER_ADMIN", "MOBILE_ADMIN", "READ_ONLY_ADMIN", "HELP_DESK_ADMIN"}

	for _, role := range r {
		if contains(validRoles, role) {
			roleStruct := okta.Role{Type: role}
			_, _, err := c.User.AddRoleToUser(u, roleStruct)

			if err != nil {
				return fmt.Errorf("[ERROR] Error Assigning Admin Roles to User: %v", err)
			}
		} else {
			return fmt.Errorf("[ERROR] %v is not a valid Okta role", role)
		}
	}

	return nil
}

func assignGroupsToUser(u string, g []string, c *okta.Client) error {
	for _, group := range g {
		_, err := c.Group.AddUserToGroup(group, u)

		if err != nil {
			return fmt.Errorf("[ERROR] Error Assigning Group to User: %v", err)
		}
	}

	return nil
}

func populateUserProfile(d *schema.ResourceData) *okta.UserProfile {
	profile := okta.UserProfile{}

	if _, ok := d.GetOk("custom_profile_attributes"); ok {
		for k, v := range d.Get("custom_profile_attributes").(map[string]interface{}) {
			profile[k] = v
		}
	}

	profile["firstName"] = d.Get("first_name").(string)
	profile["lastName"] = d.Get("last_name").(string)
	profile["login"] = d.Get("login").(string)
	profile["email"] = d.Get("email").(string)

	if _, ok := d.GetOk("city"); ok {
		profile["city"] = d.Get("city").(string)
	}

	if _, ok := d.GetOk("cost_center"); ok {
		profile["costCenter"] = d.Get("cost_center").(string)
	}

	if _, ok := d.GetOk("country_code"); ok {
		profile["countryCode"] = d.Get("country_code").(string)
	}

	if _, ok := d.GetOk("department"); ok {
		profile["department"] = d.Get("department").(string)
	}

	if _, ok := d.GetOk("display_name"); ok {
		profile["displayName"] = d.Get("display_name").(string)
	}

	if _, ok := d.GetOk("division"); ok {
		profile["division"] = d.Get("division").(string)
	}

	if _, ok := d.GetOk("employee_number"); ok {
		profile["employeeNumber"] = d.Get("employee_number").(string)
	}

	if _, ok := d.GetOk("honorific_prefix"); ok {
		profile["honorificPrefix"] = d.Get("honorific_prefix").(string)
	}

	if _, ok := d.GetOk("honorific_suffix"); ok {
		profile["honorificSuffix"] = d.Get("honorific_suffix").(string)
	}

	if _, ok := d.GetOk("locale"); ok {
		profile["locale"] = d.Get("locale").(string)
	}

	if _, ok := d.GetOk("manager"); ok {
		profile["manager"] = d.Get("manager").(string)
	}

	if _, ok := d.GetOk("manager_id"); ok {
		profile["managerId"] = d.Get("manager_id").(string)
	}

	if _, ok := d.GetOk("middle_name"); ok {
		profile["middleName"] = d.Get("middle_name").(string)
	}

	if _, ok := d.GetOk("mobile_phone"); ok {
		profile["mobilePhone"] = d.Get("mobile_phone").(string)
	}

	if _, ok := d.GetOk("nick_name"); ok {
		profile["nickName"] = d.Get("nick_name").(string)
	}

	if _, ok := d.GetOk("organization"); ok {
		profile["organization"] = d.Get("organization").(string)
	}

	// need to set profile.postalAddress to nil explicitly if not set because of a bug with this field
	// have a support ticket open with okta about it
	if _, ok := d.GetOk("postal_address"); ok {
		profile["postalAddress"] = d.Get("postal_address").(string)
	} else {
		profile["postalAddress"] = nil
	}

	if _, ok := d.GetOk("preferred_language"); ok {
		profile["preferredLanguage"] = d.Get("preferred_language").(string)
	}

	if _, ok := d.GetOk("primary_phone"); ok {
		profile["primaryPhone"] = d.Get("primary_phone").(string)
	}

	if _, ok := d.GetOk("profile_url"); ok {
		profile["profileUrl"] = d.Get("profile_url").(string)
	}

	if _, ok := d.GetOk("second_email"); ok {
		profile["secondEmail"] = d.Get("second_email").(string)
	}

	if _, ok := d.GetOk("state"); ok {
		profile["state"] = d.Get("state").(string)
	}

	if _, ok := d.GetOk("street_address"); ok {
		profile["streetAddress"] = d.Get("street_address").(string)
	}

	if _, ok := d.GetOk("timezone"); ok {
		profile["timezone"] = d.Get("timezone").(string)
	}

	if _, ok := d.GetOk("title"); ok {
		profile["title"] = d.Get("title").(string)
	}

	if _, ok := d.GetOk("user_type"); ok {
		profile["userType"] = d.Get("user_type").(string)
	}

	if _, ok := d.GetOk("zip_code"); ok {
		profile["zipCode"] = d.Get("zip_code").(string)
	}

	return &profile
}

func setAdminRoles(d *schema.ResourceData, c *okta.Client) error {
	// set all roles currently attached to user in state
	roles, _, err := c.User.ListAssignedRoles(d.Id(), nil)

	if err != nil {
		return err
	}

	roleTypes := make([]interface{}, 0)
	for _, role := range roles {
		roleTypes = append(roleTypes, role.Type)
	}

	// set the custom_profile_attributes values
	return setNonPrimitives(d, map[string]interface{}{
		"admin_roles": schema.NewSet(schema.HashString, roleTypes),
	})
}

func setGroups(d *schema.ResourceData, c *okta.Client) error {
	// set all groups currently attached to user in state
	groups, _, err := c.User.ListUserGroups(d.Id(), nil)
	if err != nil {
		return err
	}

	groupIds := make([]interface{}, 0)

	// ignore saving the Everyone group into state so we don't end up with perpetual diffs
	for _, group := range groups {
		if group.Profile.Name != "Everyone" {
			groupIds = append(groupIds, group.Id)
		}
	}

	// set the custom_profile_attributes values
	return setNonPrimitives(d, map[string]interface{}{
		"group_memberships": schema.NewSet(schema.HashString, groupIds),
	})
}

func isCustomUserAttr(key string) bool {
	return !contains(profileKeys, key)
}

func flattenUser(u *okta.User) map[string]interface{} {
	customAttributes := make(map[string]interface{})
	attrs := map[string]interface{}{}

	for k, v := range *u.Profile {
		if v != nil {
			attrKey := camelCaseToUnderscore(k)

			if isCustomUserAttr(attrKey) {
				customAttributes[k] = v
			} else {
				attrs[attrKey] = v
			}
		}
	}
	attrs["custom_profile_attributes"] = customAttributes

	return attrs
}

func setUserProfileAttributes(d *schema.ResourceData, u *okta.User) error {
	// any profile attributes that aren't explicitly outlined in the okta_user schema
	// (ie. first_name) can be considered customAttributes
	customAttributes := make(map[string]interface{})

	// set all the attributes in state that were returned from user.Profile
	for k, v := range *u.Profile {
		if v != nil {
			attribute := camelCaseToUnderscore(k)
			if err := d.Set(attribute, v); err != nil {
				if strings.Contains(err.Error(), "Invalid address to set") {
					customAttributes[k] = v
				} else {
					return fmt.Errorf("error setting %s for resource %s: %s", attribute, d.Id(), err)
				}
			}
		}
	}

	// set the custom_profile_attributes values
	return setNonPrimitives(d, map[string]interface{}{
		"custom_profile_attributes": customAttributes,
	})
}

// need to remove from all current admin roles and reassign based on terraform configs when a change is detected
func updateAdminRolesOnUser(u string, r []string, c *okta.Client) error {
	roles, _, err := c.User.ListAssignedRoles(u, nil)

	if err != nil {
		return fmt.Errorf("[ERROR] Error Updating Admin Roles On User: %v", err)
	}

	for _, role := range roles {
		_, err := c.User.RemoveRoleFromUser(u, role.Id)

		if err != nil {
			return fmt.Errorf("[ERROR] Error Updating Admin Roles On User: %v", err)
		}
	}

	err = assignAdminRolesToUser(u, r, c)

	if err != nil {
		return err
	}

	return nil
}

// need to remove from all current groups and reassign based on terraform configs when a change is detected
func updateGroupsOnUser(u string, g []string, c *okta.Client) error {
	groups, _, err := c.User.ListUserGroups(u, nil)

	if err != nil {
		return fmt.Errorf("[ERROR] Error Updating Groups On User: %v", err)
	}

	for _, group := range groups {
		if group.Profile.Name != "Everyone" {
			_, err := c.Group.RemoveGroupUser(group.Id, u)

			if err != nil {
				return fmt.Errorf("[ERROR] Error Updating Groups On User: %v", err)
			}
		}
	}

	if err = assignGroupsToUser(u, g, c); err != nil {
		return err
	}

	return nil
}

// handle setting of user status based on what the current status is because okta
// only allows transitions to certain statuses from other statuses - consult okta User API docs for more info
// https://developer.okta.com/docs/api/resources/users#lifecycle-operations
func updateUserStatus(u string, d string, c *okta.Client) error {
	user, _, err := c.User.GetUser(u)

	if err != nil {
		return err
	}

	var statusErr error
	switch d {
	case "SUSPENDED":
		_, statusErr = c.User.SuspendUser(u)
	case "DEPROVISIONED":
		_, statusErr = c.User.DeactivateUser(u)
	case "ACTIVE":
		if user.Status == "SUSPENDED" {
			_, statusErr = c.User.UnsuspendUser(u)
		} else if user.Status == "PASSWORD_EXPIRED" {
			// Ignore password expired status. This status is already activated.
			return nil
		} else {
			_, _, statusErr = c.User.ActivateUser(u, nil)
		}
	}

	if statusErr != nil {
		return statusErr
	}

	err = waitForStatusTransition(u, c)

	if err != nil {
		return err
	}

	return nil
}

// need to wait for user.TransitioningToStatus field to be empty before allowing Terraform to continue
// so the proper current status gets set in the state during the Read operation after a Status update
func waitForStatusTransition(u string, c *okta.Client) error {
	user, _, err := c.User.GetUser(u)

	if err != nil {
		return err
	}

	for {
		if user.TransitioningToStatus == "" {
			return nil
		} else {
			log.Printf("[INFO] Transitioning to status = %v; waiting for 5 more seconds...", user.TransitioningToStatus)
			time.Sleep(5 * time.Second)

			user, _, err = c.User.GetUser(u)

			if err != nil {
				return err
			}
		}
	}
}
