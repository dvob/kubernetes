use serde::{Deserialize, Serialize};
use std::io::Write;
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
fn mutate() {
    let req: Request = serde_json::from_reader(std::io::stdin()).unwrap();
    let req = req
        .request
        .request
        .expect("admission review does not contain request");

    let mut pod = serde_json::from_value::<Pod>(req.object).expect("failed to get pod");

    let mut annotations = pod.metadata.annotations.clone().unwrap_or_default();
    annotations.insert("puzzle.ch/test-annotation".into(), "foo".into());

    pod.metadata.annotations = Some(annotations);

    let patched_obj = serde_json::to_vec(&pod).expect("failed to create value from changed pod");
    let patched_obj = base64::encode(patched_obj);
    let status = admission::AdmissionResponse {
        uid: req.uid,
        allowed: true,
        patch: Some(patched_obj),
        patch_type: Some("Full".to_string()),
    };

    let mut response = admission::AdmissionReview::default();
    response.kind = Some("AdmissionReview".to_string());
    response.api_version = Some("admission.k8s.io/v1".to_string());
    response.response = Some(status);

    let resp = Response {
        response,
        error: None,
    };

    serde_json::to_writer(std::io::stdout(), &resp).unwrap();
    std::io::stdout().flush().unwrap();
}