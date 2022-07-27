use k8s_openapi::api::authorization::v1::*;
use serde::{Deserialize, Serialize};
use std::io::Write;

#[derive(Serialize, Deserialize)]
struct Request {
    request: SubjectAccessReview,
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

    let mut response = SubjectAccessReview::default();
    response.metadata.uid = sar.metadata.uid;

    let status = SubjectAccessReviewStatus{
        allowed: true,
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

