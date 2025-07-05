# Claude-specific OpenAPI-MCP: Dockerized MCP Server to allow your AI agent to access any API with existing api docs

**Generate MCP tool definitions directly from a Swagger/OpenAPI specification file.**

OpenAPI-MCP is a dockerized MCP server that reads a `swagger.json` or `openapi.yaml` file and generates a corresponding [Model Context Protocol (MCP)](https://modelcontextprotocol.io/introduction) toolset. This allows MCP-compatible clients like [Cursor](https://cursor.sh/) to interact with APIs described by standard OpenAPI specifications. Now you can enable your AI agent to access any API by simply providing its OpenAPI/Swagger specification - no additional coding required.

## Table of Contents

-   [Why OpenAPI-MCP?](#why-openapi-mcp)
-   [Features](#features)
-   [Installation](#installation)
    -   [Using the Pre-built Docker Hub Image (Recommended)](#using-the-pre-built-docker-hub-image-recommended)
    -   [Building Locally (Optional)](#building-locally-optional)
-   [Running the Weatherbit Example (Step-by-Step)](#running-the-weatherbit-example-step-by-step)
-   [Command-Line Options](#command-line-options)
    -   [Environment Variables](#environment-variables)

## Why OpenAPI-MCP?

-   **Standard Compliance:** Leverage your existing OpenAPI/Swagger documentation.
-   **Automatic Tool Generation:** Create MCP tools without manual configuration for each endpoint.
-   **Flexible API Key Handling:** Securely manage API key authentication for the proxied API without exposing keys to the MCP client.
-   **Local & Remote Specs:** Works with local specification files or remote URLs.
-   **Dockerized Tool:** Easily deploy and run as a containerized service with Docker.

## Features

-   **OpenAPI v2 (Swagger) & v3 Support:** Parses standard specification formats.
-   **Schema Generation:** Creates MCP tool schemas from OpenAPI operation parameters and request/response definitions.
-   **Secure API Key Management:**
    -   Injects API keys into requests (`header`, `query`, `path`, `cookie`) based on command-line configuration.
        -   Loads API keys directly from flags (`--api-key`), environment variables (`--api-key-env`), or `.env` files located alongside local specs.
        -   Keeps API keys hidden from the end MCP client (e.g., the AI assistant).
-   **Server URL Detection:** Uses server URLs from the spec as the base for tool interactions (can be overridden).
-   **Filtering:** Options to include/exclude specific operations or tags (`--include-tag`, `--exclude-tag`, `--include-op`, `--exclude-op`).
-   **Request Header Injection:** Pass custom headers (e.g., for additional auth, tracing) via the `REQUEST_HEADERS` environment variable.

## Installation

### Docker

The recommended way to run this tool is via Docker.

#### Building Locally

1.  **Build the Docker Image Locally:**
    ```bash
    # Navigate to the repository root
    cd openapi-mcp-claude
    # Build the Docker image (tag it as you like, e.g., openapi-mcp-claude:latest)
    docker build -t openapi-mcp-claude:latest .
    ```

2.  **Run the Container:**
    You need to provide the OpenAPI specification and any necessary API key configuration when running the container.

    *   **Example 1: Using a local spec file and `.env` file:**
        -   Create a directory (e.g., `./my-api`) containing your `openapi.json` or `swagger.yaml`.
        -   If the API requires a key, create a `.env` file in the *same directory* (e.g., `./my-api/.env`) with `API_KEY=your_actual_key` (replace `API_KEY` if your `--api-key-env` flag is different).
        ```bash
        docker run -p 8080:8080 --rm \
            -v ./my-api:/app/spec \
            --env-file ./my-api/.env \
            openapi-mcp-claude:latest \
            --spec /app/spec/openapi.json \
            --base-url https://example.com \
            --api-key-env API_KEY \
            --api-key-name X-Api-Key \
            --api-key-loc header
        ```
        *(Adjust `--spec`, `--api-key-env`, `--api-key-name`, `--api-key-loc`, and `-p` as needed.)*

    *   **Example 2: Using a remote spec URL and direct environment variable:**
        ```bash
        docker run -p 8080:8080 --rm \
            -e SOME_API_KEY="your_actual_key" \
            openapi-mcp-claude:latest \
            --spec https://petstore.swagger.io/v2/swagger.json \
            --api-key-env SOME_API_KEY \\
            --api-key-name api_key \
            --api-key-loc header
        ```

    *   **Key Docker Run Options:**
        *   `-p <host_port>:8080`: Map a port on your host to the container's default port 8080.
        *   `--rm`: Automatically remove the container when it exits.
        *   `-v <host_path>:<container_path>`: Mount a local directory containing your spec into the container. Use absolute paths or `$(pwd)/...`. Common container path: `/app/spec`.
        *   `--env-file <path_to_host_env_file>`: Load environment variables from a local file (for API keys, etc.). Path is on the host.
        *   `-e <VAR_NAME>="<value>"`: Pass a single environment variable directly.
        *   `openapi-mcp:latest`: The name of the image you built locally.
        *   `--spec ...`: **Required.** Path to the spec file *inside the container* (e.g., `/app/spec/openapi.json`) or a public URL.
        *   `--port 8080`: (Optional) Change the internal port the server listens on (must match the container port in `-p`).
        *   `--api-key-env`, `--api-key-name`, `--api-key-loc`: Required if the target API needs an API key.
        *   (See `--help` for all command-line options by running `docker run --rm openapi-mcp:latest --help`)

### Configuring Claude Code

Edit your `~/.claude.json` to include the following section:

```json
"mcpServers": {
    "openapi-mcp": {
        "type": "http",
        "url": "http://localhost:8080/messages",
        "headers": {
            "Mcp-Session-Id": "YOUR_GUID_GOES_HERE"
        }
    }
}
```

### Configuring Claude Desktop

In Claude Desktop you will need to use the mcp-remote@latest npx or docker image. Follow instructions for installation of mcp-remote@latest globally in your environment and then set up your `claude_desktop_config.json` to resemble the following:

```json
"mcpServers": {
    "openWebui": {
        "command": "npx",
        "args": [
            "mcp-remote@latest",
            "http://localhost:8080/messages",
            "--allow-http",
            "--header",
            "Mcp-Session-Id:YOUR_GUID_GOES_HERE"
        ]
    }
}
```

## Command-Line Options

The `openapi-mcp` command accepts the following flags:

| Flag                 | Description                                                                                                         | Type          | Default                          |
|----------------------|---------------------------------------------------------------------------------------------------------------------|---------------|----------------------------------|
| `--spec`             | **Required.** Path or URL to the OpenAPI specification file.                                                          | `string`      | (none)                           |
| `--port`             | Port to run the MCP server on.                                                                                      | `int`         | `8080`                           |
| `--api-key`          | Direct API key value (use `--api-key-env` or `.env` file instead for security).                                       | `string`      | (none)                           |
| `--api-key-env`      | Environment variable name containing the API key. If spec is local, also checks `.env` file in the spec's directory. | `string`      | (none)                           |
| `--api-key-name`     | **Required if key used.** Name of the API key parameter (header, query, path, or cookie name).                       | `string`      | (none)                           |
| `--api-key-loc`      | **Required if key used.** Location of API key: `header`, `query`, `path`, or `cookie`.                              | `string`      | (none)                           |
| `--include-tag`      | Tag to include (can be repeated). If include flags are used, only included items are exposed.                       | `string slice`| (none)                           |
| `--exclude-tag`      | Tag to exclude (can be repeated). Exclusions apply after inclusions.                                                | `string slice`| (none)                           |
| `--include-op`       | Operation ID to include (can be repeated).                                                                          | `string slice`| (none)                           |
| `--exclude-op`       | Operation ID to exclude (can be repeated).                                                                          | `string slice`| (none)                           |
| `--base-url`         | Manually override the target API server base URL detected from the spec.                                              | `string`      | (none)                           |
| `--name`             | Default name for the generated MCP toolset (used if spec has no title).                                             | `string`      | "OpenAPI-MCP Tools"            |
| `--desc`             | Default description for the generated MCP toolset (used if spec has no description).                                | `string`      | "Tools generated from OpenAPI spec" |
| `--state-file-path`  | Path to the state file used to track sessions. | `string` | "/tmp/openapi-conn-state.yaml" |

**Note:** You can get this list by running the tool with the `--help` flag (e.g., `docker run --rm openapi-mcp-claude:latest --help`).

### Environment Variables

*   `REQUEST_HEADERS`: Set this environment variable to a JSON string (e.g., `'{"X-Custom": "Value"}'`) to add custom headers to *all* outgoing requests to the target API.
