package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/zach-snell/jtk/internal/jira"
	mcpserver "github.com/zach-snell/jtk/internal/mcp"
)

var (
	port   int
	noAuth bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Jira MCP Server",
	Long: `Starts the Model Context Protocol (MCP) server for Jira.
By default, this runs on stdio. You can provide a --port flag to
run it using the HTTP Streamable transport.`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	RootCmd.AddCommand(mcpCmd)
	mcpCmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on for HTTP Streamable transport")
	mcpCmd.Flags().BoolVar(&noAuth, "no-auth", false, "Start server without authentication (tools will return auth-required errors when called)")
}

func runServer() {
	var s *mcp.Server

	if noAuth {
		s = mcpserver.NewUnauthenticated()
	} else {
		creds, err := jira.LoadCredentials()
		if err != nil {
			fmt.Fprintf(os.Stderr, "No credentials found. Either:\n")
			fmt.Fprintf(os.Stderr, "  1. Run: jtk auth\n")
			fmt.Fprintf(os.Stderr, "  2. Set JIRA_DOMAIN + JIRA_EMAIL + JIRA_API_TOKEN env vars\n")
			os.Exit(1)
		}

		var initErr error
		s, initErr = mcpserver.NewFromCredentials(creds)
		if initErr != nil {
			fmt.Fprintf(os.Stderr, "Error initializing MCP server: %v\n", initErr)
			os.Exit(1)
		}
	}

	if port != 0 {
		fmt.Printf("Starting Jira MCP Server on :%d (HTTP Streamable)\n", port)
		handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
			return s
		}, &mcp.StreamableHTTPOptions{JSONResponse: false})

		srv := &http.Server{
			Addr:              fmt.Sprintf(":%d", port),
			Handler:           handler,
			ReadHeaderTimeout: 3 * time.Second,
		}

		if err := srv.ListenAndServe(); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	}
}
