// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

use std::env;
use std::sync::{Arc, Mutex};

use axum::body::Bytes;
use axum::extract::State;
use axum::http::{HeaderMap, StatusCode, header};
use axum::response::{IntoResponse, Response};
use axum::routing::get;
use axum::{Json, Router};
use serde::{Deserialize, Serialize};
use tower_http::cors::CorsLayer;

type Tasks = Arc<Mutex<Vec<Task>>>;

#[derive(Deserialize)]
struct AddTaskRequest {
    task: AddTask,
}

#[derive(Deserialize)]
struct AddTask {
    title: String,
}

#[derive(Clone, Serialize)]
struct Task {
    title: String,
}

#[derive(Serialize)]
struct TaskList {
    tasks: Vec<Task>,
}

#[derive(Serialize)]
struct TaskResult {
    error: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    message: Option<String>,
}

#[derive(Serialize)]
struct CreateTaskResponse {
    result: TaskResult,
    #[serde(skip_serializing_if = "Vec::is_empty")]
    tasks: Vec<Task>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let port = env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let tasks = Arc::new(Mutex::new(Vec::new()));
    let app = Router::new()
        .route("/", get(list_tasks).post(add_task))
        .layer(CorsLayer::permissive())
        .with_state(tasks);

    let listener = tokio::net::TcpListener::bind(format!("0.0.0.0:{port}")).await?;
    println!("listening on http://localhost:{port}");
    axum::serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        if let Err(err) = tokio::signal::ctrl_c().await {
            eprintln!("failed to listen for ctrl-c signal: {err}");
        }
    };

    #[cfg(unix)]
    let terminate = async {
        match tokio::signal::unix::signal(tokio::signal::unix::SignalKind::terminate()) {
            Ok(mut signal) => {
                signal.recv().await;
            }
            Err(err) => eprintln!("failed to listen for terminate signal: {err}"),
        }
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
}

async fn list_tasks(State(tasks): State<Tasks>) -> Json<TaskList> {
    let tasks = tasks.lock().expect("tasks mutex poisoned");
    Json(TaskList {
        tasks: tasks.clone(),
    })
}

async fn add_task(State(tasks): State<Tasks>, headers: HeaderMap, body: Bytes) -> Response {
    let title = if is_json(&headers) {
        match serde_json::from_slice::<AddTaskRequest>(&body) {
            Ok(req) => req.task.title.trim().to_string(),
            Err(_) => String::new(),
        }
    } else {
        String::from_utf8_lossy(&body).trim().to_string()
    };

    if title.is_empty() {
        return (
            StatusCode::BAD_REQUEST,
            Json(CreateTaskResponse {
                result: TaskResult {
                    error: true,
                    message: Some("Task title is required.".to_string()),
                },
                tasks: Vec::new(),
            }),
        )
            .into_response();
    }

    let mut tasks = tasks.lock().expect("tasks mutex poisoned");
    tasks.push(Task { title });
    (
        StatusCode::CREATED,
        Json(CreateTaskResponse {
            result: TaskResult {
                error: false,
                message: None,
            },
            tasks: tasks.clone(),
        }),
    )
        .into_response()
}

fn is_json(headers: &HeaderMap) -> bool {
    headers
        .get(header::CONTENT_TYPE)
        .and_then(|value| value.to_str().ok())
        .is_some_and(|value| value.contains("application/json"))
}
