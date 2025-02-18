resource "octopusdeploy_library_variable_set" "octopus_library_variable_set" {
  name = "Test"
  description = "Test variable set"
}

resource "octopusdeploy_variable" "string_variable" {
  owner_id  = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  type      = "String"
  name      = "RegularVariable"
  value     = "PlainText"
}

resource "octopusdeploy_variable" "deliberate_collision" {
  owner_id  = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  type      = "String"
  name      = "Test.SecretVariable_Unscoped"
  value     = "PlainText"
}

resource "octopusdeploy_variable" "unscoped_secret" {
  name = "Test.SecretVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  sensitive_value = "Default"
}

resource "octopusdeploy_variable" "secret" {
  name = "Test.SecretVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  sensitive_value = "Development"
  scope {
    environments = [octopusdeploy_environment.development_environment.id]
  }
}

resource "octopusdeploy_variable" "secret2" {
  name = "Test.SecretVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  sensitive_value = "Test"
  scope {
    environments = [octopusdeploy_environment.test_environment.id]
  }
}

resource "octopusdeploy_variable" "secret3" {
  name = "Test.SecretVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  sensitive_value = "Test"
  scope {
    environments = [octopusdeploy_environment.production_environment.id]
  }
}

resource "octopusdeploy_variable" "unscoped" {
  name = "Test.UnscopedVariable"
  type = "Sensitive"
  description = "Test variable"
  is_sensitive = true
  is_editable = true
  owner_id = octopusdeploy_library_variable_set.octopus_library_variable_set.id
  sensitive_value = "Unscoped"
}
