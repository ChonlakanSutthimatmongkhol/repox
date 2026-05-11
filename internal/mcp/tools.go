package mcp

import "github.com/mark3labs/mcp-go/mcp"

func toolScan() mcp.Tool {
	return mcp.NewTool("repox_scan",
		mcp.WithDescription("Scan the current repository and detect project conventions (type, folder structure, nested feature flows, naming, state management, routing, imports). Saves conventions to .repox/conventions.json."),
		mcp.WithString("project_override",
			mcp.Description("Override auto-detected project type (flutter, go, node)"),
		),
	)
}

func toolGenerate() mcp.Tool {
	return mcp.NewTool("repox_generate",
		mcp.WithDescription("Generate a feature scaffold matching the repo's conventions. Returns list of generated files."),
		mcp.WithString("feature_name",
			mcp.Required(),
			mcp.Description("Name or nested path of the feature to generate (e.g. watchlist, investment/fund_list)"),
		),
		mcp.WithBoolean("use_examples",
			mcp.Description("Find and use similar existing features as reference"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Overwrite existing files"),
		),
		mcp.WithBoolean("dry_run",
			mcp.Description("Preview files without writing"),
		),
		mcp.WithString("pattern",
			mcp.Description("Override feature pattern (flat, grouped, clean_architecture)"),
		),
	)
}

func toolFindSimilar() mcp.Tool {
	return mcp.NewTool("repox_find_similar",
		mcp.WithDescription("Find existing features in the repo that are structurally similar to the target feature name."),
		mcp.WithString("feature_name",
			mcp.Required(),
			mcp.Description("Target feature name to find similar features for"),
		),
		mcp.WithNumber("top_n",
			mcp.Description("Number of similar features to return (default: 3)"),
		),
	)
}

func toolLearn() mcp.Tool {
	return mcp.NewTool("repox_learn",
		mcp.WithDescription("Compare the latest generated scaffold with current files, extract lessons from developer edits."),
		mcp.WithString("generation_id",
			mcp.Description("Specific generation ID to learn from. Defaults to latest."),
		),
		mcp.WithBoolean("auto_approve",
			mcp.Description("Automatically approve all extracted lessons"),
		),
	)
}

func toolExplainConvention() mcp.Tool {
	return mcp.NewTool("repox_explain_convention",
		mcp.WithDescription("Explain the detected conventions of the current repository in natural language."),
	)
}

func toolSetup() mcp.Tool {
	return mcp.NewTool("repox_setup",
		mcp.WithDescription("Idempotently initialize Repox, scan conventions, and generate project skill instructions."),
	)
}

func toolDoctor() mcp.Tool {
	return mcp.NewTool("repox_doctor",
		mcp.WithDescription("Diagnose whether the current repository is ready to use Repox and return suggested fixes."),
	)
}

func toolMap() mcp.Tool {
	return mcp.NewTool("repox_map",
		mcp.WithDescription("Generate Repox project/convention maps and return generated file paths."),
		mcp.WithString("feature",
			mcp.Description("Optional feature name/path for a focused feature map"),
		),
	)
}

func toolExplain() mcp.Tool {
	return mcp.NewTool("repox_explain",
		mcp.WithDescription("Return an AI-friendly convention explanation."),
		mcp.WithString("feature",
			mcp.Description("Optional feature name/path to explain"),
		),
		mcp.WithString("role",
			mcp.Description("Optional role to explain"),
		),
	)
}
