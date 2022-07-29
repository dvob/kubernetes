use wapc_guest as wapc;

#[no_mangle]
pub fn wapc_init() {
    wapc::register_function("run", run);
}

fn run(msg: &[u8]) -> wapc::CallResult {
    let input = std::str::from_utf8(msg)?;

    wapc::console_log(&format!("input : '{}'", input));

    let output = input.to_uppercase();

    Ok(output.into())
}
