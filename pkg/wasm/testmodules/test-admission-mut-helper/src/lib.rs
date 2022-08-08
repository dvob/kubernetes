use std::collections::BTreeMap;

use k8s_openapi::api::core::v1::Pod;
use k8s_wasi::admission::AdmissionReview;
use k8s_wasi::Admiter;
use serde::Deserialize;

#[derive(Deserialize)]
struct Settings {
    annotations: BTreeMap<String, String>,
}

struct MyMutator {}

impl Admiter<Settings> for MyMutator {
    fn admit(
        ar: AdmissionReview,
        settings: Settings,
    ) -> Result<AdmissionReview, Box<dyn std::error::Error>> {
        let mut request = ar.get_request()?;

        // verify / mutate request
        let mut pod: Pod = request.get_object()?;
        pod.metadata
            .annotations
            .get_or_insert_with(Default::default)
            .extend(settings.annotations);

        AdmissionReview::mutate(request.uid, pod)
    }
}

k8s_wasi::register_admiter!(MyMutator);
