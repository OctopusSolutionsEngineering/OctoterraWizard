data "octopusdeploy_lifecycles" "lifecycle_default_lifecycle" {
  ids          = null
  partial_name = "Default Lifecycle"
  skip         = 0
  take         = 1
}

resource "octopusdeploy_variable" "string_variable" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "String"
  name      = "RegularVariable"
  value     = "PlainText"
}

resource "octopusdeploy_variable" "sensitive_var_unscoped" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  is_sensitive = true
  name      = "SensitiveVariable"
  sensitive_value     = "Unscoped"
}

resource "octopusdeploy_variable" "sensitive_var_1" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  is_sensitive = true
  name      = "SensitiveVariable"
  sensitive_value     = "Secret1"
  scope {
    environments = [octopusdeploy_environment.development_environment.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_2" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  is_sensitive = true
  sensitive_value     = "Secret2"
  scope {
    environments = [octopusdeploy_environment.test_environment.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_3" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
  scope {
    environments = [octopusdeploy_environment.production_environment.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_4" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "String"
  name      = "SensitiveVariable"
  value     = "RegularVariable"
  is_sensitive = false
}

resource "octopusdeploy_variable" "sensitive_var_5" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
# This doesn't seem to work
#   scope {
#     actions = [octopusdeploy_deployment_process.test.step[0].action[0].id]
#   }
#   depends_on = [octopusdeploy_deployment_process.test]
}

resource "octopusdeploy_variable" "sensitive_var_6" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
  scope {
    processes = [octopusdeploy_project.deploy_frontend_project.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_7" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
  scope {
    machines = [octopusdeploy_cloud_region_deployment_target.target_region1.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_8" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
  scope {
    channels = [octopusdeploy_channel.backend_mainline.id]
  }
}

resource "octopusdeploy_variable" "sensitive_var_9" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "Sensitive"
  name      = "SensitiveVariable"
  sensitive_value     = "Secret3"
  is_sensitive = true
  scope {
    roles = ["MyRole"]
  }
}

resource "octopusdeploy_variable" "amazon_web_services_account_variable" {
  owner_id  = octopusdeploy_project.deploy_frontend_project.id
  type      = "AmazonWebServicesAccount"
  name      = "AWS"
  value     =  octopusdeploy_aws_account.account_aws_account.id
}

variable "project_name" {
  type        = string
  nullable    = false
  sensitive   = false
}

resource "octopusdeploy_project" "deploy_frontend_project" {
  auto_create_release                  = false
  default_guided_failure_mode          = "EnvironmentDefault"
  default_to_skip_if_already_installed = false
  description                          = "Test project"
  discrete_channel_release             = false
  is_disabled                          = false
  is_discrete_channel_release          = false
  is_version_controlled                = false
  lifecycle_id                         = data.octopusdeploy_lifecycles.lifecycle_default_lifecycle.lifecycles[0].id
  name                                 = var.project_name
  project_group_id                     = octopusdeploy_project_group.project_group_test.id
  tenanted_deployment_participation    = "Untenanted"
  space_id                             = var.octopus_space_id
  included_library_variable_sets       = []
  versioning_strategy {
    template = "#{Octopus.Version.LastMajor}.#{Octopus.Version.LastMinor}.#{Octopus.Version.LastPatch}.#{Octopus.Version.NextRevision}"
  }

  connectivity_policy {
    allow_deployments_to_no_targets = false
    exclude_unhealthy_targets       = false
    skip_machine_behavior           = "SkipUnavailableMachines"
  }
}

resource "octopusdeploy_deployment_process" "test" {
  project_id = octopusdeploy_project.deploy_frontend_project.id

  step {
    condition           = "Success"
    name                = "Hello World"
    package_requirement = "LetOctopusDecide"
    start_trigger       = "StartAfterPrevious"

    action {
      action_type                        = "Octopus.KubernetesRunScript"
      name                               = "Hello World"
      condition                          = "Success"
      run_on_server                      = true
      is_disabled                        = false
      can_be_used_for_project_versioning = true
      is_required                        = false
      worker_pool_id                     = ""
      properties                         = {
        "Octopus.Action.Script.ScriptBody" = "echo \"hi\""
        "Octopus.Action.KubernetesContainers.Namespace" = ""
        "OctopusUseBundledTooling" = "False"
        "Octopus.Action.Script.ScriptSource" = "Inline"
        "Octopus.Action.Script.Syntax" = "Bash"
      }

      environments          = []
      excluded_environments = []
      channels              = []
      tenant_tags           = []

      package {
        name                      = "package1"
        package_id                = "package1"
        acquisition_location      = "Server"
        extract_during_deployment = false
        feed_id                   = octopusdeploy_docker_container_registry.feed_docker.id
        properties                = { Extract = "True", Purpose = "", SelectionMode = "immediate" }
      }

      features = []
    }

    properties   = {}
    target_roles = ["eks"]
  }
}