// Package qoder_test provides example usage of the qoder SDK.
package qoder_test

import (
	"context"
	"io"
	"log"

	qoder "github.com/futuretea/qoder-cloud-agents-go-sdk"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/agents"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/environments"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/files"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/memorystores"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/qoderhttp"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/sessions"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/skills"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/types"
	"github.com/futuretea/qoder-cloud-agents-go-sdk/vaults"
)

func ExampleNew() {
	client := qoder.New("pt-your-token-here")
	_ = client
}

func ExampleClient_Agents_list() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	result, err := client.Agents().List(ctx, &types.ListParams{Limit: 10})
	if err != nil {
		log.Printf("Failed to list agents: %v", err)
		return
	}
	for _, agent := range result.Data {
		log.Printf("Agent: %s (%s)", agent.Name, agent.ID)
	}
}

func ExampleClient_Agents_create() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	agent, err := client.Agents().Create(ctx,
		agents.NewCreateRequest("doc-agent", "ultimate").
			WithSystem("You are a documentation assistant.").
			WithDescription("Generates API documentation").
			WithTool(agents.Tool{
				Type:         "agent_toolset_20260401",
				EnabledTools: []string{"Read", "Write", "Edit", "Bash"},
			}),
	)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		return
	}
	log.Printf("Created agent: %s", agent.ID)
}

func ExampleClient_Agents_update() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	agent, err := client.Agents().Update(ctx, "agent_xxx",
		agents.NewUpdateRequest(1). // version from current agent
						WithName("doc-agent-v2").
						WithSystem("You are a senior documentation assistant."),
	)
	if err != nil {
		log.Printf("Failed to update agent: %v", err)
		return
	}
	log.Printf("Updated agent: %s (version %d)", agent.ID, agent.Version)
}

func ExampleClient_Environments_create() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	env, err := client.Environments().Create(ctx,
		environments.NewCreateRequest("default-cloud-env").
			WithDescription("Default cloud execution environment").
			WithConfig(environments.EnvConfig{
				Type:       "cloud",
				Networking: environments.Networking{Type: "limited"},
				Packages: environments.Packages{
					Apt: []string{"curl"},
					Pip: []string{"requests"},
				},
			}),
	)
	if err != nil {
		log.Printf("Failed to create environment: %v", err)
		return
	}
	log.Printf("Created environment: %s", env.ID)
}

func ExampleClient_Sessions_create() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	session, err := client.Sessions().Create(ctx,
		sessions.NewCreateRequest("agent_xxx").
			WithEnvironment("env_xxx").
			WithTitle("Generate API docs").
			WithDeltaFlushInterval(100).
			WithResource(sessions.NewResourceFile("file_xxx", "/data/input.md")).
			WithVault("vault_xxx").
			WithMemoryStore("memstore_xxx"),
	)
	if err != nil {
		log.Printf("Failed to create session: %v", err)
		return
	}
	log.Printf("Created session: %s", session.ID)
}

func ExampleClient_Events_send() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	err := client.Events().SendMessage(ctx, "session_xxx",
		"Please analyze the repository and propose a refactor plan.",
	)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
		return
	}
	log.Println("Message sent successfully")
}

func ExampleClient_Events_stream() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	resp, err := client.Events().Stream(ctx, "session_xxx")
	if err != nil {
		log.Printf("Failed to open stream: %v", err)
		return
	}
	stream := qoderhttp.NewSSEStream(resp)
	defer func() { _ = stream.Close() }()

	for {
		evt, err := stream.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}
		_ = evt // process event
	}
}

func ExampleClient_Files_upload() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	file, err := client.Files().Upload(ctx, &files.UploadFileRequest{
		Filename: "README.md",
		Data:     []byte("# Project README\n"),
		Purpose:  "user_upload",
	})
	if err != nil {
		log.Printf("Failed to upload file: %v", err)
		return
	}
	log.Printf("Uploaded file: %s (%d bytes)", file.ID, file.Size)
}

func ExampleClient_Files_getContent() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	url, err := client.Files().GetContent(ctx, "file_xxx")
	if err != nil {
		log.Printf("Failed to get content URL: %v", err)
		return
	}
	log.Printf("Download URL: %s", url)
}

func ExampleClient_Vaults_create() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	vault, err := client.Vaults().Create(ctx,
		vaults.NewCreateRequest("prod-mcp-credentials").
			WithCredential(vaults.CreateCredential{
				MCPServerURL: "https://mcp.example.com/mcp",
				Protocol:     "streamable_http",
				Type:         "static_bearer",
				AccessToken:  "secret-token",
			}),
	)
	if err != nil {
		log.Printf("Failed to create vault: %v", err)
		return
	}
	log.Printf("Created vault: %s", vault.ID)
}

func ExampleClient_Vaults_addCredential() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	cred := vaults.NewStaticBearerCredential("https://mcp2.example.com/mcp", "streamable_http", "another-secret")
	credential, err := client.Vaults().CreateCredential(ctx, "vault_xxx", &cred)
	if err != nil {
		log.Printf("Failed to add credential: %v", err)
		return
	}
	log.Printf("Added credential: %s", credential.ID)
}

func ExampleClient_Skills_upload() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	skill, err := client.Skills().Create(ctx, &skills.CreateSkillRequest{
		Filename: "my-skill.zip",
		Data:     []byte("..."), // .zip file content
		Type:     "custom",
	})
	if err != nil {
		log.Printf("Failed to create skill: %v", err)
		return
	}
	log.Printf("Created skill: %s (v%d)", skill.ID, skill.Version)
}

func ExampleClient_Skills_get() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	skill, err := client.Skills().Get(ctx, "skill_xxx", true) // include content
	if err != nil {
		log.Printf("Failed to get skill: %v", err)
		return
	}
	log.Printf("Skill: %s (%s)", skill.Name, skill.Description)
}

func ExampleClient_MemoryStores_manage() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	// Create a memory store
	store, err := client.MemoryStores().Create(ctx,
		memorystores.NewCreateStoreRequest("project-memory").
			WithDescription("Persistent project notes"),
	)
	if err != nil {
		log.Printf("Failed to create store: %v", err)
		return
	}

	// Create a memory entry
	entry, err := client.MemoryStores().CreateEntry(ctx, store.ID,
		memorystores.NewCreateEntryRequest("notes/arch.md",
			"The service uses event streaming for session updates."),
	)
	if err != nil {
		log.Printf("Failed to create entry: %v", err)
		return
	}
	log.Printf("Created entry: %s (v%d)", entry.ID, entry.Version)

	// Update the entry
	updated, err := client.MemoryStores().UpdateEntry(ctx, store.ID, entry.ID,
		memorystores.NewUpdateEntryRequest("Updated architecture notes...").
			WithContentSHA256(entry.ContentSHA256),
	)
	if err != nil {
		log.Printf("Failed to update entry: %v", err)
		return
	}
	log.Printf("Updated entry: %s (v%d)", updated.ID, updated.Version)
}

func ExampleClient_Models_list() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	models, err := client.Models().List(ctx)
	if err != nil {
		log.Printf("Failed to list models: %v", err)
		return
	}
	for _, m := range models {
		if m.IsEnabled {
			log.Printf("Model: %s (%s)", m.ID, m.DisplayName)
		}
	}
}

func ExampleClient_MemoryStores_versions() {
	client := qoder.New("pt-your-token-here")
	ctx := context.Background()

	versions, err := client.MemoryStores().ListVersions(ctx, "memstore_xxx", nil)
	if err != nil {
		log.Printf("Failed to list versions: %v", err)
		return
	}
	for _, v := range versions.Data {
		log.Printf("Version %d: %s (redacted: %v)", v.Version, v.Action, v.Redacted)
	}
}
