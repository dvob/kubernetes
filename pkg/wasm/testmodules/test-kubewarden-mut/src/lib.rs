
use guest::prelude::*;
use kubewarden_policy_sdk::wapc_guest as guest;

use k8s_openapi::api::core::v1 as apicore;

extern crate kubewarden_policy_sdk as kubewarden;
use kubewarden::{request::ValidationRequest, validate_settings};
use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize, Default, Debug)]
#[serde(default)]
pub(crate) struct Settings {}

impl kubewarden::settings::Validatable for Settings {
    fn validate(&self) -> Result<(), String> {
        Ok(())
    }
}

#[no_mangle]
pub extern "C" fn wapc_init() {
    register_function("validate", validate);
    register_function("validate_settings", validate_settings::<Settings>);
}

fn validate(payload: &[u8]) -> CallResult {
    let validation_request: ValidationRequest<Settings> = ValidationRequest::new(payload)?;

    let mut pod = serde_json::from_value::<apicore::Pod>(validation_request.request.object)?;

    let mut annotations = pod.metadata.annotations.clone().unwrap_or_default();
    annotations.insert("puzzle.ch/test-annotation".into(), "foo".into());

    pod.metadata.annotations = Some(annotations);

    let new_obj = serde_json::to_value(pod)?;
    kubewarden::mutate_request(new_obj)
}