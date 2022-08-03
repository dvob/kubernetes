use k8s_openapi::api::authorization::v1::*;
use serde::{Deserialize, Serialize};
use std::io::Write;

#[derive(Serialize, Deserialize)]
struct Settings {
    // allow all requests
    allow_all: bool,
    // if user belongs to magic group it can change all resources with name magic_name
    magic_group: Option<String>,
    // name of the resources a member of the magic group can change
    magic_name: Option<String>,
}

impl Default for Settings {
    fn default() -> Self {
        Self {
            allow_all: true,
            magic_group: None,
            magic_name: None
        }
    }
}

#[derive(Serialize, Deserialize)]
struct Request {
    request: SubjectAccessReview,
    settings: Option<Settings>,
}

#[derive(Serialize, Deserialize)]
struct Response {
    response: SubjectAccessReview,
    error: Option<String>,
}

// why does this signature produce a functino which takes a parameter?
//fn auth() -> Result<(), Box<dyn std::error::Error>> {
#[no_mangle]
fn authz() {
    let req: Request = serde_json::from_reader(std::io::stdin()).unwrap();
    let sar = req.request;
    let settings = req.settings.unwrap_or_default();

    let mut response = SubjectAccessReview::default();
    response.metadata.uid = sar.metadata.uid;


    // if the user is a member of the magic_group he is allowd to change all resources with name magic_name
    let mut allowed = match (settings.magic_group, settings.magic_name) {
        (Some(group), Some(name)) => {
            if !sar.spec.groups.unwrap_or_default().contains(&group) {
                false
            } else {
            if let Some(res) = sar.spec.resource_attributes {
                if res.name.unwrap_or_default() == name {
                    true
                } else {
                    false
                }
            } else {
                false
            }
            }
        }
        _ => false,
    };

    if settings.allow_all {
        allowed = true
    }

    let status = SubjectAccessReviewStatus {
        allowed: allowed,
        denied: None,
        evaluation_error: None,
        reason: None,
    };

    response.status = Some(status);

    let resp = Response {
        response,
        error: None,
    };

    serde_json::to_writer(std::io::stdout(), &resp).unwrap();
    std::io::stdout().flush().unwrap();
}
