terraform {
  required_version = "0.13.5"

  backend "gcs" {
    prefix = "terraform/state"
  }
}

provider "google" {
  version = "3.50.0"
  region  = var.gcp_location
}

provider "google-beta" {
  version = "3.50.0"
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

  bgp {
    asn = 64514
  }
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
// Cloud Functions
///////////////////////////////////
resource "null_resource" "cloud_function_repeater4gcsr" {
  provisioner "local-exec" {
    working_dir = "../app"
    command     = "make deploy-functions"
    environment = {
      GCP_PROJECT       = var.gcp_project
      GCSR_SSH_KEY_USER = var.gcsr_ssh_user
    }
  }

  depends_on = [
    google_vpc_access_connector.repeater4gcsr_an1,
  ]
}

data "google_cloudfunctions_function" "cloud_function_repeater4gcsr" {
  project = data.google_project.project.name
  name    = "repeater4gcsr"
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

  cloud_function {
    function = data.google_cloudfunctions_function.cloud_function_repeater4gcsr.name
  }
}

// create backend service
resource "google_compute_backend_service" "repeater4gcsr_backend" {
  provider = google-beta
  project  = data.google_project.project.name
  name     = "repeater4gcsr-backend"

  protocol    = "HTTPS"
  timeout_sec = 30

  backend {
    group = google_compute_region_network_endpoint_group.repeater4gcsr_neg.id
  }

  security_policy = google_compute_security_policy.allow_bitbucket_cloud_public_ips.name
}

// url-map
resource "google_compute_url_map" "repeater4gcsr_urlmap" {
  project         = data.google_project.project.name
  name            = "repeater4gcsr-urlmap"
  default_service = google_compute_backend_service.repeater4gcsr_backend.name
}

// create frontend service
resource "google_compute_target_http_proxy" "repeater4gcsr_http" {
  project = data.google_project.project.name
  name    = "repeater4gcsr-http-proxy"
  url_map = google_compute_url_map.repeater4gcsr_urlmap.id
}

resource "google_compute_global_forwarding_rule" "repeater4gcsr_redirect" {
  project = data.google_project.project.name
  name    = "repeater4gcsr-lb-http"

  target     = google_compute_target_http_proxy.repeater4gcsr_http.id
  port_range = "80"
  ip_address = google_compute_global_address.repeater4gcsr_ingress_address.address
}
