resource okta_group test {
  name        = "testAcc_replace_with_uuid"
  description = "Test"
}

resource okta_group test1 {
  name        = "testAcc1_replace_with_uuid"
  description = "Test"
}

resource okta_group_rule test {
  name              = "testAcc_replace_with_uuid"
  status            = "ACTIVE"
  group_assignments = ["${element(okta_group.test.*.id, count.index)}"]
  expression_type   = "urn:okta:expression:1.0"
  expression_value  = "String.stringContains(String.toLowerCase(user.login),String.toLowerCase(\"testing1@qb.com\")) OR String.stringContains(String.toLowerCase(user.login),String.toLowerCase(\"testing2@qb.com\"))"
}

resource okta_group_rule test1 {
  name              = "testAcc1_replace_with_uuid"
  status            = "ACTIVE"
  group_assignments = ["${element(okta_group.test1.*.id, count.index)}"]
  expression_type   = "urn:okta:expression:1.0"
  expression_value  = "String.stringContains(String.toLowerCase(user.login),String.toLowerCase(\"testing1@qb.com\")) OR String.stringContains(String.toLowerCase(user.login),String.toLowerCase(\"testing2@qb.com\"))"
}
