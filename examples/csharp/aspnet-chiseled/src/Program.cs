// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

using System.Text.Json.Serialization;

var builder = WebApplication.CreateSlimBuilder(args);

builder.Services.ConfigureHttpJsonOptions(options =>
{
    options.SerializerOptions.TypeInfoResolverChain.Insert(0, AppJsonSerializerContext.Default);
});

builder.Services.AddCors(options =>
{
    options.AddDefaultPolicy(policy => policy.AllowAnyOrigin().AllowAnyMethod().AllowAnyHeader());
});

var tasks = new List<TaskItem>();
var app = builder.Build();

app.UseCors();

app.MapGet("/", () => new TaskListResponse(tasks));
app.MapPost("/", CreateTask);

app.Run();

IResult CreateTask(CreateTaskRequest request)
{
    var title = request.Task?.Title.Trim() ?? "";
    if (title == "")
    {
        return Results.Json(
            new CreateTaskResponse(new TaskResult(true, "Task title is required."), []),
            AppJsonSerializerContext.Default.CreateTaskResponse,
            statusCode: StatusCodes.Status400BadRequest
        );
    }

    tasks.Add(new TaskItem(title));
    return Results.Json(
        new CreateTaskResponse(new TaskResult(false), tasks),
        AppJsonSerializerContext.Default.CreateTaskResponse,
        statusCode: StatusCodes.Status201Created
    );
}

public sealed record TaskItem(string Title);
public sealed record CreateTaskRequest(TaskItem? Task);
public sealed record TaskResult(bool Error, [property: JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? Message = null);
public sealed record TaskListResponse(IReadOnlyList<TaskItem> Tasks);
public sealed record CreateTaskResponse(TaskResult Result, IReadOnlyList<TaskItem> Tasks);

[JsonSerializable(typeof(TaskItem))]
[JsonSerializable(typeof(CreateTaskRequest))]
[JsonSerializable(typeof(TaskResult))]
[JsonSerializable(typeof(TaskListResponse))]
[JsonSerializable(typeof(CreateTaskResponse))]
internal partial class AppJsonSerializerContext : JsonSerializerContext;
