package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/KaioH3/SearchEng/engine"
)

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	ID      json.RawMessage `json:"id,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any         `json:"result,omitempty"`
	Error   *jsonrpcError `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	engine *engine.Engine
}

func NewServer(eng *engine.Engine) *Server {
	return &Server{engine: eng}
}

func (s *Server) Run(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	// Allow up to 1MB lines
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &jsonrpcError{Code: -32700, Message: "parse error"},
			}
			writeResponse(out, resp)
			continue
		}

		resp := s.handleRequest(req)
		if resp == nil {
			// Notification — no response needed
			continue
		}
		writeResponse(out, *resp)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req jsonrpcRequest) *jsonrpcResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil // notification, no response
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonrpcError{Code: -32601, Message: "method not found: " + req.Method},
		}
	}
}

func (s *Server) handleInitialize(req jsonrpcRequest) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "searcheng",
				"version": "0.1.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req jsonrpcRequest) *jsonrpcResponse {
	tools := []map[string]any{
		{
			"name":        "search",
			"description": "Search the web using multiple search engines (DuckDuckGo, Bing, Brave, Google). Returns ranked results with snippets and trust signals.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
					"max_results": map[string]any{
						"type":        "integer",
						"description": "Maximum results to return (default: 5, max: 20)",
					},
					"safe_search": map[string]any{
						"type":        "boolean",
						"description": "Enable NSFW content filtering (default: true)",
					},
				},
				"required": []string{"query"},
			},
		},
	}

	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]any{"tools": tools},
	}
}

func (s *Server) handleToolsCall(req jsonrpcRequest) *jsonrpcResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonrpcError{Code: -32602, Message: "invalid params"},
		}
	}

	switch params.Name {
	case "search":
		return s.callSearch(req.ID, params.Arguments)
	default:
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonrpcError{Code: -32602, Message: "unknown tool: " + params.Name},
		}
	}
}

func (s *Server) callSearch(id json.RawMessage, args json.RawMessage) *jsonrpcResponse {
	var input struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
		SafeSearch *bool  `json:"safe_search"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &jsonrpcError{Code: -32602, Message: "invalid arguments: " + err.Error()},
		}
	}

	if input.Query == "" {
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &jsonrpcError{Code: -32602, Message: "query is required"},
		}
	}

	maxResults := 5
	if input.MaxResults > 0 {
		maxResults = input.MaxResults
		if maxResults > 20 {
			maxResults = 20
		}
	}

	var opts []engine.SearchOptions
	if input.SafeSearch != nil {
		opts = append(opts, engine.SearchOptions{SafeSearch: input.SafeSearch})
	}

	resp := s.engine.Search(context.Background(), input.Query, 1, opts...)

	results := resp.Results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	var sb strings.Builder
	if resp.Answer != "" {
		sb.WriteString(fmt.Sprintf("Answer: %s\n\n", resp.Answer))
	}

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n", i+1, r.Title))
		sb.WriteString(fmt.Sprintf("    URL: %s\n", r.URL))
		sb.WriteString(fmt.Sprintf("    Source: %s | Score: %.2f\n", r.Source, r.Score))
		if r.Trust != nil {
			var tags []string
			if r.Trust.IsHTTPS {
				tags = append(tags, "HTTPS")
			}
			if r.Trust.IsTrusted {
				tags = append(tags, "Trusted")
			}
			if len(tags) > 0 {
				sb.WriteString(fmt.Sprintf("    Trust: [%s]\n", strings.Join(tags, ", ")))
			}
		}
		if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("    %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	if len(resp.Claims) > 0 {
		sb.WriteString("Claims:\n")
		for _, c := range resp.Claims {
			sb.WriteString(fmt.Sprintf("  - %s (sources: %d, confidence: %.1f)\n", c.Text, c.Corroboration, c.Confidence))
		}
		sb.WriteString("\n")
	}

	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: []map[string]any{
			{
				"type": "text",
				"text": sb.String(),
			},
		},
	}
}

func writeResponse(w io.Writer, resp jsonrpcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		// Fallback: write a JSON-RPC error about the marshal failure
		fallback := fmt.Sprintf(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal error: %s"}}`, err.Error())
		fmt.Fprintln(w, fallback)
		return
	}
	fmt.Fprintf(w, "%s\n", data)
}
