use std::io::Write;
use serde::{Serialize, Deserialize};
//use kube::core::admission::*;
use k8s_openapi::api::core::v1::Pod;

mod admission;

#[derive(Serialize, Deserialize)]
struct Request {
    request: admission::AdmissionReview,
}

#[derive(Serialize, Deserialize)]
struct Response {
    response: admission::AdmissionReview,
    error: Option<String>,
}

#[no_mangle]
fn validate() {
    let req: Request = serde_json::from_reader(std::io::stdin()).unwrap();
    let req = req.request.request.expect("admission review does not contain request");

    let pod = serde_json::from_value::<Pod>(req.object).expect("failed to get pod");

    let mut status = admission::AdmissionResponse{
        uid: req.uid,
        allowed: true,
        patch: None,
        patch_type: None, 
    };

    if pod.metadata.namespace.expect("no namespace") == "default" && pod.metadata.name.expect("no name") != "allowed-pod-name" {
        status.allowed = false;
    }

    let mut response = admission::AdmissionReview::default();
    response.kind = Some("AdmissionReview".to_string());
    response.api_version = Some("admission.k8s.io/v1".to_string());
    response.response = Some(status);

    let resp = Response{
        response,
        error: None,
    };

    serde_json::to_writer(std::io::stdout(), &resp).unwrap();
    std::io::stdout().flush().unwrap();
}