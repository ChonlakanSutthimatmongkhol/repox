package mcp

import "github.com/mark3labs/mcp-go/mcp"

func toolScan() mcp.Tool {
	return mcp.NewTool("repox_scan",
		mcp.WithDescription("Scan the current repository and detect project conventions (type, folder structure, naming, state management, routing, imports). Saves conventions to .repox/conventions.json."),
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
			mcp.Description("Name of the feature to generate (e.g. watchlist, fund-profile)"),
		),
		mcp.WithBoolean("use_ai",
			mcp.Description("Use AI to generate code based on conventions and examples"),
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
