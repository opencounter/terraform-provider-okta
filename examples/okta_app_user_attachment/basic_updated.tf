resource "okta_user" "test" {
  first_name  = "testAcc"
  last_name   = "blah"
  login       = "testAcc-%[1]d@testing.com"
  email       = "testAcc-%[1]d@testing.com"
}

resource "okta_saml_app" "test" {
  preconfigured_app = "amazon_aws"
  label             = "testAcc_%[1]d"
}


resource "okta_app_user_attachment" "test" {
  app_id = "${okta_saml_app.test.id}"
  user_id = "${okta_user.test.id}"
  username = "${okta_user.test.login}"
}
