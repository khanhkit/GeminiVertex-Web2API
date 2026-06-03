# Gemini-Web2API (Go Version)

Convert the Google Gemini Web version into API formats compatible with OpenAI, Claude, and Gemini.

## Features

- **OpenAI Compatible**: `/v1/chat/completions`, `/v1/models`, `/v1/images/generations`
- **Claude Compatible**: `/v1/messages`, `/v1/messages/count_tokens`
- **Gemini Native Protocol**: `/v1beta/models/{model}:generateContent`, `:streamGenerateContent`
- **Streaming Output**: SSE (Server-Sent Events) with a typewriter effect
- **Thinking Process**: Support for extracting the model's thinking process (`reasoning_content`)
- **Image Generation**: Support for Nano Banana / Nano Banana Pro image generation
- **Image Upload**: Support for multimodal image inputs
- **Multi-Account Load Balancing**: Support for configuring multiple Google accounts
- **HTTP Proxy**: Support for global proxy and independent proxy per account (HTTP/SOCKS5)
- **Model Mapping**: Support for mapping Claude/OpenAI model names to Gemini models
- **403 Auto-Retry**: Automatically re-initialize and retry when cookies expire

## Supported Models

| Model Name | Description |
|------------|-------------|
| `gemini-2.5-flash` | Fast model |
| `gemini-3.1-pro-preview` | Pro Preview version |
| `gemini-3-flash-preview` | Flash Preview version |
| `gemini-3-flash-preview-no-thinking` | Flash model without thinking mode |
| `gemini-2.5-flash-image` | Nano Banana image generation |
| `gemini-3-pro-image-preview` | Nano Banana Pro image generation |

## Quick Start

### 1. Run
```bash
# Build
go build -o Gemini-Web2API.exe ./cmd/server

# Run
./Gemini-Web2API.exe
```

### 2. Configure Cookies

**Method 1: Automatic Retrieval (Firefox)**

The application automatically reads Google cookies from Firefox (default account).

**Method 2: Batch Retrieval via Chrome (Recommended)**
```bash
# 1. Close Chrome browser
# 2. Run the command
./Gemini-Web2API.exe --fetch-cookies

# 3. Select profile (Enter 1, 2, 3 or ALL)
```
See [internal/browser/README.en.md](internal/browser/README.en.md) for details.
Note: This method might not work for newer versions of Chrome, which applies to most Chrome users.

**Method 3: Manual Configuration**
```bash
cp .env.example .env
# Edit .env and enter the cookies
```

Multi-account configuration (with suffixes):
```
__Secure-1PSID_Account1=xxx
__Secure-1PSIDTS_Account1=yyy
__Secure-1PSID_Account2=xxx
__Secure-1PSIDTS_Account2=yyy
```

### 3. Model Mapping (Optional)
Map external model names to Gemini models:
```
MODEL_MAPPING=claude-haiku-4-5-20251001:gemini-3-flash-preview-no-thinking
```

## API Endpoints

### OpenAI Compatible
```
POST /v1/chat/completions
POST /v1/images/generations
GET  /v1/models
```

### Claude Compatible
```
POST /v1/messages
POST /v1/messages/count_tokens
GET  /v1/models/claude
```

### Gemini Native Protocol
```
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent
GET  /v1beta/models
```
Authentication is supported via three methods: `Authorization: ****** `?key=YOUR_API_KEY`, and `x-goog-api-key`.

## Usage Examples

### Chat
```bash
curl http://127.0.0.1:8007/v1/chat/completions \
  -H "Authorization: ******" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-3-flash-preview",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

### Image Generation
```bash
curl http://127.0.0.1:8007/v1/images/generations \
  -H "Authorization: ******" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash-image",
    "prompt": "a cat wearing a hat",
    "n": 1,
    "size": "1024x1024",
    "response_format": "b64_json"
  }'
```
Or use directly on the `v1/chat/completions` endpoint, and the reply will be automatically formatted as `![Generated Image 1](data:image/png;base64,xxx)`.

## Directory Structure

```
cmd/server/         # Application entry point
internal/
  adapter/          # OpenAI/Claude/Gemini protocol adapters
  balancer/         # Multi-account load balancing
  browser/          # Cookie retrieval
  claude/           # Claude protocol types
  config/           # Configuration (model mapping)
  gemini/           # Gemini Web API client
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Service port | 8007 |
| `PROXY_API_KEY` | API Key | (Empty = no authentication) |
| `PROXY` | Global proxy (http/socks5) | (Empty) |
| `PROXY_{id}` | Single account proxy, overrides global | (Empty) |
| `MODEL_MAPPING` | Model mapping | (Empty) |
| `LANGUAGE` | Language (Accept-Language / payload) | en |
| `SNAPSHOT_STREAMING` | Enable snapshot streaming (experimental) | 0 |

## Note

Not intended for production-grade security. Issues and Pull Requests are welcome.
