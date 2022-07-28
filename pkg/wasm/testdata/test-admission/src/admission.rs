use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
pub struct AdmissionReview {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub kind: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub api_version: Option<String>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub request: Option<AdmissionRequest>,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub response: Option<AdmissionResponse>,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
#[serde(default)]
#[serde(rename_all = "camelCase")]
pub struct AdmissionRequest {
    pub uid: String,
    pub kind: GroupVersionKind,
    pub resource: GroupVersionResource,
    pub sub_resource: String,
    pub request_kind: GroupVersionKind,
    pub request_resource: GroupVersionKind,
    pub request_sub_resource: String,
    pub name: String,
    pub namespace: String,
    pub operation: String,
    pub user_info: UserInfo,
    pub object: serde_json::Value,
    pub old_object: serde_json::Value,
    pub dry_run: bool,
    pub options: HashMap<String, serde_json::Value>,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
#[serde(default)]
#[serde(rename_all = "camelCase")]
pub struct AdmissionResponse {
    pub uid: String,
    pub allowed: bool,
    pub patch: Option<serde_json::Value>,
    pub patch_type: Option<String>,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
#[serde(default)]
pub struct GroupVersionKind {
    pub group: String,
    pub version: String,
    pub kind: String,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
#[serde(default)]
pub struct GroupVersionResource {
    pub group: String,
    pub version: String,
    pub kind: String,
}

#[derive(Serialize, Deserialize, Debug, Clone, Default)]
#[serde(default)]
pub struct UserInfo {
    pub username: String,
    pub uid: String,
    pub groups: HashSet<String>,
    pub extra: HashMap<String, serde_json::Value>,
}