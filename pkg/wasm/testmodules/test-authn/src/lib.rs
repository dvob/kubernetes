use std::io::Write;
use k8s_openapi::api::authentication::v1::*;
use serde::{Serialize, Deserialize};

#[derive(Serialize, Deserialize)]
struct Settings {
    token: String,
    uid: String,
    user: String,
    groups: Vec<String>,
}

#[derive(Serialize, Deserialize)]
struct Request {
    request: TokenReview,
    settings: Option<Settings>,
}

#[derive(Serialize, Deserialize)]
struct Response {
    response: TokenReview,
    error: Option<String>,
}

// why does this signature produce a functino which takes a parameter?
//fn auth() -> Result<(), Box<dyn std::error::Error>> {
#[no_mangle]
fn authn() {
    let req: Request = serde_json::from_reader(std::io::stdin()).unwrap();
    let token_review = req.request;
    let token = token_review.spec.token.expect("token missing");

    let mut response = TokenReview::default();
    let mut status = TokenReviewStatus::default();

    // get settings or use default values
    let settings = req.settings.unwrap_or(Settings{
        token: "my-test-token".to_string(),
        uid: "1337".to_string(),
        user: "my-user".to_string(),
        groups: vec!["system:masters".to_string()]
    });

    if token == settings.token {
        status.authenticated = Some(true);
        status.user = Some(UserInfo{
            username: Some(settings.user),
            uid: Some(settings.uid),
            groups: Some(settings.groups),
            extra: None,
        });
    } else {
        status.authenticated = Some(false);
        status.error = Some("invalid token".to_string())
    }

    response.status = Some(status);

    let resp = Response{
        response,
        error: None,
    };

    serde_json::to_writer(std::io::stdout(), &resp).unwrap();
    std::io::stdout().flush().unwrap();
}