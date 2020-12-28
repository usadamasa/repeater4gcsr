terraform {
  required_version = "0.13.5"

  backend "gcs" {
    prefix = "terraform/state"
  }
}

provider "google" {
  version = "3.51.0"
  region  = var.gcp_location
}

provider "google-beta" {
  version = "3.51.0"
  region  = var.gcp_location
}

data "google_project" "project" {
  project_id = var.gcp_project
}

///////////////////////////////////
// Permissions
///////////////////////////////////
resource "google_service_account" "repeater4gcsr" {
  project      = data.google_project.project.name
  account_id   = "repeater4gcsr"
  display_name = "Service Account for repeater4gcsr module"
  description  = "Service Account for repeater4gcsr module"
}

resource "google_project_iam_binding" "run_admin" {
  project = data.google_project.project.name
  role    = "roles/run.admin"

  members = [
    "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com",
  ]
}

resource "google_project_iam_binding" "service_account_token_creator" {
  project = data.google_project.project.name
  role    = "roles/iam.serviceAccountTokenCreator"

  members = [
    "serviceAccount:service-${data.google_project.project.number}@serverless-robot-prod.iam.gserviceaccount.com",
  ]
}

resource "google_project_iam_binding" "run_service_agent" {
  project = data.google_project.project.name
  role    = "roles/run.serviceAgent"

  members = [
    "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com",
    "serviceAccount:${google_service_account.repeater4gcsr.email}",
  ]
}

resource "google_container_registry" "registry" {
  project  = data.google_project.project.name
  location = var.gcp_location
}

resource "google_storage_bucket_iam_binding" "registry_viewer" {
  //  project = data.google_project.project.name
  role   = "roles/run.serviceAgent"
  bucket = google_container_registry.registry.bucket_self_link

  members = [
    "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com",
    "serviceAccount:${google_service_account.repeater4gcsr.email}",
  ]
}

resource "google_project_iam_binding" "iam_service_account_user" {
  project = data.google_project.project.name
  role    = "roles/iam.serviceAccountUser"

  members = [
    "serviceAccount:${data.google_project.project.number}-compute@developer.gserviceaccount.com",
    "serviceAccount:${data.google_project.project.number}@cloudbuild.gserviceaccount.com",
    "serviceAccount:${google_service_account.repeater4gcsr.email}",
  ]
}

resource "google_project_iam_binding" "secrets_accesser" {
  project = data.google_project.project.name
  role    = "roles/secretmanager.secretAccessor"

  members = [
    "serviceAccount:${google_service_account.repeater4gcsr.email}",
  ]
}

///////////////////////////////////
// Egress Networks
///////////////////////////////////

resource "google_compute_network" "repeater4gcsr" {
  project                         = data.google_project.project.name
  name                            = "repeater4gcsr"
  delete_default_routes_on_create = false
  auto_create_subnetworks         = false
}

resource "google_compute_subnetwork" "repeater4gcsr_subnet_an1" {
  project       = data.google_project.project.name
  name          = "repeater4gcsr-subnet-an1"
  network       = google_compute_network.repeater4gcsr.id
  ip_cidr_range = "10.1.0.0/16"
}

resource "google_compute_firewall" "allow_all_egress" {
  project   = data.google_project.project.name
  name      = "allow-all-egress"
  network   = google_compute_network.repeater4gcsr.name
  direction = "EGRESS"
  destination_ranges = [
    "0.0.0.0/0"
  ]

  allow {
    protocol = "all"
  }
}

resource "google_compute_router" "repeater4gcsr_router_an1" {
  project = data.google_project.project.name
  name    = "repeater4gcsr-router-an1"
  region  = google_compute_subnetwork.repeater4gcsr_subnet_an1.region
  network = google_compute_network.repeater4gcsr.id
}

resource "google_compute_address" "repeater4gcsr_nat_address" {
  project = data.google_project.project.name
  name    = "repeater4gcsr-nat-address"
}

resource "google_compute_router_nat" "repeater4gcsr_nat_an1" {
  project                            = data.google_project.project.name
  name                               = "repeater4gcsr-nat-an1"
  router                             = google_compute_router.repeater4gcsr_router_an1.name
  region                             = google_compute_router.repeater4gcsr_router_an1.region
  nat_ip_allocate_option             = "MANUAL_ONLY"
  nat_ips                            = google_compute_address.repeater4gcsr_nat_address.*.self_link
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}

resource "google_vpc_access_connector" "repeater4gcsr_an1" {
  project       = data.google_project.project.name
  name          = "repeater4gcsr-an1"
  region        = google_compute_subnetwork.repeater4gcsr_subnet_an1.region
  ip_cidr_range = "10.8.0.0/28"
  network       = google_compute_network.repeater4gcsr.name
}


///////////////////////////////////
// Cloud Run
///////////////////////////////////
data "google_container_registry_image" "repeater4gcsr" {
  project = data.google_project.project.name
//  region  = var.gcp_location
  name    = "repeater4gcsr"
}

resource "google_cloud_run_service" "repeater4gcsr" {
  project  = data.google_project.project.name
  name     = "repeater4gcsr"
  location = var.gcp_location

  traffic {
    percent         = 100
    latest_revision = true
  }

  template {
    spec {
      containers {
        image = data.google_container_registry_image.repeater4gcsr.id
      }
      container_concurrency = 1
      timeout_seconds       = 15 * 60
      service_account_name  = google_service_account.repeater4gcsr.email
    }

    metadata {
      annotations = {
        "autoscaling.knative.dev/maxScale" = 2
        "run.googleapis.com/launch-stage" : "BETA"
        //        "run.googleapis.com/vpc-access-egress" : "all"
        //        "run.googleapis.com/vpc-access-connector" = google_vpc_access_connector.repeater4gcsr_an1.name
      }
    }
  }
  autogenerate_revision_name = true

  timeouts {
    update = "3m"
  }
}

resource "google_cloud_run_service_iam_member" "cr_invoker" {
  project  = data.google_project.project.name
  service  = google_cloud_run_service.repeater4gcsr.name
  location = google_cloud_run_service.repeater4gcsr.location
  role     = "roles/run.invoker"
  // TODO:change "allAuthenticatedUsers"
  member   = "allUsers"
}

///////////////////////////////////
// Ingress Networks
///////////////////////////////////

resource "google_compute_global_address" "repeater4gcsr_ingress_address" {
  project = data.google_project.project.name
  name    = "repeater4gcsr-lb-address"
}

// create Cloud Armor
resource "google_compute_security_policy" "allow_bitbucket_cloud_public_ips" {
  project = data.google_project.project.name
  name    = "allow-bitbucket-cloud-public-ips"

  rule {
    action   = "deny(403)"
    priority = "2147483647"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = [
        "*"]
      }
    }
    description = "default rule"
  }

  rule {
    action   = "allow"
    priority = "10000"
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        // https://support.atlassian.com/bitbucket-cloud/docs/what-are-the-bitbucket-cloud-ip-addresses-i-should-use-to-configure-my-corporate-firewall/
        src_ip_ranges = [
          "104.192.136.0/21",
          "185.166.140.0/22",
          "18.205.93.0/25",
          "18.234.32.128/25",
          "13.52.5.0/25",
        ]
      }
    }
    description = "bitbucket cloud public ips"
  }
}

// Serverless NEG
resource "google_compute_region_network_endpoint_group" "repeater4gcsr_neg" {
  project               = data.google_project.project.name
  provider              = google-beta
  name                  = "repeater4gcsr-neg"
  network_endpoint_type = "SERVERLESS"
  region                = var.gcp_location

  cloud_run {
    service = google_cloud_run_service.repeater4gcsr.name
  }
}

// https://registry.terraform.io/modules/GoogleCloudPlatform/lb-http/google/latest/submodules/serverless_negs
module "repeater4gcsr-lb" {
  source  = "GoogleCloudPlatform/lb-http/google//modules/serverless_negs"
  version = "~> 4.4"

  project = data.google_project.project.name
  name    = "repeater4gcsr-lb"

  ssl = false
  //  managed_ssl_certificate_domains = ["your-domain.com"]
  https_redirect = false

  create_address = false
  address        = google_compute_global_address.repeater4gcsr_ingress_address.address

  backends = {
    default = {
      description            = null
      enable_cdn             = false
      custom_request_headers = null
      security_policy        = google_compute_security_policy.allow_bitbucket_cloud_public_ips.name


      log_config = {
        enable      = true
        sample_rate = 1.0
      }

      groups = [
        {
          group = google_compute_region_network_endpoint_group.repeater4gcsr_neg.id
        }
      ]

      iap_config = {
        enable               = false
        oauth2_client_id     = null
        oauth2_client_secret = null
      }
    }
  }
}