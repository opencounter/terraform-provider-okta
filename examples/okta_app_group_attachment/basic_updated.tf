resource "okta_group" "test" {
  name = "testAcc_%[1]d"
}

resource "okta_saml_app" "test" {
  preconfigured_app = "amazon_aws"
  label             = "testAcc_%[1]d"
}

resource "okta_app_group_attachment" "test" {
  app_id = "${okta_saml_app.test.id}"
  group_id = "${okta_group.test.id}"
  profile = <<EOT
{
  "role": "[okta-prod-pup] -- platform",
  "samlRoles": [
      "[articulate-dev] -- platform",
      "[articulate] -- platform",
      "[mgmt-prod-pika] -- platform",
      "[mgmt-stage-starfish] -- platform",
      "[okta-prod-pup] -- platform",
      "[rise-prod-asp] -- platform",
      "[rise-stage-macaque] -- platform",
      "[secops-prod-imp] -- platform",
      "[vpn-prod-mole] -- platform",
      "[vpn-stage-snail] -- platform"
  ]
}
EOT
}
